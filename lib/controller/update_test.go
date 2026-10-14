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
	"fmt"
	"net/http"
	"net/http/httptest"
	"reflect"
	"strings"
	"sync"
	"testing"
	"time"

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
	incidents     []messages.Incident
	lookupErr     error
}

func (this *dbMock) GetIncidents(id string, user string) (messages.IncidentMessage, bool, error) {
	return messages.IncidentMessage{}, false, nil
}

func (this *dbMock) FindIncidents(externalTaskId string, processDefinitionId string, processInstanceId string, limit int, offset int, sortBy string, asc bool, user string) ([]messages.IncidentMessage, error) {
	return nil, nil
}

// LastNotifiedIncident emulates the stored query: filter by definition, tenant and notified, newest time first.
func (this *dbMock) LastNotifiedIncident(processDefinitionId string, tenantId string) (messages.IncidentMessage, bool, error) {
	if this.lookupErr != nil {
		return messages.IncidentMessage{}, false, this.lookupErr
	}
	var last messages.IncidentMessage
	found := false
	for _, incident := range this.incidents {
		if incident.ProcessDefinitionId == processDefinitionId && incident.TenantId == tenantId && incident.Notified {
			if !found || incident.Time.After(last.Time) {
				last, found = incident, true
			}
		}
	}
	return last, found, nil
}

func (this *dbMock) DeleteByDefinitionId(id string) error { return nil }

func (this *dbMock) SaveIncident(incident messages.Incident) error {
	// replaces by id like the upsert in mongo
	for i, stored := range this.incidents {
		if stored.Id == incident.Id {
			this.incidents[i] = incident
			return nil
		}
	}
	this.incidents = append(this.incidents, incident)
	return nil
}

func (this *dbMock) DeleteIncidentByInstanceId(id string) error { return nil }

func (this *dbMock) SaveOnIncident(handler messages.OnIncident) error { return nil }

func (this *dbMock) GetOnIncident(definitionId string) (messages.OnIncident, bool, error) {
	return this.onIncident, this.hasOnIncident, nil
}

type metricMock struct{}

func (metricMock) NotifyIncidentMessage() {}

func newTestController(t *testing.T, db *dbMock, camunda *camundaMock) (*Controller, func() []notification.Message) {
	return newTestControllerWithConfig(t, configuration.Config{}, db, camunda)
}

