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
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"net/url"
	"qr-mixer-game/common"
	"qr-mixer-game/qrpb"
	"time"

	"github.com/google/uuid"
	proto "google.golang.org/protobuf/proto"
)

const STARTING_LEVEL int64 = 1
const STARTING_LIFE int64 = 5

// checkRegisteredBadge is the endpoint that checks the QR code scanned on the nocookie.html page
// and if a valid badge is found, it redirects to the confirm name page. It does not set a cookie.
func (env *Env) checkRegisteredBadge(w http.ResponseWriter, r *http.Request) {
	log.Println("Req: ", r.URL)
	// TODO: if cookie already present, ask if a new session should be started
	a := r.FormValue("answer")
	if len(a) == 0 {
		http.NotFound(w, r)
		return
	}

	qrm, err := env.cgo.GetQRMappings()
	if common.Should500(err, w, "Could not fetch the qr code mappings") {
		return
	}

	foundPlayer := qrm.LookupByQrCode(a)
	if foundPlayer == nil {
		m := NewMoveResponse()
		m.GameArtifacts = make(map[string]string, 0)
		m.GameArtifacts["action"] = "Unregistered QR."
		m.PortHTML = "You have to scan your own QR code (which is on your badge). You have to do this again, even if you scanned it once to get to this page."
		js, err := json.Marshal(m)
		if common.Should500(err, w, "could not mash together json") {
			return
		}
		w.Header().Set("Content-Type", "application/json")
		w.Write(js)
		return
	}

	// We found a QR, so we redirect to the confirmname page.
	m := NewMoveResponse()
	m.GameArtifacts = make(map[string]string, 0)
	m.GameArtifacts["redirectUrl"] = "/confirmname?qr=" + url.QueryEscape(a)
	js, err := json.Marshal(m)
	if common.Should500(err, w, "could not mash together json") {
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.Write(js)
}

// confirmName renders the page that lets the user confirm if the badge they scanned is correct
// It does not set a cookie, and redirects to the survey page on success.
func (env *Env) confirmName(w http.ResponseWriter, r *http.Request) {
	log.Println("Req: ", r.URL)
	a := r.FormValue("qr")
	if len(a) == 0 {
		http.NotFound(w, r)
		return
	}
	qrm, err := env.cgo.GetQRMappings()
	if common.Should500(err, w, "Could not fetch the qr code mappings") {
		return
	}

	foundPlayer := qrm.LookupByQrCode(a)
	if foundPlayer == nil {
		common.Should500(fmt.Errorf("that QR code does not have an username mapping"), w, "QR code has no username mapping")
		return
	}

	common.RenderTemplate(w, env.tem, "confirmname.html", struct {
		Username string
		Name     string
		Qr       string
	}{
		Username: foundPlayer.GetUsername(),
		Name:     foundPlayer.GetDisplayName(),
		Qr:       a,
	})
}

// survey renders the page that asks the survey questions. It sets a cookie, but doesn't log it to the db yet.
func (env *Env) survey(w http.ResponseWriter, r *http.Request) {
	log.Println("Req: ", r.URL)
	a := r.FormValue("qr")
	if len(a) == 0 {
		http.NotFound(w, r)
		return
	}

	qrm, err := env.cgo.GetQRMappings()
	if common.Should500(err, w, "Could not fetch the qr code mappings") {
		return
	}

	foundPlayer := qrm.LookupByQrCode(a)
	if foundPlayer == nil {
		common.Should500(fmt.Errorf("that QR code does not have an username mapping"), w, "QR code has no username mapping")
		return
	}

	qrgo, err := env.cgo.GetSurveySet()
	if err != nil {
		common.Should500(err, w, "Could not get the questions")
		return
	}

	expiration := time.Now().Add(cookieValidity)
	cookie := http.Cookie{Name: "sid", Value: uuid.New().String(), Expires: expiration, HttpOnly: true}
	http.SetCookie(w, &cookie)
	common.RenderTemplate(w, env.tem, "survey.html", struct {
		Name, Qr string
		Qrgo     *qrpb.SurveySet
	}{
		Name: foundPlayer.GetDisplayName(),
		Qr:   a,
		Qrgo: qrgo,
	})
}

// submitSurvey stores the results in the db, along with the cookie.
func (env *Env) submitSurvey(w http.ResponseWriter, r *http.Request) {
	log.Println("Req: ", r.URL)
	ck, err := r.Cookie("sid")
	if err != nil {
		common.RespondHTTP401(w)
		return
	}
	if common.Should500(r.ParseForm(), w, "error saving survey answers") {
		return
	}

	a := r.FormValue("qr")
	if len(a) == 0 {
		http.NotFound(w, r)
		return
	}

	qrm, err := env.cgo.GetQRMappings()
	if common.Should500(err, w, "Could not fetch the qr code mappings") {
		return
	}

	foundPlayer := qrm.LookupByQrCode(a)
	if foundPlayer == nil {
		common.Should500(fmt.Errorf("that QR code does not have an username mapping"), w, "QR code has no username mapping")
		return
	}

	qrgo, err := env.cgo.GetSurveySet()
	if err != nil {
		common.Should500(err, w, "Could not get the questions")
		return
	}

	var gu qrpb.GUser
	gu.Username = proto.String(foundPlayer.GetUsername())
	gu.Name = proto.String(foundPlayer.GetDisplayName())
	gu.SurveyAnswers = make([]*qrpb.SurveyAnswer, 0)

	for _, qn := range qrgo.SurveyQuestions {
		id := qn.QuestionId
		ans := r.FormValue(fmt.Sprintf("dqans%v", *id))
		var da qrpb.SurveyAnswer
		da.QuestionId = proto.Int64(*id)
		if len(ans) > 0 {
			da.IsTrue = proto.Bool(ans == "true")
		}
		gu.SurveyAnswers = append(gu.SurveyAnswers, &da)
	}

	// First try to see if this cookie already exists
	var sr *StateRow
	sr, err = GetUserStateByCookie(env.db, ck.Value)
	if common.Should500(err, w, "error fetching user from db") {
		return
	}
	if sr != nil {
		gu.Username = proto.String(sr.UserInfo.GetUsername())
		sr.UserInfo = &gu
		if common.Should500(UpdateUserDetails(env.GetDb(), sr), w, "could not update your survey details") {
			return
		}
		fmt.Fprint(w, "ok")
		return
	}

	// Next, try to see if this username already exists
	sr, err = GetUserStateByUsername(env.db, gu.GetUsername())
	if common.Should500(err, w, "error fetching username from db") {
		return
	}
	if sr != nil {
		sr.Cookie = ck.Value
		if common.Should500(UpdateUserCookie(env.db, sr), w, "could not update your cookie") {
			return
		}
		fmt.Fprint(w, "ok")
		return
	}

	// Now we know this user definitely does not exist. So we add a new entry.
	sr = &StateRow{
		Cookie:   ck.Value,
		Username: gu.GetUsername(),
		UserInfo: &gu,
		State: &qrpb.GameState{
			Life:      proto.Int64(STARTING_LIFE),
			UserLevel: proto.Int64(STARTING_LEVEL),
		},
	}
	if common.Should500(AddUser(env.GetDb(), sr), w, "could not add the user") {
		return
	}

	fmt.Fprint(w, "ok")
}
