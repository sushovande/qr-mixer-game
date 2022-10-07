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
	"testing"

	"github.com/sushovande/qr-mixer-game/qrpb"

	"google.golang.org/protobuf/proto"
)

func GetStateRow() StateRow {
	sr := NewStateRow()
	sr.Cookie = "{ldjfdjfdsjfldsfd}"
	sr.Username = "fluffy"
	sr.State = &qrpb.GameState{
		UserLevel: proto.Int64(1),
		Life:      proto.Int64(3),
	}
	sr.UserInfo = &qrpb.GUser{
		Username: proto.String("fluffy"),
		Name:     proto.String("Fluffy Person"),
	}
	return sr
}

func GetLogRow() LogRow {
	lr := NewLogRow()
	lr.Username = "fluffy"
	lr.Updated = 1572354799000000
	lr.GameLog = &qrpb.ActionLog{
		ClueShortName: proto.String("orebaba"),
		TimestampUsec: proto.Int64(1572354799000000),
		Type:          qrpb.ActionLog_ACTION_CODE_SCAN.Enum(),
	}
	return lr
}

func TestAddUser(t *testing.T) {
	db, err := DbInit(":memory:")
	defer func() {
		if err := db.Close(); err != nil {
			fmt.Printf("Error closing db %v", err)
		}
	}()
	if err != nil {
		t.Error(err)
	}
	err = db.Ping()
	if err != nil {
		t.Error(err)
	}
	sr := GetStateRow()
	err = AddUser(db, &sr)
	if err != nil {
		t.Error(err)
	}
}

func TestGetUserStateByCookie(t *testing.T) {
	db, _ := DbInit(":memory:")
	defer db.Close()
	sr := GetStateRow()
	AddUser(db, &sr)
	got, err := GetUserStateByCookie(db, sr.Cookie)
	if err != nil {
		t.Error(err)
	}
	if got.Cookie != sr.Cookie {
		t.Errorf("Wrong cookie. Expected %v, Got %v.", got.Cookie, sr.Cookie)
	}
	if got.Username != sr.Username {
		t.Errorf("Wrong Username. Expected %v, Got %v.", got.Username, sr.Username)
	}
	if got.UserInfo.GetName() != sr.UserInfo.GetName() {
		t.Errorf("Wrong UserInfo.GetName(). Expected %v, Got %v.", got.UserInfo.GetName(), sr.UserInfo.GetName())
	}
	if got.UserInfo.GetUsername() != sr.UserInfo.GetUsername() {
		t.Errorf("Wrong UserInfo.GetUsername(). Expected %v, Got %v.", got.UserInfo.GetUsername(), sr.UserInfo.GetUsername())
	}
}

func TestUpdateUserDetails(t *testing.T) {
	db, _ := DbInit(":memory:")
	defer db.Close()
	sr := GetStateRow()
	AddUser(db, &sr)
	sr.State.Life = proto.Int64(2)
	sr.UserInfo.Name = proto.String("obaba")
	err := UpdateUserDetails(db, &sr)
	if err != nil {
		t.Error(err)
	}
	got, err := GetUserStateByCookie(db, sr.Cookie)
	if err != nil {
		t.Error(err)
	}
	if got.Cookie != sr.Cookie {
		t.Errorf("Wrong cookie. Expected %v, Got %v.", got.Cookie, sr.Cookie)
	}
	if got.Username != sr.Username {
		t.Errorf("Wrong Username. Expected %v, Got %v.", sr.Username, got.Username)
	}
	if got.UserInfo.GetName() != sr.UserInfo.GetName() {
		t.Errorf("Wrong UserInfo.GetName(). Expected %v, Got %v.", got.UserInfo.GetName(), sr.UserInfo.GetName())
	}
	if got.UserInfo.GetUsername() != sr.UserInfo.GetUsername() {
		t.Errorf("Wrong UserInfo.GetUsername(). Expected %v, Got %v.", got.UserInfo.GetUsername(), sr.UserInfo.GetUsername())
	}
}

func TestDeleteCookieEntry(t *testing.T) {
	db, _ := DbInit(":memory:")
	defer db.Close()
	sr := GetStateRow()
	AddUser(db, &sr)
	err := DeleteCookieEntry(db, &sr)
	if err != nil {
		t.Error(err)
	}
	got, err := GetUserStateByCookie(db, sr.Cookie)
	if err != nil {
		t.Error(err)
	}
	if got != nil {
		t.Errorf("user was not deleted")
	}
	sr.Cookie = "asb"
	err = DeleteCookieEntry(db, &sr)
	if err == nil {
		t.Errorf("minimum length check for cookie was not performed")
	}
}

func TestAddActionLog(t *testing.T) {
	db, _ := DbInit(":memory:")
	defer db.Close()
	lr := GetLogRow()
	err := AddActionLog(db, &lr)
	if err != nil {
		t.Error(err)
	}
}

func TestGetAllLogsForUser(t *testing.T) {
	db, _ := DbInit(":memory:")
	defer db.Close()
	lr := GetLogRow()
	AddActionLog(db, &lr)
	resp, err := GetAllLogsForUser(db, lr.Username)
	if err != nil {
		t.Error(err)
	}
	if len(resp) != 1 {
		t.Fatalf("Expected one log, got: %v", len(resp))
	}
	if resp[0].Username != lr.Username {
		t.Errorf("Wrong Username in log. Expected: %v, Got %v.", resp[0].Username, lr.Username)
	}
	if resp[0].Updated != lr.Updated {
		t.Errorf("Wrong Updated in log. Expected: %v, Got %v.", resp[0].Updated, lr.Updated)
	}
	if resp[0].GameLog.GetClueShortName() != lr.GameLog.GetClueShortName() {
		t.Errorf("Wrong GameLog.GetClueShortName() in log. Expected: %v, Got %v.", resp[0].GameLog.GetClueShortName(), lr.GameLog.GetClueShortName())
	}
	if resp[0].GameLog.GetTimestampUsec() != lr.GameLog.GetTimestampUsec() {
		t.Errorf("Wrong GameLog.GetTimestampUsec() in log. Expected: %v, Got %v.", resp[0].GameLog.GetTimestampUsec(), lr.GameLog.GetTimestampUsec())
	}
	if resp[0].GameLog.GetType() != lr.GameLog.GetType() {
		t.Errorf("Wrong GameLog.GetType() in log. Expected: %v, Got %v.", resp[0].GameLog.GetType(), lr.GameLog.GetType())
	}

	lr.Updated = 1572355317000000
	lr.GameLog.TimestampUsec = proto.Int64(1572355317000000)
	lr.GameLog.ClueShortName = proto.String("wrongclue")
	err = AddActionLog(db, &lr)
	if err != nil {
		t.Error(err)
	}
	resp, err = GetAllLogsForUser(db, lr.Username)
	if err != nil {
		t.Error(err)
	}
	if len(resp) != 2 {
		t.Fatalf("Expected two logs, got: %v", len(resp))
	}
	if resp[0].GameLog.GetClueShortName() != "wrongclue" {
		t.Errorf("Wrong order of clues in log. Expected: wrongclue, Got %v.", resp[0].GameLog.GetClueShortName())
	}
}