func newTestControllerWithConfig(t *testing.T, config configuration.Config, db *dbMock, camunda *camundaMock) (*Controller, func() []notification.Message) {
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
	config.NotificationUrl = server.URL
	ctrl, err := New(context.Background(), config, db, camunda, metricMock{})
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

const throttleNote = "further incidents of this process within 1h are not notified"

var throttleTime = time.Date(2026, 10, 1, 12, 0, 0, 0, time.UTC)

func throttleIncident(id string, definition string, at time.Time) messages.Incident {
	incident := testIncident
	incident.Id = id
	incident.ProcessInstanceId = "instance-" + id
	incident.ProcessDefinitionId = definition
	incident.Time = at
	return incident
}

func notified(incident messages.Incident) messages.Incident {
	incident.Notified = true
	return incident
}

func newThrottleController(t *testing.T, throttle string, db *dbMock) (*Controller, *camundaMock, func() []notification.Message) {
	camunda := &camundaMock{startVariables: testStartVariables}
	ctrl, notifications := newTestControllerWithConfig(t, configuration.Config{NotificationThrottle: throttle}, db, camunda)
	return ctrl, camunda, notifications
}

func restartingDb() *dbMock {
	return &dbMock{onIncident: messages.OnIncident{ProcessDefinitionId: "definition", Restart: true, Notify: true}, hasOnIncident: true}
}

func TestNotificationThrottle(t *testing.T) {
	restartCalls := []string{"read", "stop", "start"}
	cases := []struct {
		name           string
		throttle       string
		stored         []messages.Incident
		lookupErr      error
		incident       messages.Incident
		expectNotified bool
		expectNote     bool
		expectedCalls  []string
	}{
		{name: "first incident notifies with the throttle note", throttle: "1h",
			incident: throttleIncident("a", "definition", throttleTime), expectNotified: true, expectNote: true, expectedCalls: restartCalls},
		{name: "second incident within the window is silent but restarts", throttle: "1h",
			stored:   []messages.Incident{notified(throttleIncident("a", "definition", throttleTime))},
			incident: throttleIncident("b", "definition", throttleTime.Add(10*time.Minute)), expectNotified: false, expectedCalls: restartCalls},
		{name: "newest notified incident decides", throttle: "1h",
			stored: []messages.Incident{
				notified(throttleIncident("old", "definition", throttleTime.Add(-5*time.Hour))),
				notified(throttleIncident("new", "definition", throttleTime.Add(-5*time.Minute))),
			},
			incident: throttleIncident("b", "definition", throttleTime), expectNotified: false, expectedCalls: restartCalls},
		{name: "a silent incident does not extend the window", throttle: "1h",
			stored: []messages.Incident{
				notified(throttleIncident("sent", "definition", throttleTime.Add(-2*time.Hour))),
				throttleIncident("silent", "definition", throttleTime.Add(-5*time.Minute)),
			},
			incident: throttleIncident("b", "definition", throttleTime), expectNotified: true, expectNote: true, expectedCalls: restartCalls},
		{name: "stored incident without the notified mark does not throttle", throttle: "1h",
			stored:   []messages.Incident{throttleIncident("a", "definition", throttleTime)},
			incident: throttleIncident("b", "definition", throttleTime.Add(time.Minute)), expectNotified: true, expectNote: true, expectedCalls: restartCalls},
		{name: "incident after the window notifies", throttle: "1h",
			stored:   []messages.Incident{notified(throttleIncident("a", "definition", throttleTime))},
			incident: throttleIncident("b", "definition", throttleTime.Add(61*time.Minute)), expectNotified: true, expectNote: true, expectedCalls: restartCalls},
		{name: "incident exactly one window later notifies", throttle: "1h",
			stored:   []messages.Incident{notified(throttleIncident("a", "definition", throttleTime))},
			incident: throttleIncident("b", "definition", throttleTime.Add(time.Hour)), expectNotified: true, expectNote: true, expectedCalls: restartCalls},
		{name: "incident one nanosecond inside the window is silent", throttle: "1h",
			stored:   []messages.Incident{notified(throttleIncident("a", "definition", throttleTime))},
			incident: throttleIncident("b", "definition", throttleTime.Add(time.Hour-time.Nanosecond)), expectNotified: false, expectedCalls: restartCalls},
		{name: "other definition notifies", throttle: "1h",
			stored:   []messages.Incident{notified(throttleIncident("a", "other-definition", throttleTime))},
			incident: throttleIncident("b", "definition", throttleTime.Add(time.Minute)), expectNotified: true, expectNote: true, expectedCalls: restartCalls},
		{name: "other tenant notifies", throttle: "1h",
			stored: []messages.Incident{func() messages.Incident {
				i := notified(throttleIncident("a", "definition", throttleTime))
				i.TenantId = "other-user"
				return i
			}()},
			incident: throttleIncident("b", "definition", throttleTime.Add(time.Minute)), expectNotified: true, expectNote: true, expectedCalls: restartCalls},
		{name: "throttle off notifies every time", throttle: "0",
			stored:   []messages.Incident{notified(throttleIncident("a", "definition", throttleTime))},
			incident: throttleIncident("b", "definition", throttleTime.Add(time.Minute)), expectNotified: true, expectedCalls: restartCalls},
		{name: "throttle 0s notifies every time", throttle: "0s",
			stored:   []messages.Incident{notified(throttleIncident("a", "definition", throttleTime))},
			incident: throttleIncident("b", "definition", throttleTime.Add(time.Minute)), expectNotified: true, expectedCalls: restartCalls},
		{name: "unset throttle notifies every time", throttle: "",
			stored:   []messages.Incident{notified(throttleIncident("a", "definition", throttleTime))},
			incident: throttleIncident("b", "definition", throttleTime.Add(time.Minute)), expectNotified: true, expectedCalls: restartCalls},
		{name: "lookup error notifies", throttle: "1h", lookupErr: errors.New("database unavailable"),
			stored:   []messages.Incident{notified(throttleIncident("a", "definition", throttleTime))},
			incident: throttleIncident("b", "definition", throttleTime.Add(time.Minute)), expectNotified: true, expectNote: true, expectedCalls: restartCalls},
		{name: "stored incident without time does not throttle", throttle: "1h",
			stored:   []messages.Incident{notified(throttleIncident("a", "definition", time.Time{}))},
			incident: throttleIncident("b", "definition", throttleTime), expectNotified: true, expectNote: true, expectedCalls: restartCalls},
		{name: "stored incident later than this one does not throttle", throttle: "1h",
			stored:   []messages.Incident{notified(throttleIncident("a", "definition", throttleTime.Add(time.Minute)))},
			incident: throttleIncident("b", "definition", throttleTime), expectNotified: true, expectNote: true, expectedCalls: restartCalls},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			db := restartingDb()
			db.incidents = c.stored
			db.lookupErr = c.lookupErr
			ctrl, camunda, notifications := newThrottleController(t, c.throttle, db)
			if err := ctrl.createIncident(c.incident); err != nil {
				t.Fatal(err)
			}
			sent := notifications()
			if c.expectNotified != (len(sent) == 1) || len(sent) > 1 {
				t.Fatalf("expected notified=%v, got %#v", c.expectNotified, sent)
			}
			if c.expectNotified {
				if !strings.Contains(sent[0].Message, "process will be restarted") {
					t.Errorf("restart line missing: %q", sent[0].Message)
				}
				if strings.Contains(sent[0].Message, throttleNote) != c.expectNote {
					t.Errorf("throttle note expected=%v in %q", c.expectNote, sent[0].Message)
				}
			}
			if !reflect.DeepEqual(camunda.calls, c.expectedCalls) {
				t.Errorf("calls %v, expected %v", camunda.calls, c.expectedCalls)
			}
			saved := db.incidents[len(db.incidents)-1]
			if saved.Id != c.incident.Id {
				t.Errorf("incident %v not saved, last saved %v", c.incident.Id, saved.Id)
			}
			if saved.Notified != c.expectNotified {
				t.Errorf("saved incident has notified=%v, expected %v", saved.Notified, c.expectNotified)
			}
		})
	}
}

