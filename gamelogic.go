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
	"math/rand"

	"github.com/sushovande/qr-mixer-game/qrpb"

	proto "google.golang.org/protobuf/proto"
)

// Step returns the GameState resulting from the action at the current GameState
func (env *Env) Step(old *qrpb.GameState, answer string) (StepResponse, error) {
	qrm, err := env.cgo.GetQRMappings()
	if err != nil {
		return StepResponse{}, err
	}

	// Set up defaults
	result := NewStepResponse()
	result.actionString = "Lost a Life!"
	result.actionResult = *qrpb.ActionLog_RESULT_LOST_LIFE.Enum()
	result.scannedClue = qrm.LookupByQrCode(answer).GetUsername()
	result.newState = proto.Clone(old).(*qrpb.GameState)

	// ENDGAME logic
	if old.GetUserLevel() == 22 {
		result.actionString = "Already Victorious!"
		result.actionResult = *qrpb.ActionLog_RESULT_ALREADY_VICTORIOUS.Enum()
		return result, nil
	}

	// 21 will be handled by regular list_username logic below, with username set to cauldron

	if old.GetUserLevel() == 20 {
		// You scanned someone and tried to get their metals
		gs, err := GetUserStateByUsername(env.db, result.scannedClue)
		if err != nil {
			return StepResponse{}, err
		}
		if gs == nil {
			// scanned someone who is not yet registered?
			return StepResponse{}, fmt.Errorf("you tried to get metals from someone who is not yet registered in the game")
		}

		if GrabMetalFromSomeone(&result, gs.State) {
			result.actionString = "Grabbed Metal!"
			result.actionResult = *qrpb.ActionLog_RESULT_GRABBED_METAL.Enum()

			if result.newState.GetHasAl() && result.newState.GetHasCu() && result.newState.GetHasSn() && result.newState.GetHasZn() {
				result.newState.UserLevel = proto.Int64(old.GetUserLevel() + 1)
			}
		} else {
			result.actionString = "Nothing Found!"
			result.actionResult = *qrpb.ActionLog_RESULT_NO_GRABBED_METAL.Enum()
		}

		return result, nil
	}

	// Regular questions
	sqs, err := env.cgo.GetGameQSet()
	if err != nil {
		return StepResponse{}, err
	}

	sq := GetQuestionByIndex(sqs, old.GetUserLevel())

	if *sq.Type == qrpb.GQType_USERNAME_LIST {
		if ListHasString(sq.AnsUsernames, result.scannedClue) {
			// TODO: check if this level arithmetic is desirable
			result.newState.UserLevel = proto.Int64(old.GetUserLevel() + 1)
			result.actionString = "Correct!"
			result.actionResult = *qrpb.ActionLog_RESULT_PROGRESS.Enum()
		} else {
			result.newState.Life = proto.Int64(old.GetLife() - 1)
		}
	} else if *sq.Type == qrpb.GQType_SURVEY_ANS {
		gu, err := GetUserInfoByUsername(env.db, result.scannedClue)
		if err != nil {
			return StepResponse{}, err
		}
		if gu == nil {
			// scanned someone who is not yet registered?
			return StepResponse{}, fmt.Errorf("you scanned someone who is not yet registered in the game")
		}
		if getSurveyResponse(gu, sq.GetSurveyId()) == sq.GetSurveyTrueIsCorrect() {
			result.newState.UserLevel = proto.Int64(old.GetUserLevel() + 1)
			result.actionString = "Correct!"
			result.actionResult = *qrpb.ActionLog_RESULT_PROGRESS.Enum()
		} else {
			result.newState.Life = proto.Int64(old.GetLife() - 1)
		}
	} else if *sq.Type == qrpb.GQType_ANY_PERSON {
		// TODO: if at all needed, remove the ability to scan inanimate objects at this time.
		result.newState.UserLevel = proto.Int64(old.GetUserLevel() + 1)
		result.actionString = "Correct!"
		result.actionResult = *qrpb.ActionLog_RESULT_PROGRESS.Enum()
	}

	MaybeGrantMetal(&result)

	if result.newState.GetLife() <= 0 {
		result.actionString = "Dead!"
		result.actionResult = *qrpb.ActionLog_RESULT_ALREADY_DEAD.Enum()
		result.newState.UserLevel = proto.Int64(-1)
	}

	result.levelClue = GetQuestionByIndex(sqs, *result.newState.UserLevel).GetQuestionHtml()
	return result, nil
}

