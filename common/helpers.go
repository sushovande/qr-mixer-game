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

package common

import (
	"fmt"
	"html/template"
	"log"
	"net/http"
	"strings"
)

// Should500 checks if err is not nil, and if so logs and sends the error
func Should500(err error, w http.ResponseWriter, msg string) bool {
	if err != nil {
		log.Printf("page-err: %v", err)
		w.WriteHeader(http.StatusInternalServerError)
		fmt.Fprintln(w, msg)
		return true
	}
	return false
}

// GetURLTokens returns the tokens in a URL minus the prefix as key-value pairs.
// Visible for testing only.
func GetURLTokens(p string, prefix string) (map[string]string, error) {
	r := make(map[string]string)
	if !strings.HasPrefix(p, prefix) {
		return r, fmt.Errorf("Unexpected prefix")
	}
	remains := p[len(prefix):]
	if len(remains) == 0 {
		return r, nil
	}
	if remains[0] == '/' {
		remains = remains[1:]
	}

	allToks := strings.Split(remains, "/")
	var k string
	for i, t := range allToks {
		if i%2 == 0 {
			k = t
		} else {
			r[k] = t
		}
	}
	return r, nil
}

// RenderTemplate takes in the template pool, name, and params, does error checking, and renders.
func RenderTemplate(w http.ResponseWriter, t *template.Template, nm string, data interface{}) {
	err := t.ExecuteTemplate(w, nm, data)
	if Should500(err, w, "HTTP 500: We could not render the page: "+nm) {
		return
	}
}

// RespondHTTP401 responds with HTTP 401 to the request, asking user to log in.
func RespondHTTP401(w http.ResponseWriter) {
	w.Header().Add("Content-Type", "text/html;charset=utf-8")
	w.WriteHeader(http.StatusUnauthorized)
	fmt.Fprintln(w, "You have to <a href=\"/login\">login</a> to view this page")
}