func TestNotificationThrottleWithoutHandler(t *testing.T) {
	db := &dbMock{hasOnIncident: false, incidents: []messages.Incident{notified(throttleIncident("a", "definition", throttleTime))}}
	ctrl, camunda, notifications := newThrottleController(t, "1h", db)
	if err := ctrl.createIncident(throttleIncident("b", "definition", throttleTime.Add(time.Minute))); err != nil {
		t.Fatal(err)
	}
	if sent := notifications(); len(sent) != 0 {
		t.Errorf("unexpected notification %#v", sent)
	}
	if !reflect.DeepEqual(camunda.calls, []string{"stop"}) {
		t.Errorf("calls %v, expected only stop", camunda.calls)
	}
}

func TestNotificationThrottleIgnoresHandlerWithoutNotify(t *testing.T) {
	db := &dbMock{onIncident: messages.OnIncident{ProcessDefinitionId: "definition", Restart: false, Notify: false}, hasOnIncident: true}
	db.lookupErr = errors.New("must not be queried")
	ctrl, _, notifications := newThrottleController(t, "1h", db)
	if err := ctrl.createIncident(throttleIncident("a", "definition", throttleTime)); err != nil {
		t.Fatal(err)
	}
	if sent := notifications(); len(sent) != 0 {
		t.Errorf("unexpected notification %#v", sent)
	}
}

func TestNotificationThrottleContinuousFault(t *testing.T) {
	db := restartingDb()
	ctrl, camunda, notifications := newThrottleController(t, "6h", db)
	count := 0
	for offset := time.Duration(0); offset <= 13*time.Hour; offset += 5 * time.Minute {
		count++
		if err := ctrl.createIncident(throttleIncident(fmt.Sprint("incident-", count), "definition", throttleTime.Add(offset))); err != nil {
			t.Fatal(err)
		}
	}
	expected := []time.Duration{0, 6 * time.Hour, 12 * time.Hour}
	actual := []time.Duration{}
	for _, incident := range db.incidents {
		if incident.Notified {
			actual = append(actual, incident.Time.Sub(throttleTime))
		}
	}
	if !reflect.DeepEqual(actual, expected) {
		t.Errorf("notified incidents at %v, expected %v", actual, expected)
	}
	if sent := notifications(); len(sent) != len(expected) {
		t.Errorf("expected %v notifications, got %v", len(expected), len(sent))
	}
	if len(camunda.calls) != 3*count {
		t.Errorf("expected every one of %v incidents to be stopped and restarted, got %v calls", count, len(camunda.calls))
	}
}

