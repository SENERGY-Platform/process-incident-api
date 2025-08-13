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
	"log"
	"net/http"
	"net/url"
	"runtime/debug"
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
		log.Println("ERROR:", resp.Status, string(temp))
		debug.PrintStack()
		return "", errors.New("unexpected response")
	}
	result := NameWrapper{}
	err = json.NewDecoder(resp.Body).Decode(&result)
	return result.Name, err
}

func (this *Camunda) StartProcess(processDefinitionId string, userId string) (err error) {
	shard, err := this.shards.EnsureShardForUser(userId)
	if err != nil {
		return err
	}

	parameters, err := this.getProcessParameters(shard, processDefinitionId)
	if err != nil {
		return err
	}
	if len(parameters) > 0 {
		return errors.New("restart of processes with start-parameters not supported")
	}

	//message := createStartMessage(nil)

	b := new(bytes.Buffer)
	err = json.NewEncoder(b).Encode(map[string]interface{}{})
	if err != nil {
		return
	}
	req, err := http.NewRequest("POST", shard+"/engine-rest/process-definition/"+url.QueryEscape(processDefinitionId)+"/submit-form", b)
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return err
	}

	defer resp.Body.Close()
	temp, _ := io.ReadAll(resp.Body)

	if resp.StatusCode != http.StatusOK {
		err = errors.New(resp.Status + " " + string(temp))
		return err
	}
	return nil
}

func (this *Camunda) StartProcessWithBusinessKey(processDefinitionId string, businessKey string, userId string) (err error) {
	shard, err := this.shards.EnsureShardForUser(userId)
	if err != nil {
		return err
	}

	parameters, err := this.getProcessParameters(shard, processDefinitionId)
	if err != nil {
		return err
	}
	if len(parameters) > 0 {
		return errors.New("restart of processes with start-parameters not supported")
	}

	message := createStartMessage(nil, businessKey)

	b := new(bytes.Buffer)
	err = json.NewEncoder(b).Encode(message)
	if err != nil {
		return
	}
	if this.config.Debug == true {
		log.Println("DEBUG: start process definition at camunda:", processDefinitionId)
	}
	req, err := http.NewRequest("POST", shard+"/engine-rest/process-definition/"+url.QueryEscape(processDefinitionId)+"/submit-form", b)
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	client := http.Client{Timeout: 5 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	if resp.StatusCode != http.StatusOK {
		temp, _ := io.ReadAll(resp.Body)
		err = errors.New(resp.Status + " " + string(temp))
		return
	}
	return nil
}

type Variable struct {
	Value     interface{} `json:"value"`
	Type      string      `json:"type"`
	ValueInfo interface{} `json:"valueInfo"`
}

func (this *Camunda) getProcessParameters(shard string, processDefinitionId string) (result map[string]Variable, err error) {
	req, err := http.NewRequest("GET", shard+"/engine-rest/process-definition/"+url.QueryEscape(processDefinitionId)+"/form-variables", nil)
	if err != nil {
		return result, err
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return result, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		temp, _ := io.ReadAll(resp.Body)
		err = errors.New(resp.Status + " " + string(temp))
		return
	}
	err = json.NewDecoder(resp.Body).Decode(&result)
	return
}

func createStartMessage(parameter map[string]interface{}, businessKey string) map[string]interface{} {
	if len(parameter) == 0 {
		return map[string]interface{}{"businessKey": businessKey}
	}
	variables := map[string]interface{}{}
	for key, val := range parameter {
		variables[key] = map[string]interface{}{
			"value": val,
		}
	}
	return map[string]interface{}{"variables": variables, "businessKey": businessKey}
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
