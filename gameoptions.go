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
	_ "embed"
	"log"
	"qr-mixer-game/qrpb"
	"time"

	"google.golang.org/protobuf/encoding/prototext"
	"google.golang.org/protobuf/proto"
)

const CACHE_TTL_SEC float64 = 60

//go:embed default-data/survey_questions.textproto
var defaultSurveyContent []byte

//go:embed default-data/static_questions.textproto
var defaultQuestionContent []byte

//go:embed default-data/qr_mappings.textproto
var defaultQRMappings []byte

type CachedGameOptions struct {
	surveySet     *qrpb.SurveySet
	gameQuestions *qrpb.GameQSet
	qrMappings    *QRMappings
	lastUpdated   time.Time
	db            *sql.DB
}

type nullableGameOptionsRow struct {
	Key   sql.NullString
	Value sql.RawBytes
}

type QRMappings struct {
	mappings        *qrpb.QRMappingSet
	usernameToQRMap map[string]*qrpb.QRMapping
	qrcodeToQRMap   map[string]*qrpb.QRMapping
}

// MaybeCreateOptionsTable creates the Options table in the db if it didn't exist
func MaybeCreateOptionsTable(db *sql.DB) error {
	const createStmt = `
	CREATE TABLE IF NOT EXISTS gameoptions (
		key TEXT PRIMARY KEY,
		value BLOB
	);
	`
	_, err := db.Exec(createStmt)
	if err != nil {
		log.Printf("Db-err: %v\n", err)
		return err
	}

	// If there isn't any data for options, put in the default values.
	// Note: Insert or Ignore operates on a per-row basis, so if survey is populated but gq is not,
	// it will not overwrite something that is present.
	const populateStmt = `INSERT OR IGNORE INTO gameoptions VALUES
		('surveyset', ?),
		('gqset', ?),
		('qrmap', ?)`

	sp, err := getHardcodedSurveySet()
	if err != nil {
		return err
	}

	sm, err := proto.Marshal(sp)
	if err != nil {
		return err
	}

	gp, err := getHardcodedGameQSet()
	if err != nil {
		return err
	}

	gm, err := proto.Marshal(gp)
	if err != nil {
		return err
	}

	qp, err := getHardcodedQRMappings()
	if err != nil {
		return err
	}

	qm, err := proto.Marshal(qp.mappings)
	if err != nil {
		return err
	}

	_, err = db.Exec(populateStmt, sm, gm, qm)
	return err
}

func CreateCachedGameOptions(db *sql.DB) *CachedGameOptions {
	var cgo CachedGameOptions
	cgo.db = db
	return &cgo
}

func (v *CachedGameOptions) GetSurveySet() (*qrpb.SurveySet, error) {
	if v.surveySet != nil {
		if time.Since(v.lastUpdated).Seconds() < CACHE_TTL_SEC {
			return v.surveySet, nil
		}
	}

	// options is null or stale. Try DB next
	qrgo, err := v.getSurveySetFromDB()
	if err == nil {
		v.surveySet = qrgo
		v.lastUpdated = time.Now()
	}
	return qrgo, err
}

func (v *CachedGameOptions) SetSurveySet(qrgo *qrpb.SurveySet) error {
	v.surveySet = qrgo
	v.lastUpdated = time.Now()
	return v.SetSurveySetToDB(qrgo)
}

func (v *CachedGameOptions) getSurveySetFromDB() (*qrpb.SurveySet, error) {
	const getStmt = `SELECT key, value FROM gameoptions WHERE key='surveyset'`
	rows, err := v.db.Query(getStmt)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	if rows.Next() {
		var r nullableGameOptionsRow
		if err := rows.Scan(&r.Key, &r.Value); err != nil {
			return nil, err
		}
		if len(r.Value) == 0 {
			return nil, nil
		}
		var qrgo qrpb.SurveySet
		if err := proto.Unmarshal(r.Value, &qrgo); err != nil {
			return nil, err
		}
		return &qrgo, nil
	}
	return nil, nil
}

func (v *CachedGameOptions) SetSurveySetToDB(qrgo *qrpb.SurveySet) error {
	const upsertStmt = `
		INSERT OR REPLACE INTO gameoptions VALUES('surveyset', ?)`
	sqlgo, err := proto.Marshal(qrgo)
	if err != nil {
		return err
	}
	_, err = v.db.Exec(upsertStmt, sqlgo)
	return err
}

func getHardcodedSurveySet() (*qrpb.SurveySet, error) {
	var qrgo qrpb.SurveySet
	if err := prototext.Unmarshal(defaultSurveyContent, &qrgo); err != nil {
		return nil, err
	}
	return &qrgo, nil
}

