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

package camunda

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"

	"github.com/SENERGY-Platform/process-incident-api/lib/configuration"
	"github.com/SENERGY-Platform/process-incident-api/lib/messages"
)

const (
	testDefinitionId = "def:1:abc"
	testInstanceId   = "instance-1"
	// 2^53+1 is not representable as float64; it must reach the engine unchanged
	testLong = "9007199254740993"
)

type fakeEngine struct {
	mux      sync.Mutex
	calls    map[string]int
	bodies   []string
	handlers map[string]func(w http.ResponseWriter, r *http.Request)
}

func newFakeEngine(t *testing.T, handlers map[string]func(w http.ResponseWriter, r *http.Request)) (*fakeEngine, string) {
	engine := &fakeEngine{calls: map[string]int{}, handlers: handlers}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		engine.mux.Lock()
		engine.calls[r.URL.Path]++
		engine.bodies = append(engine.bodies, string(body))
		engine.mux.Unlock()
		handler, ok := engine.handlers[r.URL.Path]
		if !ok {
			t.Errorf("unexpected request %v %v", r.Method, r.URL.String())
			w.WriteHeader(http.StatusNotImplemented)
			return
		}
		handler(w, r)
	}))
	t.Cleanup(server.Close)
	return engine, server.URL
}

func (this *fakeEngine) count(path string) int {
	this.mux.Lock()
	defer this.mux.Unlock()
	return this.calls[path]
}

func respond(status int, body string) func(w http.ResponseWriter, r *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(status)
		_, _ = w.Write([]byte(body))
	}
}

const (
	formVariablesPath    = "/engine-rest/process-definition/" + testDefinitionId + "/form-variables"
	instanceVariablePath = "/engine-rest/process-instance/" + testInstanceId + "/variables"
	historyVariablePath  = "/engine-rest/history/variable-instance"
	submitFormPath       = "/engine-rest/process-definition/" + testDefinitionId + "/submit-form"
)

const formVariables = `{
	"count": {"value": null, "type": "Long", "valueInfo": {}},
	"label": {"value": null, "type": "String", "valueInfo": {}},
	"unused": {"value": null, "type": "String", "valueInfo": {}}
}`

func testCamunda() *Camunda {
	return &Camunda{config: configuration.Config{}}
}

func assertVariables(t *testing.T, actual map[string]messages.Variable, expected map[string]string) {
	t.Helper()
	if len(actual) != len(expected) {
		t.Fatalf("expected %v variables, got %#v", len(expected), actual)
	}
	for name, expectedJson := range expected {
		variable, ok := actual[name]
		if !ok {
			t.Fatalf("missing variable %v in %#v", name, actual)
		}
		actualJson, err := json.Marshal(variable)
		if err != nil {
			t.Fatal(err)
		}
		if string(actualJson) != expectedJson {
			t.Errorf("variable %v:\n got  %v\n want %v", name, string(actualJson), expectedJson)
		}
	}
}

func TestCreateStartMessage(t *testing.T) {
	variables := map[string]messages.Variable{
		"count":  {Value: json.RawMessage(testLong), Type: "Long", ValueInfo: json.RawMessage(`{}`)},
		"obj":    {Value: json.RawMessage(`"{\"a\":1}"`), Type: "Object", ValueInfo: json.RawMessage(`{"objectTypeName":"java.util.HashMap","serializationDataFormat":"application/json"}`)},
		"absent": {Value: json.RawMessage(`null`), Type: "Null"},
	}
	cases := []struct {
		name        string
		variables   map[string]messages.Variable
		businessKey string
		expected    string
	}{
		{name: "nothing", expected: `{}`},
		{name: "business key only", businessKey: "bk", expected: `{"businessKey":"bk"}`},
		{name: "empty variables", variables: map[string]messages.Variable{}, businessKey: "bk", expected: `{"businessKey":"bk"}`},
		{
			name:        "variables",
			variables:   variables,
			businessKey: "bk",
			expected:    `{"businessKey":"bk","variables":{"absent":{"value":null,"type":"Null"},"count":{"value":` + testLong + `,"type":"Long","valueInfo":{}},"obj":{"value":"{\"a\":1}","type":"Object","valueInfo":{"objectTypeName":"java.util.HashMap","serializationDataFormat":"application/json"}}}}`,
		},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			actual, err := json.Marshal(createStartMessage(c.variables, c.businessKey))
			if err != nil {
				t.Fatal(err)
			}
			if string(actual) != c.expected {
				t.Errorf("\n got  %v\n want %v", string(actual), c.expected)
			}
		})
	}
}

