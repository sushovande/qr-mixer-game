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
	"reflect"
	"testing"
)

func TestGetURLTokens(t *testing.T) {
	m, err := GetURLTokens("/koo/jumbo/1/gumbo/2", "/koo")
	if err != nil {
		t.Fatal(err)
	}
	expected := map[string]string{
		"jumbo": "1",
		"gumbo": "2",
	}
	if !reflect.DeepEqual(m, expected) {
		t.Errorf("GetURLToken fail, got %v", m)
	}
	if m, _ = GetURLTokens("/jumbo/1/gumbo/2", ""); !reflect.DeepEqual(m, expected) {
		t.Errorf("GetURLToken fail, got %v", m)
	}
	if m, _ = GetURLTokens("/gumbo/2/jumbo/1", ""); !reflect.DeepEqual(m, expected) {
		t.Errorf("GetURLToken fail, got %v", m)
	}
	if m, _ = GetURLTokens("/jumbo/1/gumbo/2", "/"); !reflect.DeepEqual(m, expected) {
		t.Errorf("GetURLToken fail, got %v", m)
	}
	if m, _ = GetURLTokens("/koo/jumbo/1/gumbo/2/", "/koo"); !reflect.DeepEqual(m, expected) {
		t.Errorf("GetURLToken fail, got %v", m)
	}
	if m, _ = GetURLTokens("/koo/jumbo/1/gumbo/2/", "/koo/"); !reflect.DeepEqual(m, expected) {
		t.Errorf("GetURLToken fail, got %v", m)
	}
	if m, _ = GetURLTokens("/foo/koo/jumbo/1/gumbo/2/", "/foo/koo"); !reflect.DeepEqual(m, expected) {
		t.Errorf("GetURLToken fail, got %v", m)
	}
	if m, _ = GetURLTokens("/jumbo/1/gumbo/2/extra", ""); !reflect.DeepEqual(m, expected) {
		t.Errorf("GetURLToken fail, got %v", m)
	}
	expected = map[string]string{
		"simple": "simon",
	}
	if m, _ = GetURLTokens("/simple/simon", ""); !reflect.DeepEqual(m, expected) {
		t.Errorf("GetURLToken fail, got %v", m)
	}
	if m, _ = GetURLTokens("simple/simon", ""); !reflect.DeepEqual(m, expected) {
		t.Errorf("GetURLToken fail, got %v", m)
	}
	_, err = GetURLTokens("/koo/jumbo/1/gumbo/2", "/hey")
	if err == nil {
		t.Errorf("expected error for mismatch of prefix")
	}
}
