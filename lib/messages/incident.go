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

package messages

import "time"

type IncidentMessage = Incident
type Incident struct {
	Id                  string    `json:"id" bson:"id"`
	MsgVersion          int64     `json:"msg_version,omitempty" bson:"msg_version,omitempty"` //from version 3 onward will be set in KafkaIncidentsCommand and be copied to this field
	ExternalTaskId      string    `json:"external_task_id" bson:"external_task_id"`
	ProcessInstanceId   string    `json:"process_instance_id" bson:"process_instance_id"`
	ProcessDefinitionId string    `json:"process_definition_id" bson:"process_definition_id"`
	WorkerId            string    `json:"worker_id" bson:"worker_id"`
	ErrorMessage        string    `json:"error_message" bson:"error_message"`
	Time                time.Time `json:"time" bson:"time"`
	TenantId            string    `json:"tenant_id" bson:"tenant_id"`
	DeploymentName      string    `json:"deployment_name" bson:"deployment_name"`
	BusinessKey         string    `json:"business_key" bson:"business_key"`
}

type OnIncident struct {
	ProcessDefinitionId string `json:"process_definition_id" bson:"process_definition_id"`
	Restart             bool   `json:"restart" bson:"restart"`
	Notify              bool   `json:"notify" bson:"notify"`
}

type CamundaIncident struct {
	Id                  string `json:"id"`
	ProcessDefinitionId string `json:"processDefinitionId"`
	ProcessInstanceId   string `json:"processInstanceId"`
	ExecutionId         string `json:"executionId"`
	IncidentTimestamp   string `json:"incidentTimestamp"`
	IncidentType        string `json:"incidentType"`
	ActivityId          string `json:"activityId"`
	CauseIncidentId     string `json:"causeIncidentId"`
	RootCauseIncidentId string `json:"rootCauseIncidentId"`
	Configuration       string `json:"configuration"`
	TenantId            string `json:"tenantId"`
	IncidentMessage     string `json:"incidentMessage"`
	JobDefinitionId     string `json:"jobDefinitionId"`
}

type HistoricProcessInstance struct {
	Id                       string  `json:"id"`
	SuperProcessInstanceId   string  `json:"superProcessInstanceId"`
	SuperCaseInstanceId      string  `json:"superCaseInstanceId"`
	CaseInstanceId           string  `json:"caseInstanceId"`
	ProcessDefinitionName    string  `json:"processDefinitionName"`
	ProcessDefinitionKey     string  `json:"processDefinitionKey"`
	ProcessDefinitionVersion float64 `json:"processDefinitionVersion"`
	ProcessDefinitionId      string  `json:"processDefinitionId"`
	BusinessKey              string  `json:"businessKey"`
	StartTime                string  `json:"startTime"`
	EndTime                  string  `json:"endTime"`
	DurationInMillis         float64 `json:"durationInMillis"`
	StartUserId              string  `json:"startUserId"`
	StartActivityId          string  `json:"startActivityId"`
	DeleteReason             string  `json:"deleteReason"`
	TenantId                 string  `json:"tenantId"`
	State                    string  `json:"state"`
}
