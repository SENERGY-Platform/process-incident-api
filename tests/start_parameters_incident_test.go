/*
 * Copyright (c) 2026 InfAI (CC SES)
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
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"sync"
	"testing"
	"time"

	"github.com/SENERGY-Platform/process-incident-api/lib"
	"github.com/SENERGY-Platform/process-incident-api/lib/api"
	"github.com/SENERGY-Platform/process-incident-api/lib/camunda"
	"github.com/SENERGY-Platform/process-incident-api/lib/camunda/cache"
	"github.com/SENERGY-Platform/process-incident-api/lib/camunda/shards"
	"github.com/SENERGY-Platform/process-incident-api/lib/client"
	"github.com/SENERGY-Platform/process-incident-api/lib/configuration"
	"github.com/SENERGY-Platform/process-incident-api/lib/controller"
	"github.com/SENERGY-Platform/process-incident-api/lib/database"
	"github.com/SENERGY-Platform/process-incident-api/lib/messages"
	"github.com/SENERGY-Platform/process-incident-api/lib/metrics"
	"github.com/SENERGY-Platform/process-incident-api/tests/resources"
	"github.com/SENERGY-Platform/process-incident-api/tests/server"
	"github.com/SENERGY-Platform/process-incident-api/tests/server/docker"
)

const startParametersTopic = "start_parameters_test"

// 2^53+1 does not survive a float64 round trip
var startParameters = map[string]messages.Variable{
	"count": {Value: json.RawMessage(`9007199254740993`), Type: "Long"},
	"label": {Value: json.RawMessage(`"Wohnzimmer ü"`), Type: "String"},
	"since": {Value: json.RawMessage(`"2024-01-02T03:04:05.000+0000"`), Type: "Date"},
}

var startParameterNames = []string{"count", "label", "since", "unused"}

func TestRestartWithStartParameters(t *testing.T) {
	wg := &sync.WaitGroup{}
	defer wg.Wait()
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	defaultConfig, err := configuration.LoadConfig("../config.json")
	if err != nil {
		t.Fatal(err)
	}
	defaultConfig.Debug = true
	defaultConfig.MetricsPort, err = docker.GetFreePortStr()
	if err != nil {
		t.Fatal(err)
	}
	config, err := server.New(ctx, wg, defaultConfig)
	if err != nil {
		t.Fatal(err)
	}

	mux := sync.Mutex{}
	notifications := []string{}
	notificationServer := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		msg, _ := io.ReadAll(request.Body)
		mux.Lock()
		defer mux.Unlock()
		notifications = append(notifications, string(msg))
	}))
	defer notificationServer.Close()
	config.NotificationUrl = notificationServer.URL

	err = lib.StartWith(ctx, config, api.Factory, database.Factory, camunda.Factory)
	if err != nil {
		t.Fatal(err)
	}

	s, err := shards.New(config.ShardsDb, cache.None)
	if err != nil {
		t.Fatal(err)
	}
	shard, err := s.EnsureShardForUser("")
	if err != nil {
		t.Fatal(err)
	}
	camundaInstance, err := camunda.Factory.Get(ctx, config)
	if err != nil {
		t.Fatal(err)
	}

	definitionId, err := deployProcessWithInfo(config, "start_parameters", resources.StartParametersBpmn, resources.SvgExample, "testuser")
	if err != nil {
		t.Fatal(err)
	}

	t.Run("start variables of a stopped instance come from history or not at all", func(t *testing.T) {
		instanceId := submitStartForm(t, shard, definitionId, "bk-history", startParameters)
		expected := formScopeVariables(t, shard, instanceId)

		err = camundaInstance.StopProcessInstance(instanceId, "testuser")
		if err != nil {
			t.Fatal(err)
		}
		resp, err := http.Get(shard + "/engine-rest/process-instance/" + url.PathEscape(instanceId) + "/variables?deserializeValues=false")
		if err != nil {
			t.Fatal(err)
		}
		body, _ := io.ReadAll(resp.Body)
		resp.Body.Close()
		t.Log("runtime variables of a stopped instance:", resp.Status, string(body))
		if resp.StatusCode == http.StatusOK {
			t.Fatal("expected the runtime variables of a stopped instance to be unavailable")
		}

		history := []json.RawMessage{}
		engineRequest(t, "GET", shard+"/engine-rest/history/variable-instance?processInstanceId="+url.QueryEscape(instanceId), nil, http.StatusOK, &history)
		actual, err := camundaInstance.GetStartVariables(definitionId, instanceId, "testuser")
		if len(history) == 0 {
			// the platform engine runs with history level activity, which records no variables
			t.Log("engine records no historic variables")
			if err == nil {
				t.Fatalf("expected an error instead of guessed start variables, got %#v", actual)
			}
			return
		}
		if err != nil {
			t.Fatal(err)
		}
		assertSameVariables(t, actual, expected)
		assertStartParameters(t, actual)
	})

	t.Run("set incident handler", func(t *testing.T) {
		databaseInstance, err := database.Factory.Get(ctx, config)
		if err != nil {
			t.Fatal(err)
		}
		ctrl, err := controller.New(ctx, config, databaseInstance, camundaInstance, metrics.New())
		if err != nil {
			t.Fatal(err)
		}
		err, _ = ctrl.SetOnIncidentHandler(client.InternalAdminToken, messages.OnIncident{
			ProcessDefinitionId: definitionId,
			Restart:             true,
			Notify:              true,
		})
		if err != nil {
			t.Fatal(err)
		}
	})

	t.Run("restart after incident keeps start parameters", func(t *testing.T) {
		businessKey := "bk-restart"
		instanceId := submitStartForm(t, shard, definitionId, businessKey, startParameters)
		expected := formScopeVariables(t, shard, instanceId)
		failExternalTask(t, shard, instanceId)

		var restarted []messages.HistoricProcessInstance
		deadline := time.Now().Add(time.Minute)
		for time.Now().Before(deadline) {
			restarted = activeInstancesByBusinessKey(t, shard, businessKey)
			if len(restarted) > 0 && restarted[0].Id != instanceId {
				break
			}
			time.Sleep(time.Second)
		}
		if len(restarted) != 1 || restarted[0].Id == instanceId {
			mux.Lock()
			t.Log("notifications:", notifications)
			mux.Unlock()
			t.Fatalf("expected exactly one restarted active instance with business key %v, got %#v", businessKey, restarted)
		}
		if restarted[0].ProcessDefinitionId != definitionId {
			t.Errorf("restarted instance of definition %v", restarted[0].ProcessDefinitionId)
		}
		actual := formScopeVariables(t, shard, restarted[0].Id)
		assertSameVariables(t, actual, expected)
		assertStartParameters(t, actual)
	})
}

func engineRequest(t *testing.T, method string, endpoint string, body interface{}, expectedStatus int, result interface{}) {
	t.Helper()
	var reader io.Reader
	if body != nil {
		b, err := json.Marshal(body)
		if err != nil {
			t.Fatal(err)
		}
		reader = bytes.NewReader(b)
	}
	req, err := http.NewRequest(method, endpoint, reader)
	if err != nil {
		t.Fatal(err)
	}
	req.Header.Set("Content-Type", "application/json")
	resp, err := (&http.Client{Timeout: 10 * time.Second}).Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	payload, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != expectedStatus {
		t.Fatalf("%v %v: %v %v", method, endpoint, resp.Status, string(payload))
	}
	if result != nil {
		err = json.Unmarshal(payload, result)
		if err != nil {
			t.Fatal(err, string(payload))
		}
	}
}

func submitStartForm(t *testing.T, shard string, definitionId string, businessKey string, variables map[string]messages.Variable) string {
	t.Helper()
	result := struct {
		Id string `json:"id"`
	}{}
	engineRequest(t, "POST", shard+"/engine-rest/process-definition/"+url.PathEscape(definitionId)+"/submit-form",
		map[string]interface{}{"businessKey": businessKey, "variables": variables}, http.StatusOK, &result)
	if result.Id == "" {
		t.Fatal("missing process instance id")
	}
	return result.Id
}

// formScopeVariables returns the process instance scope variables named like the start form fields.
func formScopeVariables(t *testing.T, shard string, instanceId string) map[string]messages.Variable {
	t.Helper()
	all := map[string]messages.Variable{}
	engineRequest(t, "GET", shard+"/engine-rest/process-instance/"+url.PathEscape(instanceId)+"/variables?deserializeValues=false", nil, http.StatusOK, &all)
	result := map[string]messages.Variable{}
	for _, name := range startParameterNames {
		if variable, ok := all[name]; ok {
			result[name] = variable
		}
	}
	return result
}

func failExternalTask(t *testing.T, shard string, instanceId string) {
	t.Helper()
	tasks := []struct {
		Id                string `json:"id"`
		ProcessInstanceId string `json:"processInstanceId"`
	}{}
	engineRequest(t, "POST", shard+"/engine-rest/external-task/fetchAndLock", map[string]interface{}{
		"workerId": "test",
		"maxTasks": 10,
		"topics":   []map[string]interface{}{{"topicName": startParametersTopic, "lockDuration": 60000}},
	}, http.StatusOK, &tasks)
	for _, task := range tasks {
		if task.ProcessInstanceId != instanceId {
			continue
		}
		engineRequest(t, "POST", shard+"/engine-rest/external-task/"+url.PathEscape(task.Id)+"/failure", map[string]interface{}{
			"workerId":     "test",
			"errorMessage": "test failure",
			"retries":      0,
			"retryTimeout": 0,
		}, http.StatusNoContent, nil)
		return
	}
	t.Fatalf("no external task for %v in %#v", instanceId, tasks)
}

func activeInstancesByBusinessKey(t *testing.T, shard string, businessKey string) []messages.HistoricProcessInstance {
	t.Helper()
	result := []messages.HistoricProcessInstance{}
	query := url.Values{"processInstanceBusinessKey": {businessKey}, "active": {"true"}}
	engineRequest(t, "GET", shard+"/engine-rest/history/process-instance?"+query.Encode(), nil, http.StatusOK, &result)
	return result
}

func compactJson(t *testing.T, raw json.RawMessage) string {
	t.Helper()
	if len(raw) == 0 {
		return ""
	}
	b := new(bytes.Buffer)
	if err := json.Compact(b, raw); err != nil {
		t.Fatal(err)
	}
	return b.String()
}

func variableString(t *testing.T, variable messages.Variable) string {
	return fmt.Sprintf("%v %v %v", variable.Type, compactJson(t, variable.Value), compactJson(t, variable.ValueInfo))
}

func assertSameVariables(t *testing.T, actual map[string]messages.Variable, expected map[string]messages.Variable) {
	t.Helper()
	if len(actual) != len(expected) {
		t.Errorf("expected %#v, got %#v", expected, actual)
	}
	for name, expectedVariable := range expected {
		actualVariable, ok := actual[name]
		if !ok {
			t.Errorf("missing variable %v", name)
			continue
		}
		if variableString(t, actualVariable) != variableString(t, expectedVariable) {
			t.Errorf("variable %v: got %v, want %v", name, variableString(t, actualVariable), variableString(t, expectedVariable))
		}
	}
}

func assertStartParameters(t *testing.T, actual map[string]messages.Variable) {
	t.Helper()
	for name, expected := range startParameters {
		variable, ok := actual[name]
		if !ok {
			t.Errorf("missing start parameter %v", name)
			continue
		}
		if variable.Type != expected.Type || compactJson(t, variable.Value) != compactJson(t, expected.Value) {
			t.Errorf("start parameter %v: got %v %v, want %v %v", name, variable.Type, string(variable.Value), expected.Type, string(expected.Value))
		}
	}
	if _, ok := actual["unused"]; ok {
		t.Errorf("form field without submitted value was passed: %#v", actual["unused"])
	}
}
