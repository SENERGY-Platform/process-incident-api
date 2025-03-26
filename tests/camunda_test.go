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
	"bytes"
	"encoding/json"
	"errors"
	"github.com/SENERGY-Platform/process-incident-api/lib/camunda/cache"
	"github.com/SENERGY-Platform/process-incident-api/lib/camunda/shards"
	"github.com/SENERGY-Platform/process-incident-api/lib/configuration"
	"io"
	"log"
	"net/http"
	"net/url"
	"strings"
	"testing"
	"time"
)

const xml = `<?xml version="1.0" encoding="UTF-8"?>
<bpmn:definitions xmlns:xsi="http://www.w3.org/2001/XMLSchema-instance" xmlns:bpmn="http://www.omg.org/spec/BPMN/20100524/MODEL" xmlns:bpmndi="http://www.omg.org/spec/BPMN/20100524/DI" xmlns:dc="http://www.omg.org/spec/DD/20100524/DC" xmlns:camunda="http://camunda.org/schema/1.0/bpmn" xmlns:di="http://www.omg.org/spec/DD/20100524/DI" id="Definitions_1" targetNamespace="http://bpmn.io/schema/bpmn"><bpmn:process id="test" name="__name__" isExecutable="true"><bpmn:startEvent id="StartEvent_1"><bpmn:outgoing>SequenceFlow_0s7kcos</bpmn:outgoing></bpmn:startEvent><bpmn:sequenceFlow id="SequenceFlow_0s7kcos" sourceRef="StartEvent_1" targetRef="Task_0yf9l1o" /><bpmn:endEvent id="EndEvent_1bjwv72"><bpmn:incoming>SequenceFlow_06gsxk1</bpmn:incoming></bpmn:endEvent><bpmn:sequenceFlow id="SequenceFlow_06gsxk1" sourceRef="Task_0yf9l1o" targetRef="EndEvent_1bjwv72" /><bpmn:serviceTask id="Task_0yf9l1o" camunda:type="external" camunda:topic="test"><bpmn:incoming>SequenceFlow_0s7kcos</bpmn:incoming><bpmn:outgoing>SequenceFlow_06gsxk1</bpmn:outgoing></bpmn:serviceTask></bpmn:process><bpmndi:BPMNDiagram id="BPMNDiagram_1"><bpmndi:BPMNPlane id="BPMNPlane_1" bpmnElement="test"><bpmndi:BPMNShape id="_BPMNShape_StartEvent_2" bpmnElement="StartEvent_1"><dc:Bounds x="173" y="102" width="36" height="36" /></bpmndi:BPMNShape><bpmndi:BPMNEdge id="SequenceFlow_0s7kcos_di" bpmnElement="SequenceFlow_0s7kcos"><di:waypoint x="209" y="120" /><di:waypoint x="260" y="120" /></bpmndi:BPMNEdge><bpmndi:BPMNShape id="EndEvent_1bjwv72_di" bpmnElement="EndEvent_1bjwv72"><dc:Bounds x="412" y="102" width="36" height="36" /></bpmndi:BPMNShape><bpmndi:BPMNEdge id="SequenceFlow_06gsxk1_di" bpmnElement="SequenceFlow_06gsxk1"><di:waypoint x="360" y="120" /><di:waypoint x="412" y="120" /></bpmndi:BPMNEdge><bpmndi:BPMNShape id="ServiceTask_0s9hyr3_di" bpmnElement="Task_0yf9l1o"><dc:Bounds x="260" y="80" width="100" height="80" /></bpmndi:BPMNShape></bpmndi:BPMNPlane></bpmndi:BPMNDiagram></bpmn:definitions>`

func deployProcess(t *testing.T, config configuration.Config) (id string) {
	id, err := deployProcessRequest(config, "test")
	if err != nil {
		t.Fatal(err)
		return
	}
	return id
}

