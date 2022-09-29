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
	"qr-mixer-game/qrpb"
	"testing"

	"google.golang.org/protobuf/encoding/prototext"
	"google.golang.org/protobuf/proto"
)

func GetSyntheticStateRow(usernameno int, level int64) *StateRow {
	sr := NewStateRow()
	sr.Cookie = fmt.Sprintf("cookie-%v", usernameno)
	sr.Username = fmt.Sprintf("username-%v", usernameno)
	sr.State = &qrpb.GameState{
		UserLevel: proto.Int64(level),
		Life:      proto.Int64(3),
	}
	sr.UserInfo = &qrpb.GUser{
		Username: proto.String(sr.Username),
		Name:     proto.String(fmt.Sprintf("name-%v", usernameno)),
	}
	return &sr
}

func TestStepRegularQuestion(t *testing.T) {
	env, err := createEnv(":memory:")
	if err != nil {
		t.Fatal(err)
	}
	defer env.db.Close()
	setupSynthetic(env.cgo, 10)
	u1 := GetSyntheticStateRow(1, 6)
	AddUser(env.db, u1)

	mr, err := env.Step(u1.State, "qrcode-6")
	if err != nil {
		t.Fatal(err)
	}
	if mr.actionString != "Correct!" {
		t.Errorf("expected progress. got: %v", mr.actionString)
	}

	mr, err = env.Step(u1.State, "qrcode-9")
	if err != nil {
		t.Fatal(err)
	}
	if mr.actionString != "Lost a Life!" {
		t.Errorf("expected loss. got: %v", mr.actionString)
	}
}

func TestGetMetalOnLevel9(t *testing.T) {
	env, err := createEnv(":memory:")
	if err != nil {
		t.Fatal(err)
	}
	defer env.db.Close()
	setupSynthetic(env.cgo, 10)
	u1 := GetSyntheticStateRow(1, 9)
	AddUser(env.db, u1)

	// On level 9, if we answer correctly, we will reach level 10.
	// We don't have a metal yet, so we should always get a metal here
	mr, _ := env.Step(u1.State, "qrcode-10")
	if !(mr.newState.GetHasAl() || mr.newState.GetHasCu()) {
		ps, _ := prototext.Marshal(mr.newState)
		t.Errorf("expected to have either al or cu. got: %v", string(ps))
	}
}

func TestEndgameGrabMetal(t *testing.T) {
	env, err := createEnv(":memory:")
	if err != nil {
		t.Fatal(err)
	}
	defer env.db.Close()
	setupSynthetic(env.cgo, 10)

	u1 := GetSyntheticStateRow(1, 20)
	u1.State.HasAl = proto.Bool(true)
	u1.State.HasSn = proto.Bool(true)
	AddUser(env.db, u1)

	u2 := GetSyntheticStateRow(2, 19)
	u2.State.HasAl = proto.Bool(true)
	u2.State.HasZn = proto.Bool(true)
	AddUser(env.db, u2)

	u4 := GetSyntheticStateRow(4, 20)
	u4.State.HasCu = proto.Bool(true)
	u4.State.HasSn = proto.Bool(true)
	AddUser(env.db, u4)

	// u1 is on level 20. On scanning u2, they should grab zinc
	mr, err := env.Step(u1.State, "qrcode-2")
	if err != nil {
		t.Fatal(err)
	}
	if mr.actionString != "Grabbed Metal!" {
		t.Errorf("expected metal. got: %v", mr.actionString)
	}
	if !mr.newState.GetHasZn() {
		ps, _ := prototext.Marshal(mr.newState)
		t.Errorf("expected to have zinc. got: %v", string(ps))
	}

	// u1 has just copper missing. If they scan u4, they should progress
	u1.State = mr.newState
	mr, _ = env.Step(u1.State, "qrcode-4")
	if mr.actionString != "Grabbed Metal!" {
		t.Errorf("expected metal. got: %v", mr.actionString)
	}
	if !mr.newState.GetHasSn() {
		ps, _ := prototext.Marshal(mr.newState)
		t.Errorf("expected to have zinc. got: %v", string(ps))
	}
	if mr.newState.GetUserLevel() != 21 {
		ps, _ := prototext.Marshal(mr.newState)
		t.Errorf("expected to reach level 21. got: %v", string(ps))
	}
}
func TestAnyPersonQuestion(t *testing.T) {
	env, err := createEnv(":memory:")
	if err != nil {
		t.Fatal(err)
	}
	defer env.db.Close()
	setupSynthetic(env.cgo, 10)
	u1 := GetSyntheticStateRow(1, 19)
	AddUser(env.db, u1)

	// On level 19, all answers should be accepted
	mr, err := env.Step(u1.State, "qrcode-12")
	if err != nil {
		t.Fatal(err)
	}
	if mr.actionString != "Correct!" {
		t.Errorf("expected progress. got: %v", mr.actionString)
	}
}
