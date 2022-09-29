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
	"net/http"
	"qr-mixer-game/qrpb"
	"strings"
	"testing"
	"time"

	"google.golang.org/protobuf/encoding/prototext"
	"google.golang.org/protobuf/proto"
)

// setupSynthetic sets up the global variables and the game and survey questions.
// As global variables are being set here, package tests cannot be run parallely.
func setupSynthetic(cgo *CachedGameOptions, numPlayers int) error {
	var qrset qrpb.QRMappingSet
	var gOptions qrpb.SurveySet
	var sqs qrpb.GameQSet

	// set all the qr mappings
	qrset.QrMappings = make([]*qrpb.QRMapping, 0)
	for i := 0; i < numPlayers; i++ {
		qrp := qrpb.QRMapping{
			Username:    proto.String(fmt.Sprintf("username-%v", (i + 1))),
			Qrcode:      proto.String(fmt.Sprintf("qrcode-%v", (i + 1))),
			DisplayName: proto.String(fmt.Sprintf("name-%v", (i + 1))),
			CardSuit:    qrpb.CardSuit((i % 4) + 1).Enum(),
			CardRank:    proto.Int64(int64((i % 13) + 1)),
		}
		qrset.QrMappings = append(qrset.QrMappings, &qrp)
	}
	var qrmap QRMappings
	qrmap.mappings = &qrset
	qrmap.RefreshMappings()

	// set up the survey questions
	gOptions.SurveyQuestions = make([]*qrpb.SurveyQuestion, 0)
	for i := 0; i < 2; i++ {
		var dq qrpb.SurveyQuestion
		dq.QuestionId = proto.Int64(int64(i + 1))
		dq.QuestionText = proto.String(fmt.Sprintf("dq-%v", (i + 1)))
		dq.Type = qrpb.SurveyType_BOOLEAN.Enum()
		gOptions.SurveyQuestions = append(gOptions.SurveyQuestions, &dq)
	}

	// set up the static questions
	sqs.GameQuestions = make([]*qrpb.GameQuestion, 0)
	for i := 0; i < 23; i++ {
		var sq qrpb.GameQuestion
		sq.QuestionId = proto.Int64(int64(i + 1))
		sq.QuestionHtml = proto.String(fmt.Sprintf("qHtml-%v", (i + 1)))
		sq.Type = qrpb.GQType_USERNAME_LIST.Enum()
		sq.AnsUsernames = make([]string, 2)
		sq.AnsUsernames[0] = fmt.Sprintf("username-%v", (i + 1))
		sq.AnsUsernames[1] = fmt.Sprintf("username-%v", (i + 2))
		sqs.GameQuestions = append(sqs.GameQuestions, &sq)
	}

	// Switch Q3 and Q5 to survey
	sqs.GameQuestions[2].Type = qrpb.GQType_SURVEY_ANS.Enum()
	sqs.GameQuestions[2].SurveyId = proto.Int64(1)
	sqs.GameQuestions[2].SurveyTrueIsCorrect = proto.Bool(true)
	sqs.GameQuestions[4].Type = qrpb.GQType_SURVEY_ANS.Enum()
	sqs.GameQuestions[4].SurveyId = proto.Int64(2)
	sqs.GameQuestions[4].SurveyTrueIsCorrect = proto.Bool(false)

	// Switch Q19 to ANY_PERSON
	sqs.GameQuestions[18].Type = qrpb.GQType_ANY_PERSON.Enum()
	sqs.GameQuestions[18].AnsUsernames = []string{}

	if err := cgo.SetSurveySet(&gOptions); err != nil {
		return err
	}
	if err := cgo.SetGameQSet(&sqs); err != nil {
		return err
	}
	if err := cgo.SetQRMappings(&qrmap); err != nil {
		return err
	}
	return nil
}

