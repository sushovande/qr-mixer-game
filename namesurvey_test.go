// Copyright 2022 Google LLC
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package main

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"
)

type savedHTTPResponse struct {
	statuscode int
	resptext   string
	cookie     *http.Cookie
}

func setupGlobals(cgo *CachedGameOptions) {
	setupSynthetic(cgo, 30)
}

// callController is a helper method that calls the given controller function
// with the http verb and parameters.
func callController(verb string, path string, body string, cookie *http.Cookie,
	f func(http.ResponseWriter, *http.Request)) savedHTTPResponse {
	var r savedHTTPResponse
	a := strings.NewReader(body)
	req := httptest.NewRequest(verb, path, a)
	if cookie != nil {
		req.AddCookie(cookie)
	}
	if body != "" {
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	}
	resp := httptest.NewRecorder()
	f(resp, req)
	r.statuscode = resp.Result().StatusCode
	r.resptext = resp.Body.String()
	for _, ck := range resp.Result().Cookies() {
		if ck.Name == "sid" {
			r.cookie = ck
		}
	}

	return r
}

func TestHomepage(t *testing.T) {
	env, err := createEnv(":memory:")
	if err != nil {
		t.Fatal(err)
	}
	defer env.db.Close()

	f := callController("GET", "/", "", nil, env.handler)
	if len(f.resptext) == 0 {
		t.Errorf("expected non-empty response on homepage")
	}
}

func TestBadgingIn(t *testing.T) {
	env, err := createEnv(":memory:")
	if err != nil {
		t.Fatal(err)
	}
	defer env.db.Close()
	setupGlobals(env.cgo)

	postData := fmt.Sprintf("answer=%v",
		url.QueryEscape("qrcode-1"))

	f := callController("POST", "/checkregisteredbadge", postData, nil, env.checkRegisteredBadge)
	if len(f.resptext) == 0 {
		t.Errorf("expected non-empty response on homepage")
	}
	if !strings.Contains(f.resptext, "/confirmname?qr=") {
		t.Errorf("expected response json to contain the redirect to /confirmname. got \n%v", f.resptext)
	}
}

func TestConfirmName(t *testing.T) {
	env, err := createEnv(":memory:")
	if err != nil {
		t.Fatal(err)
	}
	defer env.db.Close()
	setupGlobals(env.cgo)

	f := callController("GET", "/confirmname?qr=qrcode-1", "", nil, env.confirmName)
	if f.statuscode != 200 {
		t.Errorf("expected OK http code. Got %v", f.statuscode)
	}
	if len(f.resptext) == 0 {
		t.Errorf("expected non-empty response on homepage")
	}
}

func TestConfirmNameWrongQr(t *testing.T) {
	env, err := createEnv(":memory:")
	if err != nil {
		t.Fatal(err)
	}
	defer env.db.Close()
	setupGlobals(env.cgo)

	f := callController("GET", "/confirmname?qr=https%3A%2F%2Fqr.sd3.in%2F%23badbadbad", "", nil, env.confirmName)
	if f.statuscode != 500 {
		t.Errorf("expected server error http code. Got %v", f.statuscode)
	}
	if !strings.Contains(f.resptext, "QR code has no username mapping") {
		t.Errorf("wrong error msg. got: %v", f.resptext)
	}
}

func TestSurveyRender(t *testing.T) {
	env, err := createEnv(":memory:")
	if err != nil {
		t.Fatal(err)
	}
	defer env.db.Close()
	setupGlobals(env.cgo)

	f := callController("GET", "/survey?qr=qrcode-1", "", nil, env.survey)
	if f.statuscode != 200 {
		t.Errorf("expected OK http code. Got %v", f.statuscode)
	}
	if len(f.resptext) == 0 {
		t.Errorf("expected non-empty response on homepage")
	}
	if !strings.Contains(f.resptext, "name-1") {
		t.Errorf("expected page contents to have the name of the person. Got %v", f.resptext)
	}
	if f.cookie == nil {
		t.Fatalf("expected a cookie to be set. it wasn't")
	}
	if f.cookie.Name != "sid" {
		t.Errorf("expected to see a cookie with name sid. got %v", f.cookie.Name)
	}
}

func TestSurveySubmit(t *testing.T) {
	env, err := createEnv(":memory:")
	if err != nil {
		t.Fatal(err)
	}
	defer env.db.Close()
	setupGlobals(env.cgo)

	postData := fmt.Sprintf("qr=%v&dqans1=true&dqans2=false",
		url.QueryEscape("qrcode-1"))
	var ck http.Cookie
	ck.Name = "sid"
	ck.Value = "foo-foo"
	ck.Expires = time.Now().Add(24 * 30 * time.Hour)

	f := callController("POST", "/submitsurvey", postData, &ck, env.submitSurvey)

	if f.statuscode != 200 {
		t.Fatalf("Expected HTTP 200. got: %v\n%v", f.statuscode, f.resptext)
	}
	if len(f.resptext) == 0 {
		t.Fatalf("empty response")
	}

	sr, err := GetUserStateByCookie(env.db, "foo-foo")
	if err != nil {
		t.Fatal(err)
	}
	if len(sr.UserInfo.SurveyAnswers) != 2 {
		t.Errorf("unexpected number of survey answers. want: 2. got: %v", len(sr.UserInfo.SurveyAnswers))
	}
	if !*sr.UserInfo.SurveyAnswers[0].IsTrue {
		t.Error("expected first survey ans to be true")
	}
	if *sr.UserInfo.SurveyAnswers[1].IsTrue {
		t.Error("expected second survey ans to be false")
	}
}

func TestRescanQr(t *testing.T) {
	env, err := createEnv(":memory:")
	if err != nil {
		t.Fatal(err)
	}
	defer env.db.Close()
	setupGlobals(env.cgo)

	// first scan
	postData := fmt.Sprintf("qr=%v&dqans1=true&dqans2=false",
		url.QueryEscape("qrcode-1"))
	var ck http.Cookie
	ck.Name = "sid"
	ck.Value = "foo-foo"
	ck.Expires = time.Now().Add(24 * 30 * time.Hour)
	callController("POST", "/submitsurvey", postData, &ck, env.submitSurvey)

	// user scans again on a fresh phone / browser
	postData = fmt.Sprintf("qr=%v&dqans1=false&dqans2=true",
		url.QueryEscape("qrcode-1"))
	ck.Name = "sid"
	ck.Value = "foo2-foo2"
	ck.Expires = time.Now().Add(24 * 30 * time.Hour)
	f := callController("POST", "/submitsurvey", postData, &ck, env.submitSurvey)

	if f.statuscode != 200 {
		t.Fatalf("Expected HTTP 200. got: %v\n%v", f.statuscode, f.resptext)
	}
	if len(f.resptext) == 0 {
		t.Fatalf("empty response")
	}

	sr, _ := GetUserStateByCookie(env.db, "foo-foo")
	if sr != nil {
		t.Errorf("expected old cookie to be overwritten. got %v", sr)
	}

	sr, err = GetUserStateByCookie(env.db, "foo2-foo2")
	if err != nil {
		t.Fatal(err)
	}
	if len(sr.UserInfo.SurveyAnswers) != 2 {
		t.Errorf("unexpected number of survey answers. want: 2. got: %v", len(sr.UserInfo.SurveyAnswers))
	}
	if !*sr.UserInfo.SurveyAnswers[0].IsTrue {
		t.Error("on second scan, the survey answers from the first scan should be preserved")
	}
}
