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
	"github.com/SENERGY-Platform/process-incident-api/lib"
	"github.com/SENERGY-Platform/process-incident-api/lib/api"
	"github.com/SENERGY-Platform/process-incident-api/lib/camunda"
	"github.com/SENERGY-Platform/process-incident-api/lib/configuration"
	"github.com/SENERGY-Platform/process-incident-api/lib/database"
	"github.com/SENERGY-Platform/process-incident-api/lib/messages"
	"github.com/SENERGY-Platform/process-incident-api/tests/server"
	"sync"
	"testing"
	"time"
)

const UserToken = `Bearer eyJhbGciOiJub25lIn0.eyJzdWIiOiJ1c2VyIiwibmFtZSI6InVzZXIiLCJhZG1pbiI6dHJ1ZSwiaWF0IjoxNzM2MjkyMTI0fQ.`
const UserId = "user"

func Test(t *testing.T) {
	wg := &sync.WaitGroup{}
	defer wg.Wait()
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	defaultConfig, err := configuration.LoadConfig("../config.json")
	if err != nil {
		t.Error(err)
		return
	}

	config, err := server.New(ctx, wg, defaultConfig)
	if err != nil {
		t.Error(err)
		return
	}

	err = lib.StartWith(ctx, config, api.Factory, database.Factory, camunda.Factory)
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
		TenantId:            "user",
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
		TenantId:            "user",
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
		TenantId:            "user",
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
		TenantId:            "user",
	}

	incident5 := messages.IncidentMessage{
		Id:                  "foo_id_5",
		MsgVersion:          1,
		ExternalTaskId:      "task_id_5",
		ProcessInstanceId:   "a",
		ProcessDefinitionId: "x",
		WorkerId:            "w",
		ErrorMessage:        "error message",
		Time:                time.Now(),
		TenantId:            "user",
	}

	t.Run("create incidents", func(t *testing.T) {
		createTestIncidents(t, config, []messages.IncidentMessage{incident1, incident2, incident3, incident4, incident5})
	})

	t.Run("by id", func(t *testing.T) {
		checkIncidentById(t, config, "foo_id_1", UserToken, incident1)
	})

	t.Run("by task id", func(t *testing.T) {
		checkIncidentsByTaskId(t, config, "foobar", UserToken, []messages.IncidentMessage{})
	})

	t.Run("by task id", func(t *testing.T) {
		checkIncidentsByTaskId(t, config, "task_id_1", UserToken, []messages.IncidentMessage{incident1})
	})

	t.Run("by task id", func(t *testing.T) {
		checkIncidentsByTaskId(t, config, "task_id_3", UserToken, []messages.IncidentMessage{incident3, incident4})
	})

	t.Run("by piid", func(t *testing.T) {
		checkIncidentsByPiid(t, config, "foobar", UserToken, []messages.IncidentMessage{})
	})

	t.Run("by piid", func(t *testing.T) {
		checkIncidentsByPiid(t, config, "piid_1", UserToken, []messages.IncidentMessage{incident1, incident2})
	})

	t.Run("by pdid", func(t *testing.T) {
		checkIncidentsByPdid(t, config, "pdid_1", UserToken, []messages.IncidentMessage{incident1, incident2})
	})

	t.Run("by pdid", func(t *testing.T) {
		checkIncidentsByPdid(t, config, "foobar", UserToken, []messages.IncidentMessage{})
	})

	t.Run("limit offset", func(t *testing.T) {
		checkApiLimitAndSort(t, config, "2", "1", "id", UserToken, []messages.IncidentMessage{incident2, incident3})
	})

	t.Run("limit offset", func(t *testing.T) {
		checkApiLimitAndSort(t, config, "2", "1", "id.asc", UserToken, []messages.IncidentMessage{incident2, incident3})
	})

	t.Run("limit offset", func(t *testing.T) {
		checkApiLimitAndSort(t, config, "2", "1", "id.desc", UserToken, []messages.IncidentMessage{incident4, incident3})
	})

	t.Run("sort", func(t *testing.T) {
		checkApiLimitAndSort(t, config, "100", "0", "id", UserToken, []messages.IncidentMessage{incident1, incident2, incident3, incident4, incident5})
	})
	t.Run("sort", func(t *testing.T) {
		checkApiLimitAndSort(t, config, "100", "0", "external_task_id", UserToken, []messages.IncidentMessage{incident1, incident2, incident3, incident4, incident5})
	})
	t.Run("sort", func(t *testing.T) {
		checkApiLimitAndSort(t, config, "100", "0", "process_instance_id", UserToken, []messages.IncidentMessage{incident5, incident1, incident2, incident3, incident4})
	})
	t.Run("sort", func(t *testing.T) {
		checkApiLimitAndSort(t, config, "100", "0", "process_definition_id", UserToken, []messages.IncidentMessage{incident1, incident2, incident3, incident4, incident5})
	})

	t.Run("sort", func(t *testing.T) {
		checkApiLimitAndSort(t, config, "100", "0", "id.desc", UserToken, []messages.IncidentMessage{incident5, incident4, incident3, incident2, incident1})
	})
}

func TestTimeSort(t *testing.T) {
	wg := &sync.WaitGroup{}
	defer wg.Wait()
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	defaultConfig, err := configuration.LoadConfig("../config.json")
	if err != nil {
		t.Error(err)
		return
	}

	config, err := server.New(ctx, wg, defaultConfig)
	if err != nil {
		t.Error(err)
		return
	}

	err = lib.StartWith(ctx, config, api.Factory, database.Factory, camunda.Factory)
	if err != nil {
		t.Error(err)
		return
	}

	now, _ := time.Parse(time.RFC3339, "2020-01-10T12:00:00Z00:00")

	incident1 := messages.IncidentMessage{
		Id:                  "a",
		MsgVersion:          2,
		ExternalTaskId:      "task_id_1",
		ProcessInstanceId:   "piid_1",
		ProcessDefinitionId: "pdid_1",
		WorkerId:            "w",
		ErrorMessage:        "error message",
		Time:                now,
		TenantId:            "user",
	}

	incident2 := messages.IncidentMessage{
		Id:                  "b",
		MsgVersion:          2,
		ExternalTaskId:      "task_id_1",
		ProcessInstanceId:   "piid_1",
		ProcessDefinitionId: "pdid_1",
		WorkerId:            "w",
		ErrorMessage:        "error message",
		Time:                now.Add(3 * time.Hour),
		TenantId:            "user",
	}

	incident3 := messages.IncidentMessage{
		Id:                  "c",
		MsgVersion:          2,
		ExternalTaskId:      "task_id_1",
		ProcessInstanceId:   "piid_1",
		ProcessDefinitionId: "pdid_1",
		WorkerId:            "w",
		ErrorMessage:        "error message",
		Time:                now.Add(1 * time.Hour),
		TenantId:            "user",
	}

	t.Run("create incidents", func(t *testing.T) {
		createTestIncidents(t, config, []messages.IncidentMessage{incident1, incident2, incident3})
	})
	t.Run("sort time", func(t *testing.T) {
		checkApiLimitAndSort(t, config, "100", "0", "time", UserToken, []messages.IncidentMessage{incident1, incident3, incident2})
	})
	t.Run("sort time.asc", func(t *testing.T) {
		checkApiLimitAndSort(t, config, "100", "0", "time.asc", UserToken, []messages.IncidentMessage{incident1, incident3, incident2})
	})
	t.Run("sort time.desc", func(t *testing.T) {
		checkApiLimitAndSort(t, config, "100", "0", "time.desc", UserToken, []messages.IncidentMessage{incident2, incident3, incident1})
	})
}
