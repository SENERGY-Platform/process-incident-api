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
	"github.com/SENERGY-Platform/process-incident-api/lib/client"
	"github.com/SENERGY-Platform/process-incident-api/lib/configuration"
	"github.com/SENERGY-Platform/process-incident-api/lib/database"
	"github.com/SENERGY-Platform/process-incident-api/lib/messages"
	"github.com/SENERGY-Platform/process-incident-api/tests/server"
	"sync"
	"testing"
	"time"
)

func TestDatabaseDeprecated(t *testing.T) {
	wg := &sync.WaitGroup{}
	defer wg.Wait()
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	defaultConfig, err := configuration.LoadConfig("../config.json")
	if err != nil {
		t.Error(err)
		return
	}
	defaultConfig.Debug = true

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

	incident := messages.Incident{
		Id:                  "foo_id",
		MsgVersion:          2,
		ExternalTaskId:      "task_id",
		ProcessInstanceId:   "piid",
		ProcessDefinitionId: "pdid",
		WorkerId:            "w",
		ErrorMessage:        "error message",
		Time:                time.Now(),
		DeploymentName:      "pdid",
		TenantId:            UserId,
	}

	t.Run("send incident", func(t *testing.T) {
		c := client.New("http://localhost:" + config.ApiPort)
		err, _ = c.CreateIncident(client.InternalAdminToken, incident)
		if err != nil {
			t.Error(err)
			return
		}
		//sendIncidentToKafka(t, config, incident)
	})

	t.Run("check database", func(t *testing.T) {
		checkIncidentInDatabase(t, config, incident)
	})
}

func TestDatabase(t *testing.T) {
	wg := &sync.WaitGroup{}
	defer wg.Wait()
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	defaultConfig, err := configuration.LoadConfig("../config.json")
	if err != nil {
		t.Error(err)
		return
	}
	defaultConfig.Debug = true

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

	incident := messages.Incident{
		MsgVersion:          3,
		Id:                  "foo_id",
		ExternalTaskId:      "task_id",
		ProcessInstanceId:   "piid",
		ProcessDefinitionId: "pdid",
		WorkerId:            "w",
		ErrorMessage:        "error message",
		Time:                time.Now(),
		DeploymentName:      "pdid",
		TenantId:            UserId,
	}

	t.Run("send incident", func(t *testing.T) {
		c := client.New("http://localhost:" + config.ApiPort)
		err, _ = c.CreateIncident(client.InternalAdminToken, incident)
		if err != nil {
			t.Error(err)
			return
		}
		//sendIncidentV3ToKafka(t, config, incident)
	})

	t.Run("check database", func(t *testing.T) {
		checkIncidentInDatabase(t, config, incident)
	})
}

func TestCamunda(t *testing.T) {
	wg := &sync.WaitGroup{}
	defer wg.Wait()
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	defaultConfig, err := configuration.LoadConfig("../config.json")
	if err != nil {
		t.Error(err)
		return
	}
	defaultConfig.Debug = true

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

	definitionId := ""
	t.Run("deploy process", func(t *testing.T) {
		definitionId = deployProcess(t, config)
	})

	time.Sleep(10 * time.Second)

	instanceId := ""
	t.Run("start process", func(t *testing.T) {
		instanceId = startProcess(t, config, definitionId)
	})

	t.Run("check process", func(t *testing.T) {
		checkProcess(t, config, instanceId, true)
	})

	incident := messages.Incident{
		MsgVersion:          3,
		Id:                  "foo_id",
		ExternalTaskId:      "task_id",
		ProcessInstanceId:   instanceId,
		ProcessDefinitionId: definitionId,
		WorkerId:            "w",
		ErrorMessage:        "error message",
		Time:                time.Now(),
		TenantId:            UserId,
	}

	t.Run("send incident", func(t *testing.T) {
		c := client.New("http://localhost:" + config.ApiPort)
		err, _ = c.CreateIncident(client.InternalAdminToken, incident)
		if err != nil {
			t.Error(err)
			return
		}
		//sendIncidentV3ToKafka(t, config, incident)
	})

	incident.DeploymentName = "test"
	t.Run("check database", func(t *testing.T) {
		incident.MsgVersion = 3
		checkIncidentInDatabase(t, config, incident)
	})

	t.Run("check process", func(t *testing.T) {
		checkProcess(t, config, instanceId, false)
	})
}

