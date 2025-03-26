/*
 * Copyright 2024 InfAI (CC SES)
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

package camundasource

import (
	"context"
	"github.com/SENERGY-Platform/process-incident-api/lib/client"
	"github.com/SENERGY-Platform/process-incident-api/lib/configuration"
	"github.com/SENERGY-Platform/process-incident-api/lib/controller"
	"github.com/SENERGY-Platform/process-incident-api/lib/interfaces"
	"github.com/SENERGY-Platform/process-incident-api/lib/messages"
	"log"
	"time"
)

func Start(ctx context.Context, config configuration.Config, camunda interfaces.Camunda, ctrl *controller.Controller) error {
	interval := time.Second
	var err error
	if config.CamundaIncidentRequestInterval != "" && config.CamundaIncidentRequestInterval != "-" {
		interval, err = time.ParseDuration(config.CamundaIncidentRequestInterval)
		if err != nil {
			return err
		}
	} else {
		return nil
	}
	go func() {
		for {
			select {
			case <-ctx.Done():
				return
			default:
				incidents, err := camunda.GetIncidents()
				if err != nil {
					log.Println("WARNING: unable to load camunda incidents", err)
					continue
				}
				for _, incident := range incidents {
					err, _ = ctrl.CreateIncident(client.InternalAdminToken, messages.Incident{
						Id:                  incident.Id,
						MsgVersion:          3,
						ExternalTaskId:      incident.ActivityId,
						ProcessInstanceId:   incident.ProcessInstanceId,
						ProcessDefinitionId: incident.ProcessDefinitionId,
						WorkerId:            "process-incident-worker",
						ErrorMessage:        incident.IncidentMessage,
						Time:                time.Now(),
						TenantId:            incident.TenantId,
					})
					if err != nil {
						log.Println("WARNING: unable to handle camunda incidents", err)
						continue
					}
				}
				time.Sleep(interval)
			}
		}
	}()
	return nil
}
