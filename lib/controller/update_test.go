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

package controller

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"reflect"
	"strings"
	"sync"
	"testing"

	"github.com/SENERGY-Platform/process-incident-api/lib/configuration"
	"github.com/SENERGY-Platform/process-incident-api/lib/messages"
	"github.com/SENERGY-Platform/process-incident-api/lib/notification"
)

type camundaMock struct {
	mux               sync.Mutex
	calls             []string
	startVariables    map[string]messages.Variable
	startVariablesErr error
	businessKey       string
	startedWith       map[string]messages.Variable
	startedKey        string
}

func (this *camundaMock) record(call string) {
	this.mux.Lock()
	defer this.mux.Unlock()
	this.calls = append(this.calls, call)
}

func (this *camundaMock) StopProcessInstance(id string, tenantId string) error {
	this.record("stop")
	return nil
}

func (this *camundaMock) GetProcessName(id string, tenantId string) (string, error) {
	return "name", nil
}

func (this *camundaMock) GetStartVariables(processDefinitionId string, processInstanceId string, tenantId string) (map[string]messages.Variable, error) {
	this.record("read")
	return this.startVariables, this.startVariablesErr
}

func (this *camundaMock) StartProcess(processDefinitionId string, userId string, variables map[string]messages.Variable) error {
	this.record("start")
	this.startedWith = variables
	return nil
}

func (this *camundaMock) StartProcessWithBusinessKey(processDefinitionId string, businessKey string, userId string, variables map[string]messages.Variable) error {
	this.record("start-with-key")
	this.startedWith = variables
	this.startedKey = businessKey
	return nil
}

func (this *camundaMock) GetIncidents() ([]messages.CamundaIncident, error) {
	return nil, nil
}

func (this *camundaMock) GetHistoricProcessInstance(id string, userId string) (messages.HistoricProcessInstance, error) {
	return messages.HistoricProcessInstance{Id: id, BusinessKey: this.businessKey}, nil
}

type dbMock struct {
	onIncident    messages.OnIncident
	hasOnIncident bool
}

func (this *dbMock) GetIncidents(id string, user string) (messages.IncidentMessage, bool, error) {
	return messages.IncidentMessage{}, false, nil
}

func (this *dbMock) FindIncidents(externalTaskId string, processDefinitionId string, processInstanceId string, limit int, offset int, sortBy string, asc bool, user string) ([]messages.IncidentMessage, error) {
	return nil, nil
}

func (this *dbMock) DeleteByDefinitionId(id string) error { return nil }

func (this *dbMock) SaveIncident(incident messages.Incident) error { return nil }

func (this *dbMock) DeleteIncidentByInstanceId(id string) error { return nil }

func (this *dbMock) SaveOnIncident(handler messages.OnIncident) error { return nil }

func (this *dbMock) GetOnIncident(definitionId string) (messages.OnIncident, bool, error) {
	return this.onIncident, this.hasOnIncident, nil
}

type metricMock struct{}

func (metricMock) NotifyIncidentMessage() {}

func newTestController(t *testing.T, db *dbMock, camunda *camundaMock) (*Controller, func() []notification.Message) {
	mux := sync.Mutex{}
	notifications := []notification.Message{}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		msg := notification.Message{}
		if err := json.NewDecoder(r.Body).Decode(&msg); err != nil {
			t.Error(err)
		}
		mux.Lock()
		notifications = append(notifications, msg)
		mux.Unlock()
	}))
	t.Cleanup(server.Close)
	ctrl, err := New(context.Background(), configuration.Config{NotificationUrl: server.URL}, db, camunda, metricMock{})
	if err != nil {
		t.Fatal(err)
	}
	return ctrl, func() []notification.Message {
		mux.Lock()
		defer mux.Unlock()
		return append([]notification.Message{}, notifications...)
	}
}

var testIncident = messages.Incident{
	Id:                  "incident",
	ProcessDefinitionId: "definition",
	ProcessInstanceId:   "instance",
	TenantId:            "user",
	ErrorMessage:        "failure",
}