func TestCamundaDeprecated(t *testing.T) {
	wg := &sync.WaitGroup{}
	defer wg.Wait()
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	defaultConfig, err := configuration.LoadConfig("../config.json")
	if err != nil {
		t.Error(err)
		return
	}
	defaultConfig.Debug = true

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

	definitionId := ""
	t.Run("deploy process", func(t *testing.T) {
		definitionId = deployProcess(t, config)
	})

	time.Sleep(10 * time.Second)

	instanceId := ""
	t.Run("start process", func(t *testing.T) {
		instanceId = startProcess(t, config, definitionId)
	})

	t.Run("check process", func(t *testing.T) {
		checkProcess(t, config, instanceId, true)
	})

	incident := messages.Incident{
		Id:                  "foo_id",
		MsgVersion:          2,
		ExternalTaskId:      "task_id",
		ProcessInstanceId:   instanceId,
		ProcessDefinitionId: definitionId,
		WorkerId:            "w",
		ErrorMessage:        "error message",
		Time:                time.Now(),
		TenantId:            UserId,
	}

	t.Run("send incident", func(t *testing.T) {
		c := client.New("http://localhost:" + config.ApiPort)
		err, _ = c.CreateIncident(client.InternalAdminToken, incident)
		if err != nil {
			t.Error(err)
			return
		}
		//sendIncidentToKafka(t, config, incident)
	})

	incident.DeploymentName = "test"
	t.Run("check database", func(t *testing.T) {
		checkIncidentInDatabase(t, config, incident)
	})

	t.Run("check process", func(t *testing.T) {
		checkProcess(t, config, instanceId, false)
	})
}

func TestDeleteByDeploymentIdDeprecated(t *testing.T) {
	wg := &sync.WaitGroup{}
	defer wg.Wait()
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	defaultConfig, err := configuration.LoadConfig("../config.json")
	if err != nil {
		t.Error(err)
		return
	}
	defaultConfig.Debug = true

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

	incident11 := messages.Incident{
		Id:                  "a",
		MsgVersion:          2,
		ExternalTaskId:      "task_id",
		ProcessInstanceId:   "piid1",
		ProcessDefinitionId: "pdid1",
		WorkerId:            "w",
		ErrorMessage:        "error message",
		Time:                time.Time{},
		DeploymentName:      "pdid1",
		TenantId:            UserId,
	}
	incident12 := messages.Incident{
		Id:                  "b",
		MsgVersion:          2,
		ExternalTaskId:      "task_id",
		ProcessInstanceId:   "piid1",
		ProcessDefinitionId: "pdid2",
		WorkerId:            "w",
		ErrorMessage:        "error message",
		Time:                time.Time{},
		DeploymentName:      "pdid2",
		TenantId:            UserId,
	}
	incident21 := messages.Incident{
		Id:                  "c",
		MsgVersion:          2,
		ExternalTaskId:      "task_id",
		ProcessInstanceId:   "piid2",
		ProcessDefinitionId: "pdid1",
		WorkerId:            "w",
		ErrorMessage:        "error message",
		Time:                time.Time{},
		DeploymentName:      "pdid1",
		TenantId:            UserId,
	}
	incident22 := messages.Incident{
		Id:                  "d",
		MsgVersion:          2,
		ExternalTaskId:      "task_id",
		ProcessInstanceId:   "piid2",
		ProcessDefinitionId: "pdid2",
		WorkerId:            "w",
		ErrorMessage:        "error message",
		Time:                time.Time{},
		DeploymentName:      "pdid2",
		TenantId:            UserId,
	}

	t.Run("send incidents", func(t *testing.T) {
		c := client.New("http://localhost:" + config.ApiPort)
		err, _ = c.CreateIncident(client.InternalAdminToken, incident11)
		if err != nil {
			t.Error(err)
			return
		}
		err, _ = c.CreateIncident(client.InternalAdminToken, incident12)
		if err != nil {
			t.Error(err)
			return
		}
		err, _ = c.CreateIncident(client.InternalAdminToken, incident21)
		if err != nil {
			t.Error(err)
			return
		}
		err, _ = c.CreateIncident(client.InternalAdminToken, incident22)
		if err != nil {
			t.Error(err)
			return
		}
		//sendIncidentToKafka(t, config, incident11)
		//sendIncidentToKafka(t, config, incident12)
		//sendIncidentToKafka(t, config, incident21)
		//sendIncidentToKafka(t, config, incident22)
	})

	t.Run("send delete by deplymentId", func(t *testing.T) {
		c := client.New("http://localhost:" + config.ApiPort)
		err, _ = c.DeleteIncidentByProcessDefinitionId(client.InternalAdminToken, "pdid1")
		if err != nil {
			t.Error(err)
			return
		}
		//sendDefinitionDeleteToKafka(t, config, "pdid1")
	})

	t.Run("check database", func(t *testing.T) {
		checkIncidentsInDatabase(t, config, incident12, incident22)
	})
}