func TestStartProcessSendsVariablesUnchanged(t *testing.T) {
	engine, shard := newFakeEngine(t, map[string]func(w http.ResponseWriter, r *http.Request){
		submitFormPath: respond(http.StatusOK, `{"id":"new"}`),
	})
	err := testCamunda().startProcess(shard, testDefinitionId, "bk", map[string]messages.Variable{
		"count": {Value: json.RawMessage(testLong), Type: "Long", ValueInfo: json.RawMessage(`{}`)},
	})
	if err != nil {
		t.Fatal(err)
	}
	if engine.count(submitFormPath) != 1 {
		t.Fatalf("expected one submit-form call, got %v", engine.calls)
	}
	expected := `{"businessKey":"bk","variables":{"count":{"value":` + testLong + `,"type":"Long","valueInfo":{}}}}`
	if strings.TrimSpace(engine.bodies[0]) != expected {
		t.Errorf("\n got  %v\n want %v", engine.bodies[0], expected)
	}
}

func TestStartProcessReportsEngineError(t *testing.T) {
	_, shard := newFakeEngine(t, map[string]func(w http.ResponseWriter, r *http.Request){
		submitFormPath: respond(http.StatusBadRequest, `{"message":"invalid"}`),
	})
	err := testCamunda().startProcess(shard, testDefinitionId, "bk", nil)
	if err == nil || !strings.Contains(err.Error(), "invalid") {
		t.Fatalf("expected engine error, got %v", err)
	}
}

func TestGetStartVariablesWithoutFormVariables(t *testing.T) {
	engine, shard := newFakeEngine(t, map[string]func(w http.ResponseWriter, r *http.Request){
		formVariablesPath: respond(http.StatusOK, `{}`),
	})
	result, err := testCamunda().getStartVariables(shard, testDefinitionId, testInstanceId)
	if err != nil {
		t.Fatal(err)
	}
	if result != nil {
		t.Errorf("expected nil, got %#v", result)
	}
	if engine.count(instanceVariablePath) != 0 || engine.count(historyVariablePath) != 0 {
		t.Errorf("instance variables read although the definition has no form variables: %v", engine.calls)
	}
}