var testStartVariables = map[string]messages.Variable{
	"count": {Value: json.RawMessage(`9007199254740993`), Type: "Long", ValueInfo: json.RawMessage(`{}`)},
}

func restartErrors(notifications []notification.Message) []notification.Message {
	result := []notification.Message{}
	for _, msg := range notifications {
		if strings.HasPrefix(msg.Title, "ERROR: unable to restart process") {
			result = append(result, msg)
		}
	}
	return result
}

func TestRestartPassesStartVariables(t *testing.T) {
	cases := []struct {
		name          string
		incidentKey   string
		historicKey   string
		expectedCalls []string
		expectedKey   string
	}{
		{name: "business key from incident", incidentKey: "bk", expectedCalls: []string{"read", "stop", "start-with-key"}, expectedKey: "bk"},
		{name: "business key from history", historicKey: "historic-bk", expectedCalls: []string{"read", "stop", "start-with-key"}, expectedKey: "historic-bk"},
		{name: "without business key", expectedCalls: []string{"read", "stop", "start"}},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			camunda := &camundaMock{startVariables: testStartVariables, businessKey: c.historicKey}
			db := &dbMock{onIncident: messages.OnIncident{ProcessDefinitionId: "definition", Restart: true, Notify: true}, hasOnIncident: true}
			ctrl, notifications := newTestController(t, db, camunda)
			incident := testIncident
			incident.BusinessKey = c.incidentKey
			if err := ctrl.createIncident(incident); err != nil {
				t.Fatal(err)
			}
			if !reflect.DeepEqual(camunda.calls, c.expectedCalls) {
				t.Errorf("calls %v, expected %v", camunda.calls, c.expectedCalls)
			}
			if !reflect.DeepEqual(camunda.startedWith, testStartVariables) {
				t.Errorf("started with %#v", camunda.startedWith)
			}
			if camunda.startedKey != c.expectedKey {
				t.Errorf("started with business key %q, expected %q", camunda.startedKey, c.expectedKey)
			}
			if errs := restartErrors(notifications()); len(errs) != 0 {
				t.Errorf("unexpected restart error notification %#v", errs)
			}
		})
	}
}

func TestRestartWithUnreadableStartVariables(t *testing.T) {
	camunda := &camundaMock{startVariablesErr: errors.New("engine unavailable")}
	db := &dbMock{onIncident: messages.OnIncident{ProcessDefinitionId: "definition", Restart: true, Notify: false}, hasOnIncident: true}
	ctrl, notifications := newTestController(t, db, camunda)
	incident := testIncident
	incident.BusinessKey = "bk"
	if err := ctrl.createIncident(incident); err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(camunda.calls, []string{"read", "stop"}) {
		t.Errorf("calls %v, expected read and stop without start", camunda.calls)
	}
	errs := restartErrors(notifications())
	if len(errs) != 1 {
		t.Fatalf("expected one restart error notification, got %#v", notifications())
	}
	if !strings.Contains(errs[0].Message, "unable to read start parameters") || !strings.Contains(errs[0].Message, "engine unavailable") {
		t.Errorf("notification does not name the reason: %v", errs[0].Message)
	}
	if errs[0].UserId != "user" {
		t.Errorf("notification for %v", errs[0].UserId)
	}
}

func TestNoRestartHandlerReadsNoStartVariables(t *testing.T) {
	for _, db := range []*dbMock{
		{hasOnIncident: false},
		{onIncident: messages.OnIncident{ProcessDefinitionId: "definition", Restart: false, Notify: true}, hasOnIncident: true},
	} {
		camunda := &camundaMock{startVariables: testStartVariables}
		ctrl, _ := newTestController(t, db, camunda)
		if err := ctrl.createIncident(testIncident); err != nil {
			t.Fatal(err)
		}
		if !reflect.DeepEqual(camunda.calls, []string{"stop"}) {
			t.Errorf("calls %v, expected only stop", camunda.calls)
		}
	}
}
