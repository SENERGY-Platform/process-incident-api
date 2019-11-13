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
	"github.com/SENERGY-Platform/incident-api/lib/configuration"
	"github.com/SENERGY-Platform/incident-api/lib/messages"
)

type Controller interface {
	GetIncident(id string) (incident messages.IncidentMessage, err error, errCode int)
	FindIncidents(processDefinitionId string, processInstanceId string, limit int, offset int, sortBy string, asc bool) (incidents []messages.IncidentMessage, err error)
}

type Database interface {
	GetIncidents(id string) (incident messages.IncidentMessage, exists bool, err error)
	FindIncidents(processDefinitionId string, processInstanceId string, limit int, offset int, sortBy string, asc bool) (incidents []messages.IncidentMessage, err error)
}

type DatabaseFactory interface {
	Get(ctx context.Context, config configuration.Config) (Database, error)
}

type ApiFactory interface {
	Start(ctx context.Context, config configuration.Config, ctrl Controller) error
}