func TestDeleteByDeploymentId(t *testing.T) {
	wg := &sync.WaitGroup{}
	defer wg.Wait()
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	defaultConfig, err := configuration.LoadConfig("../config.json")
	if err != nil {
		t.Error(err)
		return
	}
	defaultConfig.Debug = true

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

	t.Run("send incidents handler", func(t *testing.T) {
		c := client.New("http://localhost:" + config.ApiPort)
		err, _ = c.SetOnIncidentHandler(client.InternalAdminToken, client.OnIncident{
			ProcessDefinitionId: "pdid1",
		})
		if err != nil {
			t.Error(err)
			return
		}
		err, _ = c.SetOnIncidentHandler(client.InternalAdminToken, client.OnIncident{
			ProcessDefinitionId: "pdid2",
		})
		if err != nil {
			t.Error(err)
			return
		}
		//sendIncidentHandler(t, config, "pdid1")
		//sendIncidentHandler(t, config, "pdid2")
	})

	incident11 := messages.Incident{
		MsgVersion:          3,
		Id:                  "a",
		ExternalTaskId:      "task_id",
		ProcessInstanceId:   "piid1",
		ProcessDefinitionId: "pdid1",
		WorkerId:            "w",
		ErrorMessage:        "error message",
		Time:                time.Time{},
		DeploymentName:      "pdid1",
		TenantId:            UserId,
	}
	incident12 := messages.Incident{
		MsgVersion:          3,
		Id:                  "b",
		ExternalTaskId:      "task_id",
		ProcessInstanceId:   "piid1",
		ProcessDefinitionId: "pdid2",
		WorkerId:            "w",
		ErrorMessage:        "error message",
		Time:                time.Time{},
		DeploymentName:      "pdid2",
		TenantId:            UserId,
	}
	incident21 := messages.Incident{
		MsgVersion:          3,
		Id:                  "c",
		ExternalTaskId:      "task_id",
		ProcessInstanceId:   "piid2",
		ProcessDefinitionId: "pdid1",
		WorkerId:            "w",
		ErrorMessage:        "error message",
		Time:                time.Time{},
		DeploymentName:      "pdid1",
		TenantId:            UserId,
	}
	incident22 := messages.Incident{
		MsgVersion:          3,
		Id:                  "d",
		ExternalTaskId:      "task_id",
		ProcessInstanceId:   "piid2",
		ProcessDefinitionId: "pdid2",
		WorkerId:            "w",
		ErrorMessage:        "error message",
		Time:                time.Time{},
		DeploymentName:      "pdid2",
		TenantId:            UserId,
	}

	t.Run("send incidents", func(t *testing.T) {
		c := client.New("http://localhost:" + config.ApiPort)
		err, _ = c.CreateIncident(client.InternalAdminToken, incident11)
		if err != nil {
			t.Error(err)
			return
		}
		err, _ = c.CreateIncident(client.InternalAdminToken, incident12)
		if err != nil {
			t.Error(err)
			return
		}
		err, _ = c.CreateIncident(client.InternalAdminToken, incident21)
		if err != nil {
			t.Error(err)
			return
		}
		err, _ = c.CreateIncident(client.InternalAdminToken, incident22)
		if err != nil {
			t.Error(err)
			return
		}
		//sendIncidentV3ToKafka(t, config, incident11)
		//sendIncidentV3ToKafka(t, config, incident12)
		//sendIncidentV3ToKafka(t, config, incident21)
		//sendIncidentV3ToKafka(t, config, incident22)
	})

	t.Run("send delete by deplymentId", func(t *testing.T) {
		c := client.New("http://localhost:" + config.ApiPort)
		err, _ = c.DeleteIncidentByProcessDefinitionId(client.InternalAdminToken, "pdid1")
		if err != nil {
			t.Error(err)
			return
		}
		//sendDefinitionDeleteToKafka(t, config, "pdid1")
	})

	t.Run("check database", func(t *testing.T) {
		incident11.MsgVersion = 3
		incident12.MsgVersion = 3
		incident21.MsgVersion = 3
		incident22.MsgVersion = 3
		checkIncidentsInDatabase(t, config, incident12, incident22)
	})

	t.Run("check on incidents handler in database", func(t *testing.T) {
		checkOnIncidentsInDatabase(t, config, messages.OnIncident{
			ProcessDefinitionId: "pdid2",
			Restart:             false,
			Notify:              false,
		})
	})
}

