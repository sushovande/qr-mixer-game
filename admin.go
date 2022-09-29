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
	"log"
	"net/http"
	"qr-mixer-game/common"
	"qr-mixer-game/qrpb"
	"sort"
	"time"

	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/encoding/prototext"
	proto "google.golang.org/protobuf/proto"
)

// DisplayUser is used to show all users on the leaderboard.
type DisplayUser struct {
	Name          string
	Username      string
	SurveyAnswers []bool
	Level         int64
	Health        int64
	HasAl         bool
	HasCu         bool
	HasSn         bool
	HasZn         bool
}

type ByLevel []DisplayUser

func (a ByLevel) Len() int           { return len(a) }
func (a ByLevel) Less(i, j int) bool { return a[i].Level < a[j].Level }
func (a ByLevel) Swap(i, j int)      { a[i], a[j] = a[j], a[i] }

func (env *Env) adminAllUsers(w http.ResponseWriter, r *http.Request) {
	log.Println("Req: ", r.URL)
	srs, err := AdminGetAllUserStates(env.GetDb())
	if common.Should500(err, w, "could not fetch data for all users") {
		return
	}

	opt, err := env.cgo.GetSurveySet()
	if common.Should500(err, w, "could not get survey question info") {
		return
	}

	numSurveyAns := len(opt.GetSurveyQuestions())

	allU := make([]DisplayUser, 0)
	for _, u := range srs {
		var du DisplayUser
		du.Name = u.UserInfo.GetName()
		du.Username = u.UserInfo.GetUsername()
		du.SurveyAnswers = make([]bool, numSurveyAns)
		du.Level = u.State.GetUserLevel()
		du.Health = u.State.GetLife()

		for _, b := range u.UserInfo.SurveyAnswers {
			if b.GetQuestionId()-1 < int64(numSurveyAns) {
				du.SurveyAnswers[b.GetQuestionId()-1] = b.GetIsTrue()
			}
		}

		du.HasAl = u.State.GetHasAl()
		du.HasCu = u.State.GetHasCu()
		du.HasSn = u.State.GetHasSn()
		du.HasZn = u.State.GetHasZn()

		allU = append(allU, du)
	}

	SurveyQNames := make([]string, numSurveyAns)
	for i := 0; i < numSurveyAns; i++ {
		SurveyQNames[i] = fmt.Sprintf("SQ%v", (i + 1))
	}

	sort.Sort(sort.Reverse(ByLevel(allU)))

	rd := struct {
		SurveyQ []string
		Users   []DisplayUser
	}{
		SurveyQ: SurveyQNames,
		Users:   allU,
	}
	common.RenderTemplate(w, env.tem, "adminallusers.html", rd)

}

func (env *Env) adminAllLogs(w http.ResponseWriter, r *http.Request) {
	log.Println("Req: ", r.URL)
	srs, err := AdminGetAllUserLogs(env.GetDb())
	if common.Should500(err, w, "could not fetch data for all users") {
		return
	}

	type StrUs struct {
		Username string
		Updated  string
		GameLog  string
	}

	logList := make([]StrUs, 0)
	kol, err := time.LoadLocation("Asia/Kolkata")
	if common.Should500(err, w, "could not make a timezone") {
		return
	}

	for _, v := range srs {
		logList = append(logList, StrUs{
			Username: v.Username,
			Updated:  time.Unix(v.Updated/1000000, 0).In(kol).Format("2006-01-02 3:04:05 PM"),
			GameLog:  MarshalTextString(v.GameLog),
		})
	}

	renderData := struct {
		UserState string
		UserInfo  string
		Logs      []StrUs
	}{
		UserState: "",
		UserInfo:  "",
		Logs:      logList,
	}

	common.RenderTemplate(w, env.tem, "adminalllogs.html", renderData)
}

func (env *Env) adminUserLogs(w http.ResponseWriter, r *http.Request) {
	log.Println("Req: ", r.URL)
	toks, err := common.GetURLTokens(r.URL.Path, "/9283e316-beaa-4182-b3a6-0937046251ee")
	if common.Should500(err, w, "could not tokenize your URL") {
		return
	}

	uid, ok := toks["userLogs"]
	if !ok {
		common.Should500(fmt.Errorf("could not parse your username"), w, "could not parse your username")
		return
	}

	sr, err := GetUserStateByUsername(env.GetDb(), uid)
	if common.Should500(err, w, "could not fetch details for this user") {
		return
	}

	srs, err := GetAllLogsForUser(env.GetDb(), uid)
	if common.Should500(err, w, "could not fetch logs for this user") {
		return
	}

	type StrUs struct {
		Username string
		Updated  string
		GameLog  string
	}

	logList := make([]StrUs, 0)
	kol, err := time.LoadLocation("Asia/Kolkata")
	if common.Should500(err, w, "could not make a timezone") {
		return
	}

	for _, v := range srs {
		logList = append(logList, StrUs{
			Username: v.Username,
			Updated:  time.Unix(v.Updated/1000000, 0).In(kol).Format("2006-01-02 3:04:05 PM"),
			GameLog:  MarshalTextString(v.GameLog),
		})
	}

	renderData := struct {
		UserState string
		UserInfo  string
		Logs      []StrUs
	}{
		UserState: prototext.Format(sr.State),
		UserInfo:  prototext.Format(sr.UserInfo),
		Logs:      logList,
	}

	common.RenderTemplate(w, env.tem, "adminalllogs.html", renderData)
}

