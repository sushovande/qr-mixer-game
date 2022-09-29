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
	"fmt"
	"log"
	"qr-mixer-game/qrpb"

	"google.golang.org/protobuf/proto"
)

//go:generate protoc --go_out=. gamedata.proto

// StateRow represents a row from the db about a user of this site
type StateRow struct {
	// Cookie is the login token stored on the browser.
	Cookie string
	// Username is the username of the user, used to lookup the survey answers
	Username string
	// UserInfo has info about this particular user
	UserInfo *qrpb.GUser
	// State is the current state of this user's game
	State *qrpb.GameState
}

func NewStateRow() StateRow {
	var sr StateRow
	sr.UserInfo = &qrpb.GUser{}
	sr.State = &qrpb.GameState{}
	return sr
}

type nullableStateRow struct {
	Cookie   sql.NullString
	Username sql.NullString
	UserInfo sql.RawBytes
	State    sql.RawBytes
}

// LogRow describes a single action recorded by a user
type LogRow struct {
	Username string
	Updated  int64
	GameLog  *qrpb.ActionLog
}

func NewLogRow() LogRow {
	var lr LogRow
	lr.GameLog = &qrpb.ActionLog{}
	return lr
}

type nullableLogRow struct {
	Username sql.NullString
	Updated  sql.NullInt64
	GameLog  sql.RawBytes
}

// MoveResponse encapsulates the json response in response to a player making a move.
type MoveResponse struct {
	GameArtifacts map[string]string
	State         *qrpb.GameState
	PortHTML      string
}

func NewMoveResponse() MoveResponse {
	var mr MoveResponse
	mr.State = &qrpb.GameState{}
	return mr
}

// StepResponse is the result of doing one action in the game
type StepResponse struct {
	newState     *qrpb.GameState
	actionString string
	levelClue    string
	scannedClue  string
	actionResult qrpb.ActionLog_ActionResult
}

func NewStepResponse() StepResponse {
	var sr StepResponse
	sr.newState = &qrpb.GameState{}
	return sr
}

func (nsr nullableStateRow) toStateRow() (*StateRow, error) {
	sr := NewStateRow()
	if nsr.Cookie.Valid {
		sr.Cookie = nsr.Cookie.String
	}
	if nsr.Username.Valid {
		sr.Username = nsr.Username.String
	}
	if len(nsr.UserInfo) > 0 {
		if err := proto.Unmarshal(nsr.UserInfo, sr.UserInfo); err != nil {
			return nil, err
		}
	}
	if len(nsr.State) > 0 {
		if err := proto.Unmarshal(nsr.State, sr.State); err != nil {
			return nil, err
		}
	}
	return &sr, nil
}

func (nlr nullableLogRow) toLogRow() (*LogRow, error) {
	lr := NewLogRow()
	if nlr.Username.Valid {
		lr.Username = nlr.Username.String
	}
	if nlr.Updated.Valid {
		lr.Updated = nlr.Updated.Int64
	}
	if len(nlr.GameLog) > 0 {
		if err := proto.Unmarshal(nlr.GameLog, lr.GameLog); err != nil {
			return nil, err
		}
	}
	return &lr, nil
}

// MaybeCreateUserTable creates the user table in the db if it didn't exist
func MaybeCreateUserTable(db *sql.DB) error {
	const createStmt = `
	CREATE TABLE IF NOT EXISTS userstate (
		cookie TEXT PRIMARY KEY,
		username TEXT,
		userinfo BLOB,
		state BLOB
	);
	CREATE TABLE IF NOT EXISTS gamelogs (
		username TEXT,
		updated INT,
		gamelog BLOB
	);
	`
	_, err := db.Exec(createStmt)
	if err != nil {
		log.Printf("Db-err: %v\n", err)
		return err
	}
	return nil
}

// GetUserStateByCookie returns the row corresponding to the user with the given cookie.
func GetUserStateByCookie(db *sql.DB, cookie string) (*StateRow, error) {
	const getStmt = `SELECT cookie, username, userinfo, state FROM userstate WHERE cookie=?`
	rows, err := db.Query(getStmt, cookie)
	if err != nil {
		log.Printf("Db-err: %v\n", err)
		return nil, err
	}
	defer rows.Close()
	if rows.Next() {
		var s nullableStateRow
		if err = rows.Scan(&s.Cookie, &s.Username, &s.UserInfo, &s.State); err != nil {
			return nil, err
		}
		sr, err := s.toStateRow()
		if err != nil {
			return nil, err
		}
		return sr, nil
	}
	return nil, nil
}

// GetUserStateByUsername returns the row corresponding to the user with the given username.
func GetUserStateByUsername(db *sql.DB, username string) (*StateRow, error) {
	const getStmt = `SELECT cookie, username, userinfo, state FROM userstate WHERE username=?`
	rows, err := db.Query(getStmt, username)
	if err != nil {
		log.Printf("Db-err: %v\n", err)
		return nil, err
	}
	defer rows.Close()
	if rows.Next() {
		var s nullableStateRow
		if err = rows.Scan(&s.Cookie, &s.Username, &s.UserInfo, &s.State); err != nil {
			return nil, err
		}
		sr, err := s.toStateRow()
		if err != nil {
			return nil, err
		}
		return sr, nil
	}
	return nil, nil
}