func TestIncidentCannotBeCreatedAsNotified(t *testing.T) {
	db := restartingDb()
	ctrl, _, notifications := newThrottleController(t, "1h", db)
	db.incidents = []messages.Incident{notified(throttleIncident("a", "definition", throttleTime))}
	incident := notified(throttleIncident("b", "other-definition", throttleTime))
	db.hasOnIncident = true
	db.onIncident = messages.OnIncident{ProcessDefinitionId: "other-definition", Notify: false}
	if err := ctrl.createIncident(incident); err != nil {
		t.Fatal(err)
	}
	if sent := notifications(); len(sent) != 0 {
		t.Errorf("unexpected notification %#v", sent)
	}
	if saved := db.incidents[len(db.incidents)-1]; saved.Notified {
		t.Errorf("incident saved as notified without a notification")
	}
}

func TestAnIncidentHandledAgainKeepsItsNotification(t *testing.T) {
	db := restartingDb()
	ctrl, _, notifications := newThrottleController(t, "1h", db)
	incident := throttleIncident("a", "definition", throttleTime)
	for i := 0; i < 2; i++ {
		if err := ctrl.createIncident(incident); err != nil {
			t.Fatal(err)
		}
	}
	if sent := notifications(); len(sent) != 1 {
		t.Errorf("expected one notification, got %#v", sent)
	}
	if len(db.incidents) != 1 || !db.incidents[0].Notified {
		t.Errorf("the incident lost its notified mark: %#v", db.incidents)
	}
	if err := ctrl.createIncident(throttleIncident("b", "definition", throttleTime.Add(30*time.Minute))); err != nil {
		t.Fatal(err)
	}
	if sent := notifications(); len(sent) != 1 {
		t.Errorf("the window restarted after the incident was handled again: %#v", sent)
	}
}

func TestNotificationThrottleSequence(t *testing.T) {
	ctrl, camunda, notifications := newThrottleController(t, "1h", restartingDb())
	for i, offset := range []time.Duration{0, 5 * time.Minute, 10 * time.Minute} {
		if err := ctrl.createIncident(throttleIncident(string(rune('a'+i)), "definition", throttleTime.Add(offset))); err != nil {
			t.Fatal(err)
		}
	}
	if sent := notifications(); len(sent) != 1 {
		t.Errorf("expected one notification for three incidents, got %#v", sent)
	}
	if len(camunda.calls) != 9 {
		t.Errorf("expected three restarts of read, stop, start, got %v", camunda.calls)
	}
}

func TestRestartErrorNotificationIsNotThrottled(t *testing.T) {
	db := restartingDb()
	db.incidents = []messages.Incident{notified(throttleIncident("a", "definition", throttleTime))}
	camunda := &camundaMock{startVariablesErr: errors.New("engine unavailable")}
	ctrl, notifications := newTestControllerWithConfig(t, configuration.Config{NotificationThrottle: "1h"}, db, camunda)
	if err := ctrl.createIncident(throttleIncident("b", "definition", throttleTime.Add(time.Minute))); err != nil {
		t.Fatal(err)
	}
	sent := notifications()
	if len(sent) != 1 || len(restartErrors(sent)) != 1 {
		t.Errorf("expected only the restart error notification, got %#v", sent)
	}
}

func TestIncidentWithoutTimeGetsHandlingTime(t *testing.T) {
	db := restartingDb()
	ctrl, _, notifications := newThrottleController(t, "1h", db)
	before := time.Now()
	if err := ctrl.createIncident(throttleIncident("a", "definition", time.Time{})); err != nil {
		t.Fatal(err)
	}
	saved := db.incidents[0]
	if saved.Time.Before(before) || saved.Time.After(time.Now()) {
		t.Errorf("saved time %v not set to the time of handling", saved.Time)
	}
	// the stored time is what throttles the next incident without time
	if err := ctrl.createIncident(throttleIncident("b", "definition", time.Time{})); err != nil {
		t.Fatal(err)
	}
	if sent := notifications(); len(sent) != 1 {
		t.Errorf("expected one notification for two incidents without time, got %#v", sent)
	}
}

func TestNewRejectsInvalidNotificationThrottle(t *testing.T) {
	for _, throttle := range []string{"soon", "-1h", "6"} {
		_, err := New(context.Background(), configuration.Config{NotificationThrottle: throttle}, &dbMock{}, &camundaMock{}, metricMock{})
		if err == nil {
			t.Errorf("expected an error for notification_throttle %q", throttle)
		}
	}
}
