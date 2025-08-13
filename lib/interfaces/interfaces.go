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

package interfaces

import (
	"context"

	"github.com/SENERGY-Platform/process-incident-api/lib/configuration"
	"github.com/SENERGY-Platform/process-incident-api/lib/messages"
)

type Controller interface {
	GetIncident(token string, id string) (incident messages.IncidentMessage, err error, errCode int)
	FindIncidents(token string, externalTaskId string, processDefinitionId string, processInstanceId string, limit int, offset int, sortBy string, asc bool) (incidents []messages.IncidentMessage, err error, errCode int)
	CreateIncident(token string, incident messages.Incident) (err error, code int)
	SetOnIncidentHandler(token string, incident messages.OnIncident) (err error, code int)
	DeleteIncidentByProcessInstanceId(token string, id string) (err error, code int)
	DeleteIncidentByProcessDefinitionId(token string, id string) (err error, code int)
}

type Database interface {
	GetIncidents(id string, user string) (incident messages.IncidentMessage, exists bool, err error)
	FindIncidents(externalTaskId string, processDefinitionId string, processInstanceId string, limit int, offset int, sortBy string, asc bool, user string) (incidents []messages.IncidentMessage, err error)
	DeleteByDefinitionId(id string) error
	SaveIncident(incident messages.Incident) error
	DeleteIncidentByInstanceId(id string) error
	SaveOnIncident(handler messages.OnIncident) error
	GetOnIncident(definitionId string) (incident messages.OnIncident, exists bool, err error)
}

type DatabaseFactory interface {
	Get(ctx context.Context, config configuration.Config) (Database, error)
}

type ApiFactory interface {
	Start(ctx context.Context, config configuration.Config, ctrl Controller) error
}

type CamundaFactory interface {
	Get(ctx context.Context, config configuration.Config) (Camunda, error)
}

type Camunda interface {
	StopProcessInstance(id string, tenantId string) (err error)
	GetProcessName(id string, tenantId string) (string, error)
	StartProcess(processDefinitionId string, userId string) (err error)
	StartProcessWithBusinessKey(processDefinitionId string, businessKey string, userId string) (err error)
	GetIncidents() (result []messages.CamundaIncident, err error)
	GetHistoricProcessInstance(id string, userId string) (result messages.HistoricProcessInstance, err error)
}
