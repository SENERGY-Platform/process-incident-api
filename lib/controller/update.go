/*
 * Copyright 2025 InfAI (CC SES)
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
	"errors"
	"fmt"
	developerNotifications "github.com/SENERGY-Platform/developer-notifications/pkg/client"
	"github.com/SENERGY-Platform/process-incident-api/lib/messages"
	"github.com/SENERGY-Platform/process-incident-api/lib/notification"
	"github.com/SENERGY-Platform/service-commons/pkg/cache"
	"github.com/SENERGY-Platform/service-commons/pkg/jwt"
	"log"
	"net/http"
	"runtime/debug"
	"time"
)

func (this *Controller) CreateIncident(token string, incident messages.Incident) (err error, code int) {
	jwtToken, err := jwt.Parse(token)
	if err != nil {
		return err, http.StatusUnauthorized
	}
	if !jwtToken.IsAdmin() {
		return errors.New("only admins may create incidents"), http.StatusForbidden
	}
	err = this.ValidateIncident(incident)
	if err != nil {
		return err, http.StatusBadRequest
	}
	topic := incident.ProcessDefinitionId + "+" + incident.ProcessInstanceId
	this.mux.Lock(topic)
	defer this.mux.Unlock(topic)
	//for every process instance an incident may only be handled once every 5 min
	//use the cache.Use method to do incident handling, only if the process instance is not found in cache
	//incident.ProcessInstanceId should be enough as key but existing tests would fail, so the incident.ProcessDefinitionId is added
	_, err = cache.Use[string](this.handledIncidentsCache, topic, func() (string, error) {
		return "", this.createIncident(incident)
	}, cache.NoValidation, 5*time.Minute)
	if err != nil {
		return err, http.StatusInternalServerError
	}
	return err, http.StatusOK
}

func (this *Controller) ValidateIncident(incident messages.Incident) error {
	if incident.ProcessDefinitionId == "" {
		return errors.New("missing process definition id")
	}
	if incident.ProcessInstanceId == "" {
		return errors.New("missing process instance id")
	}
	if incident.Id == "" {
		return errors.New("missing incident id")
	}
	if incident.TenantId == "" {
		return errors.New("missing tenant id")
	}
	return nil
}

func (this *Controller) createIncident(incident messages.Incident) (err error) {
	this.metrics.NotifyIncidentMessage()
	handling, registeredHandling, err := this.db.GetOnIncident(incident.ProcessDefinitionId)
	if err != nil {
		log.Println("ERROR: ", err)
		debug.PrintStack()
		return err
	}
	name, err := this.camunda.GetProcessName(incident.ProcessDefinitionId, incident.TenantId)
	if err != nil {
		this.logger.Error("unable to get process name", "snrgy-log-type", "warning", "error", err.Error())
		incident.DeploymentName = incident.ProcessDefinitionId
	} else {
		incident.DeploymentName = name
	}
	this.logger.Info("process-incident", "snrgy-log-type", "process-incident", "error", incident.ErrorMessage, "user", incident.TenantId, "deployment-name", incident.DeploymentName, "process-definition-id", incident.ProcessDefinitionId, "process-instance-id", incident.ProcessInstanceId)
	if incident.TenantId != "" {
		if !registeredHandling || handling.Notify {
			msg := notification.Message{
				UserId:  incident.TenantId,
				Title:   "Process-Incident in " + incident.DeploymentName,
				Message: incident.ErrorMessage,
				Topic:   notification.Topic,
			}
			if registeredHandling && handling.Restart {
				msg.Message = msg.Message + "\n\nprocess will be restarted"
			}
			this.Notify(msg)
		}
	}
	err = this.camunda.StopProcessInstance(incident.ProcessInstanceId, incident.TenantId)
	if err != nil {
		return err
	}
	err = this.db.SaveIncident(incident)
	if err != nil {
		return err
	}
	if registeredHandling && handling.Restart {
		err = this.camunda.StartProcess(incident.ProcessDefinitionId, incident.TenantId)
		if err != nil {
			this.logger.Error("unable to restart process", "snrgy-log-type", "process-incident", "error", err.Error(), "user", incident.TenantId, "deployment-name", incident.DeploymentName, "process-definition-id", incident.ProcessDefinitionId, "process-instance-id", incident.ProcessInstanceId)
			if incident.TenantId != "" {
				this.Notify(notification.Message{
					UserId:  incident.TenantId,
					Title:   "ERROR: unable to restart process after incident in: " + incident.DeploymentName,
					Message: fmt.Sprintf("Restart-Error: %v \n\n Incident: %v \n", err, incident.ErrorMessage),
					Topic:   notification.Topic,
				})
			}
		}
	}
	return nil
}

func (this *Controller) DeleteIncidentByProcessInstanceId(token string, id string) (err error, code int) {
	jwtToken, err := jwt.Parse(token)
	if err != nil {
		return err, http.StatusUnauthorized
	}
	if !jwtToken.IsAdmin() {
		return errors.New("only admins may create incidents"), http.StatusForbidden
	}
	err = this.db.DeleteIncidentByInstanceId(id)
	if err != nil {
		return err, http.StatusInternalServerError
	}
	return nil, http.StatusOK
}

func (this *Controller) DeleteIncidentByProcessDefinitionId(token string, id string) (err error, code int) {
	jwtToken, err := jwt.Parse(token)
	if err != nil {
		return err, http.StatusUnauthorized
	}
	if !jwtToken.IsAdmin() {
		return errors.New("only admins may create incidents"), http.StatusForbidden
	}
	err = this.db.DeleteByDefinitionId(id)
	if err != nil {
		return err, http.StatusInternalServerError
	}
	return nil, http.StatusOK
}

func (this *Controller) SetOnIncidentHandler(token string, handler messages.OnIncident) (err error, code int) {
	jwtToken, err := jwt.Parse(token)
	if err != nil {
		return err, http.StatusUnauthorized
	}
	if !jwtToken.IsAdmin() {
		return errors.New("only admins may create incidents"), http.StatusForbidden
	}
	if handler.ProcessDefinitionId == "" {
		return errors.New("missing process_definition_id"), http.StatusBadRequest
	}
	err = this.db.SaveOnIncident(handler)
	if err != nil {
		return err, http.StatusInternalServerError
	}
	return nil, http.StatusOK
}

func (this *Controller) Notify(msg notification.Message) {
	_ = notification.Send(this.config.NotificationUrl, msg)
	if this.devNotifications != nil {
		go func() {
			if this.config.Debug {
				log.Println("DEBUG: send developer-notification")
			}
			err := this.devNotifications.SendMessage(developerNotifications.Message{
				Sender: "github.com/SENERGY-Platform/process-incident-worker",
				Title:  "Process-Incident-User-Notification",
				Tags:   []string{"process-incident", "user-notification", msg.UserId},
				Body:   fmt.Sprintf("Notification For %v\nTitle: %v\nMessage: %v\n", msg.UserId, msg.Title, msg.Message),
			})
			if err != nil {
				log.Println("ERROR: unable to send developer-notification", err)
			}
		}()
	}
}
