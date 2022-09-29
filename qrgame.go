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
	"database/sql"
	"encoding/json"
	"flag"
	"fmt"
	"html/template"
	"io/ioutil"
	"log"
	"math/rand"
	"net/http"
	"path"
	"qr-mixer-game/common"
	"qr-mixer-game/qrpb"
	"time"

	proto "google.golang.org/protobuf/proto"
)

var (
	templateDir    = "templates/"
	cookieValidity = 24 * 30 * time.Hour
)

// Env holds information about connections and templates that's shared across requests
type Env struct {
	db  *sql.DB
	tem *template.Template
	cgo *CachedGameOptions
}

func createEnv(dbPath string) (*Env, error) {
	dbConn, err := DbInit(dbPath)
	if err != nil {
		return nil, err
	}
	return &Env{
		db:  dbConn,
		tem: loadAllTemplateFiles(),
		cgo: CreateCachedGameOptions(dbConn),
	}, nil
}

func main() {
	env, err := createEnv("datastore.db")
	if err != nil {
		log.Panic(err)
		return
	}
	defer env.db.Close()

	rand.Seed(time.Now().UnixNano())
	fs := http.FileServer(http.Dir("static"))
	http.Handle("/static/", http.StripPrefix("/static/", denyDirectoryListings(fs)))
	http.Handle("/favicon.ico", fs)

	http.HandleFunc("/", env.handler)
	http.HandleFunc("/checkregisteredbadge", env.checkRegisteredBadge)
	http.HandleFunc("/confirmname", env.confirmName)
	http.HandleFunc("/survey", env.survey)
	http.HandleFunc("/submitsurvey", env.submitSurvey)
	http.HandleFunc("/game", env.gameHandler)  // frontend
	http.HandleFunc("/makemove", env.makeMove) // backend
	http.HandleFunc("/logout", env.logout)
	http.HandleFunc("/9283e316-beaa-4182-b3a6-0937046251ee/allUsers", env.adminAllUsers)
	http.HandleFunc("/9283e316-beaa-4182-b3a6-0937046251ee/allLogs", env.adminAllLogs)
	http.HandleFunc("/9283e316-beaa-4182-b3a6-0937046251ee/userLogs/", env.adminUserLogs)
	http.HandleFunc("/9283e316-beaa-4182-b3a6-0937046251ee/questions", env.adminRenderQuestions)
	http.HandleFunc("/9283e316-beaa-4182-b3a6-0937046251ee/saveQuestions", env.adminSaveQuestions)
	http.HandleFunc("/9283e316-beaa-4182-b3a6-0937046251ee/updateUser", env.adminUpdateUser)
	http.HandleFunc("/9283e316-beaa-4182-b3a6-0937046251ee/manageUsers", env.adminRenderManagerUsers)
	http.HandleFunc("/9283e316-beaa-4182-b3a6-0937046251ee/saveUserQrMapping", env.adminSaveUserQrMapping)
	http.HandleFunc("/9283e316-beaa-4182-b3a6-0937046251ee/card/", env.adminGetCard)
	http.HandleFunc("/9283e316-beaa-4182-b3a6-0937046251ee/printBadges", env.adminRenderPrintBadges)
	http.HandleFunc("/9283e316-beaa-4182-b3a6-0937046251ee/qrimage", env.adminGetQrImage)

	flagPort := flag.String("port", "8080", "what port to listen at")
	flag.Parse()
	strPort := fmt.Sprintf(":%v", *flagPort)

	log.Printf("ready (port: %v).\n", *flagPort)
	log.Fatal(http.ListenAndServe(strPort, nil))
}

// GetDb returns the persistent db connection
func (env *Env) GetDb() *sql.DB {
	return env.db
}

// GetTem returns the template collection
func (env *Env) GetTem() *template.Template {
	return env.tem
}

func (env *Env) handler(w http.ResponseWriter, r *http.Request) {
	log.Println("Req: ", r.URL)
	ck, err := r.Cookie("sid")
	if err != nil {
		common.RenderTemplate(w, env.tem, "nocookie.html", nil)
		return
	}

	u, err := GetUserStateByCookie(env.db, ck.Value)
	if err != nil || u == nil {
		common.RenderTemplate(w, env.tem, "nocookie.html", nil)
		return
	}

	http.Redirect(w, r, "/game", http.StatusTemporaryRedirect)
}

