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

	_ "github.com/mattn/go-sqlite3"
)

// DbInit creates the connection and tables.
func DbInit(dbFilename string) (*sql.DB, error) {
	db, err := sql.Open("sqlite3", dbFilename)
	if err != nil {
		return nil, err
	}
	if err = db.Ping(); err != nil {
		return nil, err
	}
	err = MaybeCreateTables(db)
	if err != nil {
		return db, err
	}
	return db, nil
}

// MaybeCreateTables creates the required db tables if they didn't already exist
func MaybeCreateTables(db *sql.DB) error {
	if err := MaybeCreateUserTable(db); err != nil {
		return err
	}
	if err := MaybeCreateOptionsTable(db); err != nil {
		return err
	}
	return nil
}