func (v *CachedGameOptions) GetGameQSet() (*qrpb.GameQSet, error) {
	if v.gameQuestions != nil {
		if time.Since(v.lastUpdated).Seconds() < CACHE_TTL_SEC {
			return v.gameQuestions, nil
		}
	}

	// options is null or stale. Try DB next
	qrgo, err := v.getGameQSetFromDB()
	if err == nil && qrgo != nil {
		v.gameQuestions = qrgo
		v.lastUpdated = time.Now()
	}
	return qrgo, err
}

func (v *CachedGameOptions) SetGameQSet(qrgo *qrpb.GameQSet) error {
	v.gameQuestions = qrgo
	v.lastUpdated = time.Now()
	return v.SetGameQSetToDB(qrgo)
}

func (v *CachedGameOptions) getGameQSetFromDB() (*qrpb.GameQSet, error) {
	const getStmt = `SELECT key, value FROM gameoptions WHERE key='gqset'`
	rows, err := v.db.Query(getStmt)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	if rows.Next() {
		var r nullableGameOptionsRow
		if err := rows.Scan(&r.Key, &r.Value); err != nil {
			return nil, err
		}
		if len(r.Value) == 0 {
			return nil, nil
		}
		var qrgo qrpb.GameQSet
		if err := proto.Unmarshal(r.Value, &qrgo); err != nil {
			return nil, err
		}
		return &qrgo, nil
	}
	return nil, nil
}

func (v *CachedGameOptions) SetGameQSetToDB(qrgo *qrpb.GameQSet) error {
	const upsertStmt = `
		INSERT OR REPLACE INTO gameoptions VALUES('gqset', ?)`
	sqlgo, err := proto.Marshal(qrgo)
	if err != nil {
		return err
	}
	_, err = v.db.Exec(upsertStmt, sqlgo)
	return err
}

func getHardcodedGameQSet() (*qrpb.GameQSet, error) {
	var qrgo qrpb.GameQSet
	if err := prototext.Unmarshal(defaultQuestionContent, &qrgo); err != nil {
		return nil, err
	}
	return &qrgo, nil
}

func (v *CachedGameOptions) GetQRMappings() (*QRMappings, error) {
	if v.qrMappings != nil {
		if time.Since(v.lastUpdated).Seconds() < CACHE_TTL_SEC {
			return v.qrMappings, nil
		}
	}

	// qrMappings is null or stale. Try DB next
	qrm, err := v.getQRMappingsFromDB()
	if err == nil && qrm != nil {
		v.qrMappings = qrm
		v.qrMappings.RefreshMappings()
		v.lastUpdated = time.Now()
	}
	return v.qrMappings, err
}

func (v *CachedGameOptions) SetQRMappings(qrm *QRMappings) error {
	v.qrMappings = qrm
	v.qrMappings.RefreshMappings()
	v.lastUpdated = time.Now()
	return v.setQRMappingsToDB(qrm)
}

func (v *CachedGameOptions) getQRMappingsFromDB() (*QRMappings, error) {
	const getStmt = `SELECT key, value FROM gameoptions WHERE key='qrmap'`
	rows, err := v.db.Query(getStmt)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	if rows.Next() {
		var r nullableGameOptionsRow
		if err := rows.Scan(&r.Key, &r.Value); err != nil {
			return nil, err
		}
		if len(r.Value) == 0 {
			return nil, nil
		}
		var qrmp qrpb.QRMappingSet
		if err := proto.Unmarshal(r.Value, &qrmp); err != nil {
			return nil, err
		}
		return &QRMappings{mappings: &qrmp}, nil
	}
	return nil, nil
}

func (v *CachedGameOptions) setQRMappingsToDB(qrm *QRMappings) error {
	const upsertStmt = `
		INSERT OR REPLACE INTO gameoptions VALUES('qrmap', ?)`
	sqlgo, err := proto.Marshal(qrm.mappings)
	if err != nil {
		return err
	}
	_, err = v.db.Exec(upsertStmt, sqlgo)
	return err
}

func getHardcodedQRMappings() (*QRMappings, error) {
	var qrmp qrpb.QRMappingSet
	if err := prototext.Unmarshal(defaultQRMappings, &qrmp); err != nil {
		return nil, err
	}
	return &QRMappings{mappings: &qrmp}, nil
}

// RefreshMappings sets up the maps so that lookups can work.
// Call this every time `mappings` is set.
func (mp *QRMappings) RefreshMappings() {
	mp.usernameToQRMap = make(map[string]*qrpb.QRMapping)
	mp.qrcodeToQRMap = make(map[string]*qrpb.QRMapping)
	for _, m := range mp.mappings.GetQrMappings() {
		mp.usernameToQRMap[m.GetUsername()] = m
		mp.qrcodeToQRMap[m.GetQrcode()] = m
	}
}

func (mp *QRMappings) LookupByUsername(u string) *qrpb.QRMapping {
	return mp.usernameToQRMap[u]
}

func (mp *QRMappings) LookupByQrCode(q string) *qrpb.QRMapping {
	return mp.qrcodeToQRMap[q]
}