func TestStaticAnswering(t *testing.T) {
	env, err := createEnv(":memory:")
	if err != nil {
		t.Fatal(err)
	}
	defer env.db.Close()
	setupSynthetic(env.cgo, 30)

	// log in player 1
	postData := "qr=qrcode-1&dqans1=true&dqans2=false"
	ck1 := http.Cookie{Name: "sid", Value: "cookie-1", Expires: time.Now().Add(24 * 30 * time.Hour)}
	f := callController("POST", "/submitsurvey", postData, &ck1, env.submitSurvey)
	if f.statuscode != 200 {
		t.Fatalf("Expected HTTP 200. got: %v\n%v", f.statuscode, f.resptext)
	}

	// check that the game renders fine
	f = callController("GET", "/game", "", &ck1, env.gameHandler)
	if f.statuscode != 200 {
		t.Fatalf("Expected HTTP 200 on game render. got: %v\n%v", f.statuscode, f.resptext)
	}
	if !strings.Contains(f.resptext, "qHtml-1") {
		t.Fatalf("Expected game render to have question qHtml-1. got: %v", f.resptext)
	}

	// submit the wrong answer
	postData = "answer=qrcode-4"
	f = callController("POST", "/makemove", postData, &ck1, env.makeMove)
	if f.statuscode != 200 {
		t.Fatalf("Expected HTTP 200. got: %v\n%v", f.statuscode, f.resptext)
	}
	var mr MoveResponse
	if err := json.Unmarshal([]byte(f.resptext), &mr); err != nil {
		t.Fatal(err)
	}
	if mr.GameArtifacts["action"] != "Lost a Life!" {
		t.Errorf("expected game action to be Lost a Life!. got: %v", mr.GameArtifacts["action"])
	}
	if *mr.State.Life != (STARTING_LIFE - 1) {
		t.Errorf("did not reduce life. want: %v. got: %v", (STARTING_LIFE - 1), *mr.State.Life)
	}
	if *mr.State.UserLevel != 1 {
		t.Errorf("did not stay in same level. want: 1. got: %v", *mr.State.UserLevel)
	}

	// submit the right answer (both qrcode-1 and qrcode-2 are correct answers)
	postData = "answer=qrcode-2"
	f = callController("POST", "/makemove", postData, &ck1, env.makeMove)
	if f.statuscode != 200 {
		t.Fatalf("Expected HTTP 200. got: %v\n%v", f.statuscode, f.resptext)
	}
	if err := json.Unmarshal([]byte(f.resptext), &mr); err != nil {
		t.Fatal(err)
	}
	if mr.GameArtifacts["action"] != "Correct!" {
		t.Errorf("expected game action to be Correct!. got: %v", mr.GameArtifacts["action"])
	}
	if *mr.State.Life != (STARTING_LIFE - 1) {
		t.Errorf("did not keep same life. want: %v. got: %v", (STARTING_LIFE - 1), *mr.State.Life)
	}
	if *mr.State.UserLevel != 2 {
		t.Errorf("did not move to next level. want: 2. got: %v", *mr.State.UserLevel)
	}
}