func (env *Env) gameHandler(w http.ResponseWriter, r *http.Request) {
	log.Println("Req: ", r.URL)
	ck, err := r.Cookie("sid")
	if err != nil {
		common.RenderTemplate(w, env.tem, "nocookie.html", nil)
		return
	}
	u, err := GetUserStateByCookie(env.db, ck.Value)
	if err != nil || u == nil {
		if err != nil {
			fmt.Printf("db error: %v\n", err)
		}
		common.RenderTemplate(w, env.tem, "nocookie.html", nil)
		return
	}
	sqs, err := env.cgo.GetGameQSet()
	if err != nil {
		common.Should500(err, w, "There was a problem figuring out the questions, maybe try again?")
		return
	}

	qn := GetQuestionByIndex(sqs, *u.State.UserLevel)
	qnht := qn.GetQuestionHtml()

	renderData := struct {
		U    *StateRow
		Clue template.HTML
	}{
		u,
		template.HTML(qnht),
	}

	common.RenderTemplate(w, env.tem, "hascookie.html", renderData)
}

func (env *Env) makeMove(w http.ResponseWriter, r *http.Request) {
	log.Println("Req: ", r.URL)
	ck, err := r.Cookie("sid")
	if err != nil {
		common.RespondHTTP401(w)
		return
	}

	u, err := GetUserStateByCookie(env.db, ck.Value)
	if err != nil {
		common.RespondHTTP401(w)
		return
	}

	a := r.FormValue("answer")
	if len(a) == 0 {
		http.NotFound(w, r)
		return
	}

	// do logic and respond
	mr := NewMoveResponse()
	stepResult, err := env.Step(u.State, a)
	if err != nil {
		common.Should500(err, w, fmt.Sprintf("You scanned someone unexpected: %v", err.Error()))
		return
	}

	lr := NewLogRow()
	lr.Username = u.Username
	lr.Updated = time.Now().UnixNano() / 1000
	lr.GameLog = &qrpb.ActionLog{
		OldState:      u.State,
		TimestampUsec: proto.Int64(lr.Updated),
		ClueShortName: proto.String(stepResult.scannedClue),
		Result:        &stepResult.actionResult,
		Type:          qrpb.ActionLog_ACTION_CODE_SCAN.Enum(),
	}

	u.State = stepResult.newState

	if common.Should500(UpdateUserDetails(env.GetDb(), u), w, "could not record your action, please try again") {
		return
	}
	if common.Should500(AddActionLog(env.GetDb(), &lr), w, "could not log your action, refresh the page") {
		return
	}
	sqs, err := env.cgo.GetGameQSet()
	if err != nil {
		common.Should500(err, w, "There was a problem figuring out the questions, maybe try again?")
		return
	}

	mr.GameArtifacts = make(map[string]string, 0)
	mr.GameArtifacts["action"] = stepResult.actionString
	mr.State = u.State
	mr.PortHTML = GetQuestionByIndex(sqs, *u.State.UserLevel).GetQuestionHtml()
	js, err := json.Marshal(mr)
	if common.Should500(err, w, "error encoding json") {
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.Write(js)
}

func (env *Env) logout(w http.ResponseWriter, r *http.Request) {
	log.Println("Req: ", r.URL)
	ck, err := r.Cookie("sid")
	w.Header().Set("Content-Type", "text/html")
	if err != nil {
		expiration := time.Now().Add(-24 * time.Hour)
		cookie := http.Cookie{Name: "sid", Value: "abcd", Expires: expiration, MaxAge: -1, HttpOnly: true}
		http.SetCookie(w, &cookie)
		fmt.Fprintf(w, "Weird, you were already logged out. <a href=\"/\">Start Again?</a>")
		return
	}

	u, err := GetUserStateByCookie(env.db, ck.Value)
	if err != nil || u == nil {
		expiration := time.Now().Add(-24 * time.Hour)
		cookie := http.Cookie{Name: "sid", Value: "abcd", Expires: expiration, MaxAge: -1, HttpOnly: true}
		http.SetCookie(w, &cookie)
		fmt.Fprintf(w, "Weird, you were already logged out. <a href=\"/\">Start Again?</a>")
		return
	}
	expiration := time.Now().Add(-24 * time.Hour)
	cookie := http.Cookie{Name: "sid", Value: "abcd", Expires: expiration, MaxAge: -1, HttpOnly: true}
	http.SetCookie(w, &cookie)

	fmt.Fprintf(w, "We've logged you out. <a href=\"/\">Start Again?</a>")
	DeleteCookieEntry(env.db, u)
}

func (env *Env) serve404(w http.ResponseWriter, r *http.Request) {
	http.NotFound(w, r)
}

func denyDirectoryListings(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "" {
			http.NotFound(w, r)
			return
		}

		next.ServeHTTP(w, r)
	})
}

func loadAllTemplateFiles() *template.Template {
	f, err := ioutil.ReadDir(templateDir)
	if err != nil {
		panic(err)
	}
	fnames := make([]string, len(f))
	for i, v := range f {
		fnames[i] = path.Join(templateDir, v.Name())
	}
	t, err := template.ParseFiles(fnames...)
	if err != nil {
		panic(err)
	}
	return t
}