func TestDeleteByInstanceIdDeprecated(t *testing.T) {
	wg := &sync.WaitGroup{}
	defer wg.Wait()
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	defaultConfig, err := configuration.LoadConfig("../config.json")
	if err != nil {
		t.Error(err)
		return
	}
	defaultConfig.Debug = true

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

	incident11 := messages.Incident{
		Id:                  "a",
		MsgVersion:          2,
		ExternalTaskId:      "task_id",
		ProcessInstanceId:   "piid1",
		ProcessDefinitionId: "pdid1",
		WorkerId:            "w",
		ErrorMessage:        "error message",
		Time:                time.Time{},
		DeploymentName:      "pdid1",
		TenantId:            UserId,
	}
	incident12 := messages.Incident{
		Id:                  "b",
		MsgVersion:          2,
		ExternalTaskId:      "task_id",
		ProcessInstanceId:   "piid1",
		ProcessDefinitionId: "pdid2",
		WorkerId:            "w",
		ErrorMessage:        "error message",
		Time:                time.Time{},
		DeploymentName:      "pdid2",
		TenantId:            UserId,
	}
	incident21 := messages.Incident{
		Id:                  "c",
		MsgVersion:          2,
		ExternalTaskId:      "task_id",
		ProcessInstanceId:   "piid2",
		ProcessDefinitionId: "pdid1",
		WorkerId:            "w",
		ErrorMessage:        "error message",
		Time:                time.Time{},
		DeploymentName:      "pdid1",
		TenantId:            UserId,
	}
	incident22 := messages.Incident{
		Id:                  "d",
		MsgVersion:          2,
		ExternalTaskId:      "task_id",
		ProcessInstanceId:   "piid2",
		ProcessDefinitionId: "pdid2",
		WorkerId:            "w",
		ErrorMessage:        "error message",
		Time:                time.Time{},
		DeploymentName:      "pdid2",
		TenantId:            UserId,
	}

	t.Run("send incidents", func(t *testing.T) {
		c := client.New("http://localhost:" + config.ApiPort)
		err, _ = c.CreateIncident(client.InternalAdminToken, incident11)
		if err != nil {
			t.Error(err)
			return
		}
		err, _ = c.CreateIncident(client.InternalAdminToken, incident12)
		if err != nil {
			t.Error(err)
			return
		}
		err, _ = c.CreateIncident(client.InternalAdminToken, incident21)
		if err != nil {
			t.Error(err)
			return
		}
		err, _ = c.CreateIncident(client.InternalAdminToken, incident22)
		if err != nil {
			t.Error(err)
			return
		}
		//sendIncidentToKafka(t, config, incident11)
		//sendIncidentToKafka(t, config, incident12)
		//sendIncidentToKafka(t, config, incident21)
		//sendIncidentToKafka(t, config, incident22)
	})

	t.Run("send delete by instance", func(t *testing.T) {
		c := client.New("http://localhost:" + config.ApiPort)
		err, _ = c.DeleteIncidentByProcessInstanceId(client.InternalAdminToken, "piid1")
		if err != nil {
			t.Error(err)
			return
		}
		//sendInstanceDeleteToKafka(t, config, "piid1")
	})

	t.Run("check database", func(t *testing.T) {
		checkIncidentsInDatabase(t, config, incident21, incident22)
	})
}

