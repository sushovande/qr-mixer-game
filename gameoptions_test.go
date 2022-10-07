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
	"testing"
	"time"

	"github.com/sushovande/qr-mixer-game/qrpb"

	"google.golang.org/protobuf/proto"
)

func TestGetHardcodedOptions(t *testing.T) {
	db, _ := DbInit(":memory:")
	defer db.Close()
	cgo := CreateCachedGameOptions(db)
	qrgo, err := cgo.GetSurveySet()
	if err != nil {
		t.Fatal(err)
	}
	if len(qrgo.SurveyQuestions) == 0 {
		t.Errorf("expected nonzero survey_question, got %v", len(qrgo.SurveyQuestions))
	}
}

func TestSetAndRetrieveOptions(t *testing.T) {
	db, _ := DbInit(":memory:")
	defer db.Close()
	cgo := CreateCachedGameOptions(db)
	qrgo, _ := cgo.GetSurveySet()
	qrgo.SurveyQuestions[0].QuestionText = proto.String("foo")
	err := cgo.SetSurveySet(qrgo)
	if err != nil {
		t.Fatal(err)
	}

	// check the cached value
	r1, err := cgo.GetSurveySet()
	if err != nil {
		t.Fatal(err)
	}
	if *r1.SurveyQuestions[0].QuestionText != "foo" {
		t.Errorf("Did not get the right question text back. Expected foo, got %v", *r1.SurveyQuestions[0].QuestionText)
	}

	// dirty the cache
	cgo.lastUpdated = time.Unix(10, 10)
	// It should fetch from db and still get the right value

	r2, err := cgo.GetSurveySet()
	if err != nil {
		t.Fatal(err)
	}
	if *r2.SurveyQuestions[0].QuestionText != "foo" {
		t.Errorf("Did not get the right question text back. Expected foo, got %v", *r2.SurveyQuestions[0].QuestionText)
	}
}

func TestGameQSet(t *testing.T) {
	db, _ := DbInit(":memory:")
	defer db.Close()
	cgo := CreateCachedGameOptions(db)
	sqs, err := cgo.GetGameQSet()
	if err != nil {
		t.Fatal(err)
	}

	if len(sqs.GameQuestions) == 0 {
		t.Fatalf("expected non-empty questions from hardcoded SQS")
	}

	if *sqs.GameQuestions[0].Type != qrpb.GQType_USERNAME_LIST {
		t.Errorf("expected the first question to be of type USERNAME list, got: %v", sqs.GameQuestions[0].Type)
	}

	var s3 qrpb.GameQuestion
	s3.QuestionId = proto.Int64(1)
	s3.QuestionHtml = proto.String("foo")
	s3.Type = qrpb.GQType_USERNAME_LIST.Enum()
	s3.AnsUsernames = make([]string, 0)
	s3.AnsUsernames = append(s3.AnsUsernames, "ld1")
	s3.AnsUsernames = append(s3.AnsUsernames, "ld2")

	var sqs2 qrpb.GameQSet
	sqs2.GameQuestions = make([]*qrpb.GameQuestion, 1)
	sqs2.GameQuestions[0] = &s3
	err = cgo.SetGameQSet(&sqs2)
	if err != nil {
		t.Fatal(err)
	}

	sqs, err = cgo.GetGameQSet()
	if err != nil {
		t.Fatal(err)
	}

	if len(sqs.GameQuestions) != 1 {
		t.Fatalf("expected exactly 1 item in the modified SQS, got %v", len(sqs.GameQuestions))
	}

	if *sqs.GameQuestions[0].Type != qrpb.GQType_USERNAME_LIST {
		t.Errorf("expected the first question to be of type USERNAME list, got: %v", sqs.GameQuestions[0].Type)
	}
	if *sqs.GameQuestions[0].QuestionHtml != "foo" {
		t.Errorf("expected html foo, got: %v", sqs.GameQuestions[0].GetQuestionHtml())
	}
	if len(sqs.GameQuestions[0].AnsUsernames) != 2 {
		t.Errorf("expected 2 ans usernames, got: %v", len(sqs.GameQuestions[0].AnsUsernames))
	}
	if sqs.GameQuestions[0].AnsUsernames[0] != "ld1" {
		t.Errorf("expected the first ans username to be ld1, got: %v", sqs.GameQuestions[0].AnsUsernames[0])
	}

	// dirty the cache
	cgo.lastUpdated = time.Unix(10, 10)
	// It should fetch from db and still get the right value
	sqs, err = cgo.GetGameQSet()
	if err != nil {
		t.Fatal(err)
	}

	if len(sqs.GameQuestions) != 1 {
		t.Fatalf("expected exactly 1 item in the modified SQS, got %v", len(sqs.GameQuestions))
	}
}

