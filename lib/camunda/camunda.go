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

package camunda

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"runtime/debug"
	"slices"
	"time"

	"github.com/SENERGY-Platform/process-incident-api/lib/camunda/cache"
	"github.com/SENERGY-Platform/process-incident-api/lib/camunda/shards"
	"github.com/SENERGY-Platform/process-incident-api/lib/configuration"
	"github.com/SENERGY-Platform/process-incident-api/lib/interfaces"
	"github.com/SENERGY-Platform/process-incident-api/lib/messages"
)

type FactoryType struct{}

var Factory = &FactoryType{}

type Camunda struct {
	config configuration.Config
	shards *shards.Shards
}

func (this *FactoryType) Get(ctx context.Context, config configuration.Config) (interfaces.Camunda, error) {
	s, err := shards.New(config.ShardsDb, cache.New(&cache.CacheConfig{L1Expiration: 60}))
	if err != nil {
		return nil, err
	}
	return &Camunda{config: config, shards: s}, nil
}

func (this *Camunda) StopProcessInstance(id string, tenantId string) (err error) {
	shard, err := this.shards.EnsureShardForUser(tenantId)
	if err != nil {
		return err
	}
	client := &http.Client{Timeout: 5 * time.Second}
	request, err := http.NewRequest("DELETE", shard+"/engine-rest/process-instance/"+url.PathEscape(id)+"?skipIoMappings=true", nil)
	if err != nil {
		return err
	}
	resp, err := client.Do(request)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode == http.StatusNotFound {
		return nil
	}
	if resp.StatusCode == 200 || resp.StatusCode == 204 {
		return nil
	}
	msg, _ := io.ReadAll(resp.Body)
	err = errors.New("error on delete in engine for " + shard + "/engine-rest/process-instance/" + url.PathEscape(id) + ": " + resp.Status + " " + string(msg))
	return err
}

type NameWrapper struct {
	Name string `json:"name"`
}

func (this *Camunda) GetProcessName(id string, tenantId string) (name string, err error) {
	shard, err := this.shards.EnsureShardForUser(tenantId)
	if err != nil {
		return "", err
	}
	client := &http.Client{Timeout: 5 * time.Second}
	request, err := http.NewRequest("GET", shard+"/engine-rest/process-definition/"+url.PathEscape(id), nil)
	if err != nil {
		return "", err
	}
	resp, err := client.Do(request)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		temp, _ := io.ReadAll(resp.Body)
		this.config.GetLogger().Error("unexpected response", "status", resp.Status, "body", string(temp))
		debug.PrintStack()
		return "", errors.New("unexpected response")
	}
	result := NameWrapper{}
	err = json.NewDecoder(resp.Body).Decode(&result)
	return result.Name, err
}

func (this *Camunda) StartProcess(processDefinitionId string, userId string, variables map[string]messages.Variable) (err error) {
	return this.StartProcessWithBusinessKey(processDefinitionId, "", userId, variables)
}

// StartProcessWithBusinessKey starts the definition through its start form; an empty business key is omitted.
func (this *Camunda) StartProcessWithBusinessKey(processDefinitionId string, businessKey string, userId string, variables map[string]messages.Variable) (err error) {
	shard, err := this.shards.EnsureShardForUser(userId)
	if err != nil {
		return err
	}
	return this.startProcess(shard, processDefinitionId, businessKey, variables)
}

func (this *Camunda) startProcess(shard string, processDefinitionId string, businessKey string, variables map[string]messages.Variable) (err error) {
	message := createStartMessage(variables, businessKey)
	b := new(bytes.Buffer)
	err = json.NewEncoder(b).Encode(message)
	if err != nil {
		return err
	}
	this.config.GetLogger().Debug("start process definition at camunda", "processDefinitionId", processDefinitionId, "businessKey", businessKey, "variables", variableNames(variables))
	req, err := http.NewRequest("POST", shard+"/engine-rest/process-definition/"+url.PathEscape(processDefinitionId)+"/submit-form", b)
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	client := http.Client{Timeout: 5 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		temp, _ := io.ReadAll(resp.Body)
		return errors.New(resp.Status + " " + string(temp))
	}
	return nil
}

type Variable = messages.Variable

type StartMessage struct {
	BusinessKey string                       `json:"businessKey,omitempty"`
	Variables   map[string]messages.Variable `json:"variables,omitempty"`
}

func createStartMessage(variables map[string]messages.Variable, businessKey string) StartMessage {
	return StartMessage{BusinessKey: businessKey, Variables: variables}
}

func variableNames(variables map[string]messages.Variable) []string {
	result := make([]string, 0, len(variables))
	for name := range variables {
		result = append(result, name)
	}
	slices.Sort(result)
	return result
}

// GetStartVariables returns the variables a restart of the process instance has to submit to its start form,
// or nil if the definition has no start form variables.
func (this *Camunda) GetStartVariables(processDefinitionId string, processInstanceId string, tenantId string) (result map[string]messages.Variable, err error) {
	shard, err := this.shards.EnsureShardForUser(tenantId)
	if err != nil {
		return nil, err
	}
	return this.getStartVariables(shard, processDefinitionId, processInstanceId)
}