func TestDeleteByInstanceId(t *testing.T) {
	wg := &sync.WaitGroup{}
	defer wg.Wait()
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	defaultConfig, err := configuration.LoadConfig("../config.json")
	if err != nil {
		t.Error(err)
		return
	}
	defaultConfig.Debug = true

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

	incident11 := messages.Incident{
		MsgVersion:          3,
		Id:                  "a",
		ExternalTaskId:      "task_id",
		ProcessInstanceId:   "piid1",
		ProcessDefinitionId: "pdid1",
		WorkerId:            "w",
		ErrorMessage:        "error message",
		Time:                time.Time{},
		DeploymentName:      "pdid1",
		TenantId:            UserId,
	}
	incident12 := messages.Incident{
		MsgVersion:          3,
		Id:                  "b",
		ExternalTaskId:      "task_id",
		ProcessInstanceId:   "piid1",
		ProcessDefinitionId: "pdid2",
		WorkerId:            "w",
		ErrorMessage:        "error message",
		Time:                time.Time{},
		DeploymentName:      "pdid2",
		TenantId:            UserId,
	}
	incident21 := messages.Incident{
		MsgVersion:          3,
		Id:                  "c",
		ExternalTaskId:      "task_id",
		ProcessInstanceId:   "piid2",
		ProcessDefinitionId: "pdid1",
		WorkerId:            "w",
		ErrorMessage:        "error message",
		Time:                time.Time{},
		DeploymentName:      "pdid1",
		TenantId:            UserId,
	}
	incident22 := messages.Incident{
		MsgVersion:          3,
		Id:                  "d",
		ExternalTaskId:      "task_id",
		ProcessInstanceId:   "piid2",
		ProcessDefinitionId: "pdid2",
		WorkerId:            "w",
		ErrorMessage:        "error message",
		Time:                time.Time{},
		DeploymentName:      "pdid2",
		TenantId:            UserId,
	}

	t.Run("send incidents", func(t *testing.T) {
		c := client.New("http://localhost:" + config.ApiPort)
		err, _ = c.CreateIncident(client.InternalAdminToken, incident11)
		if err != nil {
			t.Error(err)
			return
		}
		err, _ = c.CreateIncident(client.InternalAdminToken, incident12)
		if err != nil {
			t.Error(err)
			return
		}
		err, _ = c.CreateIncident(client.InternalAdminToken, incident21)
		if err != nil {
			t.Error(err)
			return
		}
		err, _ = c.CreateIncident(client.InternalAdminToken, incident22)
		if err != nil {
			t.Error(err)
			return
		}
		//sendIncidentV3ToKafka(t, config, incident11)
		//sendIncidentV3ToKafka(t, config, incident12)
		//sendIncidentV3ToKafka(t, config, incident21)
		//sendIncidentV3ToKafka(t, config, incident22)
	})

	t.Run("send delete by instance", func(t *testing.T) {
		c := client.New("http://localhost:" + config.ApiPort)
		err, _ = c.DeleteIncidentByProcessInstanceId(client.InternalAdminToken, "piid1")
		if err != nil {
			t.Error(err)
			return
		}
		//sendInstanceDeleteToKafka(t, config, "piid1")
	})

	t.Run("check database", func(t *testing.T) {
		incident11.MsgVersion = 3
		incident12.MsgVersion = 3
		incident21.MsgVersion = 3
		incident22.MsgVersion = 3
		checkIncidentsInDatabase(t, config, incident21, incident22)
	})
}