func TestSQAndGOTogether(t *testing.T) {
	db, _ := DbInit(":memory:")
	defer db.Close()
	cgo := CreateCachedGameOptions(db)
	sqs, _ := cgo.GetGameQSet()
	qrgo, _ := cgo.GetSurveySet()
	cgo.SetGameQSet(sqs)
	cgo.SetSurveySet(qrgo)
	cgo.lastUpdated = time.Unix(10, 10)
	// Fetch it again, it should hit the db, and there should be no parse errors.
	sqs, err := cgo.GetGameQSet()
	if err != nil {
		t.Fatal(err)
	}
	qrgo, err = cgo.GetSurveySet()
	if err != nil {
		t.Fatal(err)
	}

	if len(sqs.GameQuestions) == 0 {
		t.Errorf("expected nonzero questions")
	}
	if len(qrgo.SurveyQuestions) == 0 {
		t.Error("expected nonzero SurveyQuestion questions")
	}
}

func TestGetHardcodedMappings(t *testing.T) {
	db, _ := DbInit(":memory:")
	defer db.Close()
	cgo := CreateCachedGameOptions(db)

	mp, err := cgo.GetQRMappings()
	if err != nil {
		t.Fatal(err)
	}

	if mp == nil {
		t.Fatal("expected non nil QR Mappings")
	}

	if len(mp.mappings.GetQrMappings()) == 0 {
		t.Fatal("expected non zero QR Mappings")
	}
}

func TestSetQRMappings(t *testing.T) {
	db, _ := DbInit(":memory:")
	defer db.Close()
	cgo := CreateCachedGameOptions(db)

	q1 := &qrpb.QRMapping{
		Username:    proto.String("u1"),
		DisplayName: proto.String("User 1"),
		Qrcode:      proto.String("qrcode01"),
		CardSuit:    qrpb.CardSuit_CLUBS.Enum(),
		CardRank:    proto.Int64(2),
	}
	qms := &qrpb.QRMappingSet{QrMappings: []*qrpb.QRMapping{q1}}

	var qmo QRMappings
	qmo.mappings = qms
	qmo.RefreshMappings()

	err := cgo.SetQRMappings(&qmo)

	if err != nil {
		t.Fatal(err)
	}

	qmgot, err := cgo.GetQRMappings()
	if err != nil {
		t.Fatal(err)
	}
	if qmgot == nil {
		t.Fatal("unexpected nil qr mappings on reading back")
	}

	q := qmgot.LookupByQrCode("qrcode01")
	if q == nil {
		t.Fatal("unexpected nil q on looking up by qr code")
	}
	if q.GetUsername() != "u1" {
		t.Errorf("Wrong username on lookup by qr code. got %v, want u1", q.GetUsername())
	}

	q = qmgot.LookupByUsername("u1")
	if q == nil {
		t.Fatal("unexpected nil q on looking up by username")
	}
	if q.GetQrcode() != "qrcode01" {
		t.Errorf("Wrong qrcode on lookup by username. got %v, want qrcode01", q.GetQrcode())
	}
}
