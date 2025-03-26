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

package controller

import (
	"errors"
	"github.com/SENERGY-Platform/process-incident-api/lib/messages"
	"github.com/SENERGY-Platform/service-commons/pkg/jwt"
	"log"
	"net/http"
)

func (this *Controller) GetIncident(token string, id string) (incident messages.IncidentMessage, err error, errCode int) {
	jwtToken, err := jwt.Parse(token)
	if err != nil {
		return incident, err, http.StatusUnauthorized
	}
	incident, exists, err := this.db.GetIncidents(id, jwtToken.GetUserId())
	if err != nil {
		log.Printf("ERROR: %+v \n", err) //prints error with stack trace if error is from github.com/pkg/errors
		return incident, errors.New("database error"), http.StatusInternalServerError
	}
	if !exists {
		return incident, errors.New("not found"), http.StatusNotFound
	}
	return incident, nil, http.StatusOK
}

func (this *Controller) FindIncidents(token string, externalTaskId string, processDefinitionId string, processInstanceId string, limit int, offset int, sortBy string, asc bool) (incidents []messages.IncidentMessage, err error, errCode int) {
	jwtToken, err := jwt.Parse(token)
	if err != nil {
		return incidents, err, http.StatusUnauthorized
	}
	incidents, err = this.db.FindIncidents(externalTaskId, processDefinitionId, processInstanceId, limit, offset, sortBy, asc, jwtToken.GetUserId())
	if err != nil {
		log.Printf("ERROR: %+v \n", err) //prints error with stack trace if error is from github.com/pkg/errors
		err = errors.New("database error")
		return incidents, err, http.StatusInternalServerError
	}
	return incidents, nil, http.StatusOK
}