func GetUserInfoByUsername(db *sql.DB, username string) (*qrpb.GUser, error) {
	const getStmt = `SELECT userinfo FROM userstate WHERE username=?`
	rows, err := db.Query(getStmt, username)
	if err != nil {
		log.Printf("Db-err: %v\n", err)
		return nil, err
	}
	defer rows.Close()
	if rows.Next() {
		var s nullableStateRow
		if err = rows.Scan(&s.UserInfo); err != nil {
			return nil, err
		}
		sr, err := s.toStateRow()
		if err != nil {
			return nil, err
		}
		return sr.UserInfo, nil
	}
	return nil, nil
}

// AddUser adds a StateRow to the db
func AddUser(db *sql.DB, sr *StateRow) error {
	const insData = `INSERT INTO userstate VALUES(?,?,?,?)`
	userinfo, err := proto.Marshal(sr.UserInfo)
	if err != nil {
		return err
	}
	state, err := proto.Marshal(sr.State)
	if err != nil {
		return err
	}
	_, err = db.Exec(insData, sr.Cookie, sr.UserInfo.GetUsername(), userinfo, state)
	if err != nil {
		return err
	}
	return nil
}

// UpdateUserDetails updates the info associated with the sr.Sub passed in
func UpdateUserDetails(db *sql.DB, sr *StateRow) error {
	const updStmt = `UPDATE userstate SET username=?, userinfo=?, state=? WHERE cookie=?`
	userinfo, err := proto.Marshal(sr.UserInfo)
	if err != nil {
		return err
	}
	state, err := proto.Marshal(sr.State)
	if err != nil {
		return err
	}
	_, err = db.Exec(updStmt, sr.UserInfo.GetUsername(), userinfo, state, sr.Cookie)
	return err
}

// UpdateUserCookie sets a new cookie for this username
func UpdateUserCookie(db *sql.DB, sr *StateRow) error {
	const updStmt = `UPDATE userstate SET cookie=? WHERE username=?`
	_, err := db.Exec(updStmt, sr.Cookie, sr.UserInfo.GetUsername())
	return err
}

// UpdateUserDetailsWithProto updates the info from the precomputed protos
func UpdateUserDetailsWithProto(db *sql.DB, info *qrpb.GUser, state *qrpb.GameState) error {
	const updStmt = `UPDATE userstate SET userinfo=?, state=? WHERE username=?`
	infob, err := proto.Marshal(info)
	if err != nil {
		return err
	}
	stateb, err := proto.Marshal(state)
	if err != nil {
		return err
	}
	_, err = db.Exec(updStmt, infob, stateb, info.GetUsername())
	return err
}

// DeleteCookieEntry removes a cookie
func DeleteCookieEntry(db *sql.DB, sr *StateRow) error {
	const delCookie = `DELETE FROM userstate WHERE cookie=?`
	if len(sr.Cookie) < 10 {
		return fmt.Errorf("cookie too small")
	}
	_, err := db.Exec(delCookie, sr.Cookie)
	if err != nil {
		return err
	}
	return nil
}

// AddActionLog records a new gamelog for a user
func AddActionLog(db *sql.DB, lr *LogRow) error {
	const insData = `INSERT INTO gamelogs VALUES(?,?,?)`
	gamelog, err := proto.Marshal(lr.GameLog)
	if err != nil {
		return err
	}
	_, err = db.Exec(insData, lr.Username, lr.Updated, gamelog)
	if err != nil {
		return err
	}
	return nil
}

// GetAllLogsForUser fetches all the LogRows for the given user ID.
func GetAllLogsForUser(db *sql.DB, username string) ([]LogRow, error) {
	const getData = `SELECT username, updated, gamelog FROM gamelogs WHERE username=? ORDER BY updated DESC`
	rows, err := db.Query(getData, username)
	if err != nil {
		log.Printf("Db-err: %v\n", err)
		return nil, err
	}
	defer rows.Close()
	reply := make([]LogRow, 0)
	for rows.Next() {
		var s nullableLogRow
		if err = rows.Scan(&s.Username, &s.Updated, &s.GameLog); err != nil {
			return nil, err
		}
		sr, err := s.toLogRow()
		if err != nil {
			return nil, err
		}
		reply = append(reply, *sr)
	}
	return reply, nil
}

// AdminGetAllUserStates gets an admin view of all the users
func AdminGetAllUserStates(db *sql.DB) ([]StateRow, error) {
	const getData = `SELECT username, userinfo, state FROM userstate`
	rows, err := db.Query(getData)
	if err != nil {
		log.Printf("Db-err: %v\n", err)
		return nil, err
	}
	defer rows.Close()
	reply := make([]StateRow, 0)
	for rows.Next() {
		var s nullableStateRow
		if err = rows.Scan(&s.Username, &s.UserInfo, &s.State); err != nil {
			return nil, err
		}
		sr, err := s.toStateRow()
		if err != nil {
			return nil, err
		}
		reply = append(reply, *sr)
	}
	return reply, nil
}

// AdminGetAllUserLogs gets an admin view of all the logs
func AdminGetAllUserLogs(db *sql.DB) ([]LogRow, error) {
	const getData = `SELECT username, updated, gamelog FROM gamelogs ORDER BY updated DESC LIMIT 1000`
	rows, err := db.Query(getData)
	if err != nil {
		log.Printf("Db-err: %v\n", err)
		return nil, err
	}
	defer rows.Close()
	reply := make([]LogRow, 0)
	for rows.Next() {
		var s nullableLogRow
		if err = rows.Scan(&s.Username, &s.Updated, &s.GameLog); err != nil {
			return nil, err
		}
		sr, err := s.toLogRow()
		if err != nil {
			return nil, err
		}
		reply = append(reply, *sr)
	}
	return reply, nil
}
