/*
 * Copyright 2019 InfAI (CC SES)
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *    http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

package tests

import (
	"context"
	"github.com/SENERGY-Platform/incident-api/lib"
	"github.com/SENERGY-Platform/incident-api/lib/api"
	"github.com/SENERGY-Platform/incident-api/lib/configuration"
	"github.com/SENERGY-Platform/incident-api/lib/database"
	"github.com/SENERGY-Platform/incident-api/lib/messages"
	"github.com/SENERGY-Platform/incident-api/tests/server"
	"testing"
	"time"
)

func TestInit(t *testing.T) {
	defaultConfig, err := configuration.LoadConfig("../config.json")
	if err != nil {
		t.Error(err)
		return
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer time.Sleep(10 * time.Second) //wait for docker cleanup
	defer cancel()

	config, err := server.New(ctx, defaultConfig)
	if err != nil {
		t.Error(err)
		return
	}

	err = lib.StartWith(ctx, config, api.Factory, database.Factory)
	if err != nil {
		t.Error(err)
		return
	}
}

func TestDatabase(t *testing.T) {
	defaultConfig, err := configuration.LoadConfig("../config.json")
	if err != nil {
		t.Error(err)
		return
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer time.Sleep(10 * time.Second) //wait for docker cleanup
	defer cancel()

	config, err := server.New(ctx, defaultConfig)
	if err != nil {
		t.Error(err)
		return
	}

	err = lib.StartWith(ctx, config, api.Factory, database.Factory)
	if err != nil {
		t.Error(err)
		return
	}

	incident1 := messages.IncidentMessage{
		Id:                  "foo_id_1",
		MsgVersion:          1,
		ExternalTaskId:      "task_id_1",
		ProcessInstanceId:   "piid_1",
		ProcessDefinitionId: "pdid_1",
		WorkerId:            "w",
		ErrorMessage:        "error message",
		Time:                time.Now(),
	}

	incident2 := messages.IncidentMessage{
		Id:                  "foo_id_2",
		MsgVersion:          1,
		ExternalTaskId:      "task_id_2",
		ProcessInstanceId:   "piid_1",
		ProcessDefinitionId: "pdid_1",
		WorkerId:            "w",
		ErrorMessage:        "error message",
		Time:                time.Now(),
	}

	incident3 := messages.IncidentMessage{
		Id:                  "foo_id_3",
		MsgVersion:          1,
		ExternalTaskId:      "task_id_3",
		ProcessInstanceId:   "piid_3",
		ProcessDefinitionId: "pdid_3",
		WorkerId:            "w",
		ErrorMessage:        "error message",
		Time:                time.Now(),
	}

	incident4 := messages.IncidentMessage{
		Id:                  "foo_id_4",
		MsgVersion:          1,
		ExternalTaskId:      "task_id_3",
		ProcessInstanceId:   "piid_3",
		ProcessDefinitionId: "pdid_3",
		WorkerId:            "w",
		ErrorMessage:        "error message",
		Time:                time.Now(),
	}

	incident5 := messages.IncidentMessage{
		Id:                  "foo_id_5",
		MsgVersion:          1,
		ExternalTaskId:      "task_id_5",
		ProcessInstanceId:   "piid_5",
		ProcessDefinitionId: "pdid_5",
		WorkerId:            "w",
		ErrorMessage:        "error message",
		Time:                time.Now(),
	}

	t.Run("create incidents", func(t *testing.T) {
		createTestIncidents(t, config, []messages.IncidentMessage{incident1, incident2, incident3, incident4, incident5})
	})

	t.Run("by id", func(t *testing.T) {
		checkIncidentById(t, config, "foo_id_1", incident1)
	})

	t.Run("by task id", func(t *testing.T) {
		checkIncidentsByTaskId(t, config, "foobar", []messages.IncidentMessage{})
	})

	t.Run("by task id", func(t *testing.T) {
		checkIncidentsByTaskId(t, config, "task_id_1", []messages.IncidentMessage{incident1})
	})

	t.Run("by task id", func(t *testing.T) {
		checkIncidentsByTaskId(t, config, "task_id_3", []messages.IncidentMessage{incident3, incident4})
	})

	t.Run("by piid", func(t *testing.T) {
		checkIncidentsByPiid(t, config, "foobar", []messages.IncidentMessage{})
	})

	t.Run("by piid", func(t *testing.T) {
		checkIncidentsByPiid(t, config, "piid_1", []messages.IncidentMessage{incident1, incident2})
	})

	t.Run("by pdid", func(t *testing.T) {
		checkIncidentsByPdid(t, config, "pdid_1", []messages.IncidentMessage{incident1, incident2})
	})

	t.Run("by pdid", func(t *testing.T) {
		checkIncidentsByPdid(t, config, "foobar", []messages.IncidentMessage{})
	})
}