func (this *Camunda) getStartVariables(shard string, processDefinitionId string, processInstanceId string) (result map[string]messages.Variable, err error) {
	formVariables, err := this.getProcessParameters(shard, processDefinitionId)
	if err != nil {
		return nil, fmt.Errorf("unable to read form variables of process definition %v: %w", processDefinitionId, err)
	}
	if len(formVariables) == 0 {
		return nil, nil
	}
	instanceVariables, runtimeErr := this.getInstanceVariables(shard, processInstanceId)
	if runtimeErr != nil {
		// the engine answers 500, not 404, for an instance that no longer runs
		var historyErr error
		instanceVariables, historyErr = this.getHistoricInstanceVariables(shard, processInstanceId)
		if historyErr != nil {
			return nil, fmt.Errorf("unable to read variables of process instance %v: %w", processInstanceId, errors.Join(runtimeErr, historyErr))
		}
	}
	result = map[string]messages.Variable{}
	for name := range formVariables {
		if variable, ok := instanceVariables[name]; ok {
			result[name] = variable
		}
	}
	return result, nil
}

func (this *Camunda) getProcessParameters(shard string, processDefinitionId string) (result map[string]messages.Variable, err error) {
	err = getJson(shard+"/engine-rest/process-definition/"+url.PathEscape(processDefinitionId)+"/form-variables", &result)
	return result, err
}

func (this *Camunda) getInstanceVariables(shard string, processInstanceId string) (result map[string]messages.Variable, err error) {
	err = getJson(shard+"/engine-rest/process-instance/"+url.PathEscape(processInstanceId)+"/variables?deserializeValues=false", &result)
	return result, err
}

type HistoricVariableInstance struct {
	Name              string          `json:"name"`
	Type              string          `json:"type"`
	Value             json.RawMessage `json:"value"`
	ValueInfo         json.RawMessage `json:"valueInfo"`
	ProcessInstanceId string          `json:"processInstanceId"`
	ExecutionId       string          `json:"executionId"`
	TaskId            *string         `json:"taskId"`
}

// getHistoricInstanceVariables returns the variables of the process instance scope; local variables of
// sub-scopes and tasks are excluded because they may shadow start variables by name.
func (this *Camunda) getHistoricInstanceVariables(shard string, processInstanceId string) (result map[string]messages.Variable, err error) {
	list := []HistoricVariableInstance{}
	query := url.Values{"processInstanceId": {processInstanceId}, "deserializeValues": {"false"}}
	err = getJson(shard+"/engine-rest/history/variable-instance?"+query.Encode(), &list)
	if err != nil {
		return nil, err
	}
	// with a history level below audit the engine records no variables; an empty list is no evidence of an empty start
	if len(list) == 0 {
		return nil, errors.New("no historic variables recorded for process instance " + processInstanceId)
	}
	result = map[string]messages.Variable{}
	for _, variable := range list {
		if variable.ExecutionId != processInstanceId || (variable.TaskId != nil && *variable.TaskId != "") {
			continue
		}
		result[variable.Name] = messages.Variable{Value: variable.Value, Type: variable.Type, ValueInfo: variable.ValueInfo}
	}
	return result, nil
}

func getJson(endpoint string, result interface{}) error {
	client := http.Client{Timeout: 5 * time.Second}
	resp, err := client.Get(endpoint)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		temp, _ := io.ReadAll(resp.Body)
		return errors.New(resp.Status + " " + string(temp))
	}
	return json.NewDecoder(resp.Body).Decode(result)
}

func (this *Camunda) GetIncidents() (result []messages.CamundaIncident, err error) {
	shards, err := this.shards.GetShards()
	if err != nil {
		return result, err
	}
	for _, shard := range shards {
		temp, err := this.GetShardIncidents(shard)
		if err != nil {
			return result, err
		}
		result = append(result, temp...)
	}
	return result, nil
}

func (this *Camunda) GetShardIncidents(shard string) (result []messages.CamundaIncident, err error) {
	client := http.Client{Timeout: 5 * time.Second}
	resp, err := client.Get(shard + "/engine-rest/incident")
	if err != nil {
		return result, err
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 300 {
		pl, _ := io.ReadAll(resp.Body)
		err = fmt.Errorf("unable to load incidents: %v", string(pl))
		return result, err
	}
	err = json.NewDecoder(resp.Body).Decode(&result)
	if err != nil {
		return result, err
	}
	return result, nil
}

func (this *Camunda) GetHistoricProcessInstance(id string, userId string) (result messages.HistoricProcessInstance, err error) {
	shard, err := this.shards.EnsureShardForUser(userId)
	if err != nil {
		return result, err
	}

	client := http.Client{Timeout: 5 * time.Second}
	resp, err := client.Get(shard + "/engine-rest/history/process-instance/" + url.QueryEscape(id))
	if err != nil {
		return result, err
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 300 {
		pl, _ := io.ReadAll(resp.Body)
		err = fmt.Errorf("unable to load process-instance: %v", string(pl))
		return result, err
	}
	err = json.NewDecoder(resp.Body).Decode(&result)
	if err != nil {
		return result, err
	}
	return result, nil
}
