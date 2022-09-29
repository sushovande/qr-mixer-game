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
	"net/http"
	"qr-mixer-game/common"
	"strings"

	qrcode "github.com/skip2/go-qrcode"
)

func (env *Env) adminGetCard(w http.ResponseWriter, r *http.Request) {
	toks, err := common.GetURLTokens(r.URL.Path, "/9283e316-beaa-4182-b3a6-0937046251ee")
	if common.Should500(err, w, "could not tokenize your URL") {
		return
	}

	cardPath, ok := toks["card"]
	if !ok {
		common.Should500(fmt.Errorf("could not find the card param"), w, "could not find the card param")
		return
	}

	words := strings.Split(cardPath, "-")
	if len(words) != 2 {
		common.Should500(fmt.Errorf("could not figure out card value"), w, "could not figure out card value")
		return
	}

	suit := words[0]
	rankString := strings.TrimSuffix(words[1], ".svg")
	colorString := "#000"

	suitUnicode := "♠"
	switch suit {
	case "SPADES":
		suitUnicode = "♠"
	case "HEARTS":
		suitUnicode = "♥"
		colorString = "#f00"
	case "CLUBS":
		suitUnicode = "♣"
	case "DIAMONDS":
		suitUnicode = "♦"
		colorString = "#f00"
	}

	switch rankString {
	case "1":
		rankString = "A"
	case "11":
		rankString = "J"
	case "12":
		rankString = "Q"
	case "13":
		rankString = "K"
	}

	disp := struct {
		Suit  string
		Rank  string
		Color string
	}{
		Suit:  suitUnicode,
		Rank:  rankString,
		Color: colorString,
	}

	w.Header().Set("Content-Type", "image/svg+xml")
	common.RenderTemplate(w, env.tem, "card-template.svg", disp)
}

func (env *Env) adminGetQrImage(w http.ResponseWriter, r *http.Request) {
	qrstring := r.URL.Query().Get("qr")
	if len(qrstring) == 0 {
		common.Should500(fmt.Errorf("could not find the card param"), w, "could not find the card param")
		return
	}

	qrstring = strings.TrimSuffix(qrstring, ".png")
	png, err := qrcode.Encode(qrstring, qrcode.High, 450)

	if common.Should500(err, w, "could not encode qrcode") {
		return
	}

	w.Header().Set("Content-Type", "image/png")
	w.Write(png)
}