func TestSurveyQuestionAnswering(t *testing.T) {
	env, err := createEnv(":memory:")
	if err != nil {
		t.Fatal(err)
	}
	defer env.db.Close()
	setupSynthetic(env.cgo, 10)

	// log in player 1
	postData := "qr=qrcode-1&dqans1=false&dqans2=true"
	ck1 := http.Cookie{Name: "sid", Value: "cookie-1", Expires: time.Now().Add(24 * 30 * time.Hour)}
	callController("POST", "/submitsurvey", postData, &ck1, env.submitSurvey)

	// log in player 7
	postData = "qr=qrcode-7&dqans1=true&dqans2=false"
	ck7 := http.Cookie{Name: "sid", Value: "cookie-7", Expires: time.Now().Add(24 * 30 * time.Hour)}
	callController("POST", "/submitsurvey", postData, &ck7, env.submitSurvey)

	// player 1 answers 2 correct questions
	callController("POST", "/makemove", "answer=qrcode-1", &ck1, env.makeMove)
	callController("POST", "/makemove", "answer=qrcode-2", &ck1, env.makeMove)

	// Q3 is a survey question, player 7 has answered 'true' to that survey
	// submitting qrcode-1 should be wrong answer, because player 1 has answered 'false'
	f := callController("POST", "/makemove", "answer=qrcode-1", &ck1, env.makeMove)
	var mr MoveResponse
	if err := json.Unmarshal([]byte(f.resptext), &mr); err != nil {
		t.Fatal(err)
	}
	if mr.GameArtifacts["action"] != "Lost a Life!" {
		t.Errorf("expected game action to be Lost a Life!. got: %v", mr.GameArtifacts["action"])
	}

	// submitting qrcode-7 should be the right answer, because player 7 has answered 'true'
	f = callController("POST", "/makemove", "answer=qrcode-7", &ck1, env.makeMove)
	if err := json.Unmarshal([]byte(f.resptext), &mr); err != nil {
		t.Fatal(err)
	}
	if mr.GameArtifacts["action"] != "Correct!" {
		t.Errorf("expected game action to be Correct!. got: %v", mr.GameArtifacts["action"])
	}
	if *mr.State.UserLevel != 4 {
		t.Errorf("did not move to next level. want: 4. got: %v", *mr.State.UserLevel)
	}
}

func TestMetalGranting(t *testing.T) {
	env, err := createEnv(":memory:")
	if err != nil {
		t.Fatal(err)
	}
	defer env.db.Close()
	setupSynthetic(env.cgo, 10)

	// log in player 1
	postData := "qr=qrcode-1&dqans1=false&dqans2=true"
	ck1 := http.Cookie{Name: "sid", Value: "cookie-1", Expires: time.Now().Add(24 * 30 * time.Hour)}
	callController("POST", "/submitsurvey", postData, &ck1, env.submitSurvey)

	// log in player 7
	postData = "qr=qrcode-7&dqans1=true&dqans2=false"
	ck7 := http.Cookie{Name: "sid", Value: "cookie-7", Expires: time.Now().Add(24 * 30 * time.Hour)}
	callController("POST", "/submitsurvey", postData, &ck7, env.submitSurvey)

	// player 1 answers 2 correct questions
	callController("POST", "/makemove", "answer=qrcode-1", &ck1, env.makeMove)
	callController("POST", "/makemove", "answer=qrcode-2", &ck1, env.makeMove)
	// SurveyQuestion Q3. submitting qrcode-7 should be the right answer, because player 7 has answered 'true'
	callController("POST", "/makemove", "answer=qrcode-7", &ck1, env.makeMove)
	callController("POST", "/makemove", "answer=qrcode-4", &ck1, env.makeMove)
	// SurveyQuestion Q5. submitting qrcode-7 should be the right answer, because player 7 has answered 'false'
	f := callController("POST", "/makemove", "answer=qrcode-7", &ck1, env.makeMove)
	var mr MoveResponse
	if err := json.Unmarshal([]byte(f.resptext), &mr); err != nil {
		t.Fatal(err)
	}
	if mr.GameArtifacts["action"] != "Correct!" {
		t.Errorf("expected game action to be Correct!. got: %v", mr.GameArtifacts["action"])
	}
	if *mr.State.UserLevel != 6 {
		t.Errorf("did not move to next level. want: 6. got: %v", *mr.State.UserLevel)
	}

	// Continue from Q6 now
	callController("POST", "/makemove", "answer=qrcode-6", &ck1, env.makeMove)
	callController("POST", "/makemove", "answer=qrcode-7", &ck1, env.makeMove)
	callController("POST", "/makemove", "answer=qrcode-8", &ck1, env.makeMove)
	callController("POST", "/makemove", "answer=qrcode-9", &ck1, env.makeMove)
	f = callController("POST", "/makemove", "answer=qrcode-10", &ck1, env.makeMove)

	if err := json.Unmarshal([]byte(f.resptext), &mr); err != nil {
		t.Fatal(err)
	}
	if mr.GameArtifacts["action"] != "Correct!" {
		t.Errorf("expected game action to be Correct!. got: %v", mr.GameArtifacts["action"])
	}
	if *mr.State.UserLevel != 11 {
		t.Errorf("did not move to next level. want: 11. got: %v", *mr.State.UserLevel)
	}
	if !(mr.State.GetHasAl() || mr.State.GetHasCu()) {
		pt, _ := prototext.Marshal(mr.State)
		t.Errorf("Expected to have either Al or Cu. got: %v", string(pt))
	}
	if mr.State.GetHasAl() && mr.State.GetHasCu() {
		pt, _ := prototext.Marshal(mr.State)
		t.Errorf("Expected not to have both Al and Cu. got: %v", string(pt))
	}
}

