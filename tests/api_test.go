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
	"encoding/json"
	"github.com/SENERGY-Platform/process-incident-api/lib/configuration"
	"github.com/SENERGY-Platform/process-incident-api/lib/messages"
	"io/ioutil"
	"net/http"
	"net/url"
	"reflect"
	"sort"
	"testing"
	"time"
)

func checkIncidentsByPdid(t *testing.T, config configuration.Config, processDefinitionId string, expected []messages.IncidentMessage) {
	checkApiListFilter(t, config, "process_definition_id="+url.QueryEscape(processDefinitionId), expected)
}

func checkIncidentsByPiid(t *testing.T, config configuration.Config, processInstanceId string, expected []messages.IncidentMessage) {
	checkApiListFilter(t, config, "process_instance_id="+url.QueryEscape(processInstanceId), expected)
}

func checkIncidentsByTaskId(t *testing.T, config configuration.Config, taskId string, expected []messages.IncidentMessage) {
	checkApiListFilter(t, config, "external_task_id="+url.QueryEscape(taskId), expected)
}

func checkIncidentById(t *testing.T, config configuration.Config, id string, expected messages.IncidentMessage) {
	client := &http.Client{Timeout: 5 * time.Second}
	request, err := http.NewRequest("GET", "http://localhost:"+config.ApiPort+"/incidents/"+url.PathEscape(id), nil)
	if err != nil {
		t.Fatal(err)
		return
	}
	resp, err := client.Do(request)
	if err != nil {
		t.Fatal(err)
		return
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		t.Fatal(resp.StatusCode)
		return
	}
	result := messages.IncidentMessage{}
	err = json.NewDecoder(resp.Body).Decode(&result)
	if err != nil {
		t.Fatal(err)
		return
	}
	if expected.Time.Unix() != result.Time.Unix() {
		t.Fatal(expected.Time.Unix(), result.Time.Unix())
	}
	result.Time = time.Time{}
	expected.Time = time.Time{}
	if !reflect.DeepEqual(result, expected) {
		t.Fatal(result, expected)
		return
	}
}

func checkApiLimitAndSort(t *testing.T, config configuration.Config, limit string, offset string, sort string, expected []messages.IncidentMessage) {
	client := &http.Client{Timeout: 5 * time.Second}
	request, err := http.NewRequest("GET", "http://localhost:"+config.ApiPort+"/incidents?limit="+url.QueryEscape(limit)+"&offset="+url.QueryEscape(offset)+"&sort="+url.QueryEscape(sort), nil)
	if err != nil {
		t.Fatal(err)
		return
	}
	resp, err := client.Do(request)
	if err != nil {
		t.Fatal(err)
		return
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		b, _ := ioutil.ReadAll(resp.Body)
		t.Fatal(resp.StatusCode, string(b))
		return
	}
	result := []messages.IncidentMessage{}
	err = json.NewDecoder(resp.Body).Decode(&result)
	if err != nil {
		t.Fatal(err)
		return
	}
	if len(expected) != len(result) {
		t.Fatal(len(expected), len(result), result, expected)
		return
	}
	for i := 0; i < len(result); i++ {
		if expected[i].Time.Unix() != result[i].Time.Unix() {
			t.Fatal(expected[i].Time.Unix(), result[i].Time.Unix())
		}
		result[i].Time = time.Time{}
		expected[i].Time = time.Time{}
	}
	if !reflect.DeepEqual(result, expected) {
		t.Fatal(result, expected)
		return
	}
}

func checkApiListFilter(t *testing.T, config configuration.Config, query string, expected []messages.IncidentMessage) {
	client := &http.Client{Timeout: 5 * time.Second}
	request, err := http.NewRequest("GET", "http://localhost:"+config.ApiPort+"/incidents?"+query, nil)
	if err != nil {
		t.Fatal(err)
		return
	}
	resp, err := client.Do(request)
	if err != nil {
		t.Fatal(err)
		return
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		t.Fatal(resp.StatusCode)
		return
	}
	result := []messages.IncidentMessage{}
	err = json.NewDecoder(resp.Body).Decode(&result)
	if err != nil {
		t.Fatal(err)
		return
	}
	sort.Slice(result, func(i, j int) bool {
		return result[i].Id < result[j].Id
	})
	sort.Slice(expected, func(i, j int) bool {
		return expected[i].Id < expected[j].Id
	})
	if len(expected) != len(result) {
		t.Fatal(len(expected), len(result), result, expected)
		return
	}
	for i := 0; i < len(result); i++ {
		if expected[i].Time.Unix() != result[i].Time.Unix() {
			t.Fatal(expected[i].Time.Unix(), result[i].Time.Unix())
		}
		result[i].Time = time.Time{}
		expected[i].Time = time.Time{}
	}
	if !reflect.DeepEqual(result, expected) {
		t.Fatal(result, expected)
		return
	}
}