func MaybeGrantMetal(result *StepResponse) {
	if result.actionResult != *qrpb.ActionLog_RESULT_PROGRESS.Enum() {
		return
	}

	// Check stage-1 granting.
	if *result.newState.UserLevel >= 8 && *result.newState.UserLevel <= 10 {
		if result.newState.GetHasAl() || result.newState.GetHasCu() {
			return
		}

		// choose a metal to grant
		metal := "al"
		if rand.Float32() > 0.5 {
			metal = "cu"
		}

		if *result.newState.UserLevel == 8 {
			if rand.Float32() > 0.4 {
				GrantMetal(result, metal)
			}
		} else if *result.newState.UserLevel == 9 {
			if rand.Float32() > 0.7 {
				GrantMetal(result, metal)
			}
		} else if *result.newState.UserLevel == 10 {
			GrantMetal(result, metal)
		}
	}

	// Check stage-2 granting.
	if *result.newState.UserLevel >= 18 && *result.newState.UserLevel <= 20 {
		if result.newState.GetHasSn() || result.newState.GetHasZn() {
			return
		}

		// choose a metal to grant
		metal := "sn"
		if rand.Float32() > 0.5 {
			metal = "zn"
		}

		if *result.newState.UserLevel == 18 {
			if rand.Float32() > 0.4 {
				GrantMetal(result, metal)
			}
		} else if *result.newState.UserLevel == 19 {
			if rand.Float32() > 0.7 {
				GrantMetal(result, metal)
			}
		} else if *result.newState.UserLevel == 20 {
			GrantMetal(result, metal)
		}
	}
}

// GrabMetalFromSomeone grabs a shared metal from another user in the endgame.
// Returns true if a metal was grabbed.
func GrabMetalFromSomeone(result *StepResponse, gs *qrpb.GameState) bool {
	grabbable := make([]string, 0)
	if gs.GetHasAl() && !result.newState.GetHasAl() {
		grabbable = append(grabbable, "al")
	}
	if gs.GetHasCu() && !result.newState.GetHasCu() {
		grabbable = append(grabbable, "cu")
	}
	if gs.GetHasSn() && !result.newState.GetHasSn() {
		grabbable = append(grabbable, "sn")
	}
	if gs.GetHasZn() && !result.newState.GetHasZn() {
		grabbable = append(grabbable, "zn")
	}

	if len(grabbable) == 0 {
		return false
	}
	w := rand.Intn(len(grabbable))
	GrantMetal(result, grabbable[w])
	return true
}

func GrantMetal(result *StepResponse, metal string) {
	switch metal {
	case "al":
		result.newState.HasAl = proto.Bool(true)
		return
	case "cu":
		result.newState.HasCu = proto.Bool(true)
		return
	case "sn":
		result.newState.HasSn = proto.Bool(true)
		return
	case "zn":
		result.newState.HasZn = proto.Bool(true)
		return
	}
}

func GetQuestionByIndex(sqs *qrpb.GameQSet, n int64) *qrpb.GameQuestion {
	for _, q := range sqs.GameQuestions {
		if q.GetQuestionId() == n {
			return q
		}
	}
	return nil
}

func GetSurveyQuestionByIndex(gopt *qrpb.SurveySet, n int64) *qrpb.SurveyQuestion {
	for _, q := range gopt.SurveyQuestions {
		if q.GetQuestionId() == n {
			return q
		}
	}
	return nil
}

func getSurveyResponse(gu *qrpb.GUser, qid int64) bool {
	for _, sa := range gu.SurveyAnswers {
		if *sa.QuestionId == qid {
			return sa.GetIsTrue()
		}
	}
	return false
}

func ListHasString(list []string, needle string) bool {
	for _, s := range list {
		if s == needle {
			return true
		}
	}
	return false
}