func checkProcess(t *testing.T, config configuration.Config, instanceId string, expectExistence bool) {
	s, err := shards.New(config.ShardsDb, cache.None)
	if err != nil {
		t.Fatal(err)
		return
	}

	shard, err := s.EnsureShardForUser("")
	if err != nil {
		t.Fatal(err)
		return
	}

	client := &http.Client{Timeout: 5 * time.Second}
	request, err := http.NewRequest("GET", shard+"/engine-rest/process-instance/"+url.QueryEscape(instanceId), nil)
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
	if resp.StatusCode == http.StatusNotFound && expectExistence {
		t.Fatal(resp.Status, expectExistence)
	}
	if resp.StatusCode != http.StatusNotFound && !expectExistence {
		t.Fatal(resp.Status, expectExistence)
	}
}

func startProcess(t *testing.T, config configuration.Config, processDefinitionId string) string {
	s, err := shards.New(config.ShardsDb, cache.None)
	if err != nil {
		t.Fatal(err)
		return ""
	}

	shard, err := s.EnsureShardForUser("")
	if err != nil {
		t.Fatal(err)
		return ""
	}
	client := &http.Client{Timeout: 5 * time.Second}
	request, err := http.NewRequest("POST", shard+"/engine-rest/process-definition/"+url.QueryEscape(processDefinitionId)+"/start", bytes.NewBuffer([]byte("{}")))
	if err != nil {
		t.Fatal(err)
		return ""
	}
	request.Header.Set("Content-Type", "application/json")
	resp, err := client.Do(request)
	if err != nil {
		t.Fatal(err)
		return ""
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		b, _ := io.ReadAll(resp.Body)
		t.Fatal(string(b))
		return ""
	}
	result := map[string]interface{}{}
	err = json.NewDecoder(resp.Body).Decode(&result)
	if err != nil {
		t.Fatal(err)
		return ""
	}
	return result["id"].(string)
}

func deployProcessRequest(config configuration.Config, name string) (id string, err error) {
	return deployProcessWithInfo(config, name, xml, "<svg/>", "")
}

func deployProcessWithInfo(config configuration.Config, name string, xml string, svg string, owner string) (id string, err error) {
	s, err := shards.New(config.ShardsDb, cache.None)
	if err != nil {
		return id, err
	}

	shard, err := s.EnsureShardForUser("")
	if err != nil {
		return id, err
	}
	result := map[string]interface{}{}
	boundary := "---------------------------" + time.Now().String()
	b := strings.NewReader(buildPayLoad(name, strings.ReplaceAll(xml, "__name__", name), svg, boundary, owner))
	resp, err := http.Post(shard+"/engine-rest/deployment/create", "multipart/form-data; boundary="+boundary, b)
	if err != nil {
		log.Println("ERROR: request to processengine ", err)
		return id, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		b, _ := io.ReadAll(resp.Body)
		return "", errors.New(string(b))
	}
	err = json.NewDecoder(resp.Body).Decode(&result)
	if err != nil {
		return id, err
	}
	definitions := result["deployedProcessDefinitions"].(map[string]interface{})
	for definitionId, _ := range definitions {
		return definitionId, nil
	}
	return "", errors.New("missing definition")
}

func buildPayLoad(name string, xml string, svg string, boundary string, owner string) string {
	segments := []string{}
	deploymentSource := "sepl"

	segments = append(segments, "Content-Disposition: form-data; name=\"data\"; "+"filename=\""+name+".bpmn\"\r\nContent-Type: text/xml\r\n\r\n"+xml+"\r\n")
	segments = append(segments, "Content-Disposition: form-data; name=\"diagram\"; "+"filename=\""+name+".svg\"\r\nContent-Type: image/svg+xml\r\n\r\n"+svg+"\r\n")
	segments = append(segments, "Content-Disposition: form-data; name=\"deployment-name\"\r\n\r\n"+name+"\r\n")
	segments = append(segments, "Content-Disposition: form-data; name=\"deployment-source\"\r\n\r\n"+deploymentSource+"\r\n")
	segments = append(segments, "Content-Disposition: form-data; name=\"tenant-id\"\r\n\r\n"+owner+"\r\n")

	return "--" + boundary + "\r\n" + strings.Join(segments, "--"+boundary+"\r\n") + "--" + boundary + "--\r\n"
}