func (env *Env) adminRenderQuestions(w http.ResponseWriter, r *http.Request) {
	log.Println("Req: ", r.URL)

	surveyq, err := env.cgo.GetSurveySet()
	if err != nil {
		common.Should500(err, w, "could not read survey")
	}

	gqset, err := env.cgo.GetGameQSet()
	if err != nil {
		common.Should500(err, w, "could not read static")
	}

	qns := struct {
		SurveyQuestions string
		GameQuestions   string
	}{
		SurveyQuestions: prototext.Format(surveyq),
		GameQuestions:   prototext.Format(gqset),
	}

	common.RenderTemplate(w, env.tem, "adminquestions.html", qns)
}

func (env *Env) adminSaveQuestions(w http.ResponseWriter, r *http.Request) {
	log.Println("Req: ", r.URL)
	if common.Should500(r.ParseForm(), w, "error saving survey answers") {
		return
	}

	surveyq := r.FormValue("survey")
	if len(surveyq) > 0 {
		var sset qrpb.SurveySet
		if common.Should500(prototext.Unmarshal([]byte(surveyq), &sset), w, "proto parse error survey") {
			return
		}
		if common.Should500(env.cgo.SetSurveySet(&sset), w, "error saving survey qn") {
			return
		}
	}

	gqsetfv := r.FormValue("gameq")
	if len(gqsetfv) > 0 {
		var gqset qrpb.GameQSet
		if common.Should500(prototext.Unmarshal([]byte(gqsetfv), &gqset), w, "proto parse error static qn") {
			return
		}
		if common.Should500(env.cgo.SetGameQSet(&gqset), w, "error saving static qn") {
			return
		}
	}

	fmt.Fprint(w, "ok")
}

func (env *Env) adminUpdateUser(w http.ResponseWriter, r *http.Request) {
	log.Println("Req: ", r.URL)
	if common.Should500(r.ParseForm(), w, "error parsing data for update user") {
		return
	}

	var userinfo qrpb.GUser
	var userstate qrpb.GameState
	userinfotxt := r.FormValue("userinfo")
	userstatetxt := r.FormValue("userstate")
	if len(userinfotxt) > 0 && len(userstatetxt) > 0 {
		if common.Should500(prototext.Unmarshal([]byte(userinfotxt), &userinfo), w, "proto parse error user info") {
			return
		}
		if common.Should500(prototext.Unmarshal([]byte(userstatetxt), &userstate), w, "proto parse error user state") {
			return
		}
		if common.Should500(UpdateUserDetailsWithProto(env.GetDb(), &userinfo, &userstate), w, "error saving data") {
			return
		}
	}

	fmt.Fprint(w, "ok")
}

func (env *Env) adminRenderManagerUsers(w http.ResponseWriter, r *http.Request) {
	log.Println("Req: ", r.URL)

	userTSV := "Name\tUsername\tqrcode\tcardsuit\tcardrank\n"
	qrm, err := env.cgo.GetQRMappings()
	if common.Should500(err, w, "could not get list of users") {
		return
	}

	for _, k := range qrm.mappings.GetQrMappings() {
		userTSV += fmt.Sprintf("%v\t%v\t%v\t%v\t%v\n",
			k.GetDisplayName(),
			k.GetUsername(),
			k.GetQrcode(),
			k.GetCardSuit(),
			k.GetCardRank())
	}

	uss := struct {
		UserTSV string
	}{
		UserTSV: userTSV,
	}

	common.RenderTemplate(w, env.tem, "adminmanageusers.html", uss)
}

func (env *Env) adminSaveUserQrMapping(w http.ResponseWriter, r *http.Request) {
	log.Println("Req: ", r.URL)
	if common.Should500(r.ParseForm(), w, "error parsing data for save user qr mapping") {
		return
	}

	usertxt := r.FormValue("users")

	if len(usertxt) == 0 {
		common.Should500(fmt.Errorf("empty user data received"), w, "empty user data received")
	}

	var qrset qrpb.QRMappingSet
	if common.Should500(protojson.Unmarshal([]byte(usertxt), &qrset), w, "could not parse user data") {
		return
	}

	if common.Should500(env.cgo.SetQRMappings(&QRMappings{mappings: &qrset}), w, "could not store user set") {
		return
	}

	fmt.Fprint(w, "ok")
}

func (env *Env) adminRenderPrintBadges(w http.ResponseWriter, r *http.Request) {
	log.Println("Req: ", r.URL)

	qrm, err := env.cgo.GetQRMappings()
	if common.Should500(err, w, "could not get list of users") {
		return
	}

	common.RenderTemplate(w, env.tem, "adminprintbadges.html", qrm.mappings)
}

func MarshalTextString(m proto.Message) string {
	b, err := prototext.Marshal(m)
	if err != nil {
		return err.Error()
	}
	return string(b)
}