func Test100Players(t *testing.T) {
	env, err := createEnv(":memory:")
	if err != nil {
		t.Fatal(err)
	}
	defer env.db.Close()
	setupSynthetic(env.cgo, 100)

	cookies := make([]http.Cookie, 100)

	// log in player 1, who has the right answers to the survey questions
	cookies[0] = http.Cookie{Name: "sid", Value: "cookie-1", Expires: time.Now().Add(24 * 30 * time.Hour)}
	callController("POST", "/submitsurvey", "qr=qrcode-1&dqans1=true&dqans2=false", &cookies[0], env.submitSurvey)

	// log in 99 other players
	for i := 1; i < 100; i++ {
		postData := fmt.Sprintf("qr=qrcode-%v&dqans1=false&dqans2=true", (i + 1))
		cookies[i] = http.Cookie{
			Name:    "sid",
			Value:   fmt.Sprintf("cookie-%v", (i + 1)),
			Expires: time.Now().Add(24 * 30 * time.Hour),
		}
		callController("POST", "/submitsurvey", postData, &cookies[i], env.submitSurvey)
	}

	// Everyone answers the first 5 questions, including the 2 survey ones
	for i := 0; i < 100; i++ {
		callController("POST", "/makemove", "answer=qrcode-1", &cookies[i], env.makeMove)
		callController("POST", "/makemove", "answer=qrcode-2", &cookies[i], env.makeMove)
		callController("POST", "/makemove", "answer=qrcode-1", &cookies[i], env.makeMove) // Survey
		callController("POST", "/makemove", "answer=qrcode-4", &cookies[i], env.makeMove)
		f := callController("POST", "/makemove", "answer=qrcode-1", &cookies[i], env.makeMove) // Survey
		var mr MoveResponse
		if err := json.Unmarshal([]byte(f.resptext), &mr); err != nil {
			t.Fatalf("response: %v\nerror:%v", f.resptext, err)
		}
		if mr.GameArtifacts["action"] != "Correct!" {
			t.Errorf("expected game action to be Correct!. got: %v", mr.GameArtifacts["action"])
		}
		if *mr.State.UserLevel != 6 {
			t.Errorf("did not move to next level. want: 6. got: %v", *mr.State.UserLevel)
		}
	}

	// Everyone answers the remaining questions Q6-18
	for i := 0; i < 100; i++ {
		for q := 6; q <= 18; q++ {
			f := callController("POST", "/makemove", fmt.Sprintf("answer=qrcode-%v", q), &cookies[i], env.makeMove)
			if f.statuscode != 200 {
				t.Errorf("expected question submission to succeed. got HTTP %v.\n%v", f.statuscode, f.resptext)
			}
		}
	}

	with_al := -1
	with_cu := -1
	with_sn := -1
	with_zn := -1
	// Everyone answers the wildcard question Q19 and we track a person who picked up each metal
	for i := 0; i < 100; i++ {
		f := callController("POST", "/makemove", "answer=qrcode-9", &cookies[i], env.makeMove)
		if f.statuscode != 200 {
			t.Errorf("expected question submission to succeed. got HTTP %v.\n%v", f.statuscode, f.resptext)
		}
		var mr MoveResponse
		if err := json.Unmarshal([]byte(f.resptext), &mr); err != nil {
			t.Fatalf("response: %v\nerror:%v", f.resptext, err)
		}

		if with_al == -1 {
			if mr.State.GetHasAl() {
				with_al = i
			}
		}
		if with_cu == -1 {
			if mr.State.GetHasCu() {
				with_cu = i
			}
		}
		if with_sn == -1 {
			if mr.State.GetHasSn() {
				with_sn = i
			}
		}
		if with_zn == -1 {
			if mr.State.GetHasZn() {
				with_zn = i
			}
		}
	}

	// Check that there is at least one person with each metal
	if with_al == -1 {
		t.Error("Nobody has al")
	}
	if with_cu == -1 {
		t.Error("Nobody has cu")
	}
	if with_sn == -1 {
		t.Error("Nobody has sn")
	}
	if with_zn == -1 {
		t.Error("Nobody has zn")
	}

	// Everyone collects metal and scans the cauldron (Q20)
	for i := 0; i < 100; i++ {
		callController("POST", "/makemove", fmt.Sprintf("answer=qrcode-%v", (with_al+1)), &cookies[i], env.makeMove)
		callController("POST", "/makemove", fmt.Sprintf("answer=qrcode-%v", (with_cu+1)), &cookies[i], env.makeMove)
		callController("POST", "/makemove", fmt.Sprintf("answer=qrcode-%v", (with_sn+1)), &cookies[i], env.makeMove)
		callController("POST", "/makemove", fmt.Sprintf("answer=qrcode-%v", (with_zn+1)), &cookies[i], env.makeMove)
		f := callController("POST", "/makemove", "answer=qrcode-21", &cookies[i], env.makeMove)
		if f.statuscode != 200 {
			t.Errorf("expected question submission to succeed. got HTTP %v.\n%v", f.statuscode, f.resptext)
		}
		var mr MoveResponse
		if err := json.Unmarshal([]byte(f.resptext), &mr); err != nil {
			t.Fatalf("response: %v\nerror:%v", f.resptext, err)
		}
		if mr.State.GetUserLevel() != 22 {
			t.Errorf("did not win. want: 22. got: %v", *mr.State.UserLevel)
		}
		if mr.GameArtifacts["action"] != "Correct!" {
			t.Errorf("expected game action to be Correct!. got: %v", mr.GameArtifacts["action"])
		}

		// A future scan should always respond with already victorious
		f = callController("POST", "/makemove", "answer=qrcode-21", &cookies[i], env.makeMove)
		json.Unmarshal([]byte(f.resptext), &mr)
		if mr.GameArtifacts["action"] != "Already Victorious!" {
			t.Errorf("expected game action to be Already Victorious!. got: %v", mr.GameArtifacts["action"])
		}
	}
}

func TestRenderWithoutSetup(t *testing.T) {
	env, err := createEnv(":memory:")
	if err != nil {
		t.Fatal(err)
	}
	defer env.db.Close()

	// we don't call setup synthetic. so just pull up the static list and find the first qrcode.
	qrmap, err := env.cgo.GetQRMappings()
	if err != nil {
		t.Fatal(err)
	}
	var qr1 string
	for k := range qrmap.qrcodeToQRMap {
		qr1 = k
		break
	}

	// log in player 1
	postData := fmt.Sprintf("qr=%v&dqans1=true&dqans2=false", qr1)
	ck1 := http.Cookie{Name: "sid", Value: "cookie-1", Expires: time.Now().Add(24 * 30 * time.Hour)}
	f := callController("POST", "/submitsurvey", postData, &ck1, env.submitSurvey)
	if f.statuscode != 200 {
		t.Fatalf("Expected HTTP 200. got: %v\n%v", f.statuscode, f.resptext)
	}

	// check that the game renders fine
	f = callController("GET", "/game", "", &ck1, env.gameHandler)
	if f.statuscode != 200 {
		t.Fatalf("Expected HTTP 200 on game render. got: %v\n%v", f.statuscode, f.resptext)
	}
}