func TestGetStartVariablesFormVariablesUnreadable(t *testing.T) {
	_, shard := newFakeEngine(t, map[string]func(w http.ResponseWriter, r *http.Request){
		formVariablesPath: respond(http.StatusNotFound, `{"message":"no definition"}`),
	})
	_, err := testCamunda().getStartVariables(shard, testDefinitionId, testInstanceId)
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestGetStartVariablesFromRuntime(t *testing.T) {
	engine, shard := newFakeEngine(t, map[string]func(w http.ResponseWriter, r *http.Request){
		formVariablesPath: respond(http.StatusOK, formVariables),
		instanceVariablePath: func(w http.ResponseWriter, r *http.Request) {
			if r.URL.Query().Get("deserializeValues") != "false" {
				t.Errorf("expected deserializeValues=false, got %v", r.URL.RawQuery)
			}
			respond(http.StatusOK, `{
				"count": {"value": `+testLong+`, "type": "Long", "valueInfo": {}},
				"label": {"value": "Wohnzimmer \u00fc", "type": "String", "valueInfo": {}},
				"other": {"value": "not a form field", "type": "String", "valueInfo": {}}
			}`)(w, r)
		},
	})
	result, err := testCamunda().getStartVariables(shard, testDefinitionId, testInstanceId)
	if err != nil {
		t.Fatal(err)
	}
	assertVariables(t, result, map[string]string{
		"count": `{"value":` + testLong + `,"type":"Long","valueInfo":{}}`,
		"label": `{"value":"Wohnzimmer \u00fc","type":"String","valueInfo":{}}`,
	})
	if engine.count(historyVariablePath) != 0 {
		t.Errorf("history read although runtime answered")
	}
}

func TestGetStartVariablesFallsBackToHistory(t *testing.T) {
	for _, status := range []int{http.StatusNotFound, http.StatusInternalServerError} {
		t.Run(http.StatusText(status), func(t *testing.T) {
			engine, shard := newFakeEngine(t, map[string]func(w http.ResponseWriter, r *http.Request){
				formVariablesPath:    respond(http.StatusOK, formVariables),
				instanceVariablePath: respond(status, `{"type":"ProcessEngineException","message":"instance does not exist"}`),
				historyVariablePath: func(w http.ResponseWriter, r *http.Request) {
					query := r.URL.Query()
					if query.Get("processInstanceId") != testInstanceId || query.Get("deserializeValues") != "false" {
						t.Errorf("unexpected history query %v", r.URL.RawQuery)
					}
					respond(http.StatusOK, `[
						{"name": "label", "type": "String", "value": "local", "valueInfo": {}, "processInstanceId": "`+testInstanceId+`", "executionId": "sub-execution", "taskId": null},
						{"name": "count", "type": "Long", "value": `+testLong+`, "valueInfo": {}, "processInstanceId": "`+testInstanceId+`", "executionId": "`+testInstanceId+`", "taskId": null},
						{"name": "label", "type": "String", "value": "original", "valueInfo": {}, "processInstanceId": "`+testInstanceId+`", "executionId": "`+testInstanceId+`", "taskId": null},
						{"name": "unused", "type": "String", "value": "task local", "valueInfo": {}, "processInstanceId": "`+testInstanceId+`", "executionId": "`+testInstanceId+`", "taskId": "task-1"}
					]`)(w, r)
				},
			})
			result, err := testCamunda().getStartVariables(shard, testDefinitionId, testInstanceId)
			if err != nil {
				t.Fatal(err)
			}
			assertVariables(t, result, map[string]string{
				"count": `{"value":` + testLong + `,"type":"Long","valueInfo":{}}`,
				"label": `{"value":"original","type":"String","valueInfo":{}}`,
			})
			if engine.count(historyVariablePath) != 1 {
				t.Errorf("expected one history call, got %v", engine.calls)
			}
		})
	}
}

func TestGetStartVariablesUnreadable(t *testing.T) {
	cases := map[string]func(w http.ResponseWriter, r *http.Request){
		"history fails":       respond(http.StatusInternalServerError, `{"message":"history down"}`),
		"history empty":       respond(http.StatusOK, `[]`),
		"history not decoded": respond(http.StatusOK, `{"unexpected": true}`),
	}
	for name, history := range cases {
		t.Run(name, func(t *testing.T) {
			_, shard := newFakeEngine(t, map[string]func(w http.ResponseWriter, r *http.Request){
				formVariablesPath:    respond(http.StatusOK, formVariables),
				instanceVariablePath: respond(http.StatusInternalServerError, `{"message":"runtime down"}`),
				historyVariablePath:  history,
			})
			result, err := testCamunda().getStartVariables(shard, testDefinitionId, testInstanceId)
			if err == nil {
				t.Fatalf("expected error, got %#v", result)
			}
			if !strings.Contains(err.Error(), "runtime down") {
				t.Errorf("runtime error missing in %v", err)
			}
		})
	}
}
