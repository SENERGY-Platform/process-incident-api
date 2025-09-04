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

package client

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"

	"github.com/SENERGY-Platform/process-incident-api/lib/interfaces"
	"github.com/SENERGY-Platform/process-incident-api/lib/messages"
)

type Client interface {
	interfaces.Controller
}

func New(serverUrl string) (client *ClientImpl) {
	return &ClientImpl{serverUrl: serverUrl}
}

type ClientImpl struct {
	serverUrl string
}

type OnIncident = messages.OnIncident
type IncidentMessage = messages.IncidentMessage

func (this *ClientImpl) GetIncident(token string, id string) (incident messages.IncidentMessage, err error, errCode int) {
	req, err := http.NewRequest(http.MethodGet, fmt.Sprintf("%v/incidents/%v", this.serverUrl, url.PathEscape(id)), nil)
	if err != nil {
		return incident, err, 0
	}
	return do[messages.IncidentMessage](token, req)
}

func (this *ClientImpl) FindIncidents(token string, externalTaskId string, processDefinitionId string, processInstanceId string, limit int, offset int, sortBy string, asc bool) (incidents []messages.IncidentMessage, err error, code int) {
	query := url.Values{}
	query.Add("limit", strconv.Itoa(limit))
	query.Add("offset", strconv.Itoa(offset))
	sort := sortBy
	if asc {
		sort = sort + ".asc"
	} else {
		sort = sort + ".desc"
	}
	query.Add("sort", sort)
	if externalTaskId != "" {
		query.Add("external_task_id", externalTaskId)
	}
	if processDefinitionId != "" {
		query.Add("process_definition_id", processDefinitionId)
	}
	if processInstanceId != "" {
		query.Add("process_instance_id", processInstanceId)
	}
	req, err := http.NewRequest(http.MethodGet, fmt.Sprintf("%v/incidents?"+query.Encode(), this.serverUrl), nil)
	if err != nil {
		return incidents, err, 0
	}
	return do[[]messages.IncidentMessage](token, req)
}

func (this *ClientImpl) CreateIncident(token string, incident messages.Incident) (err error, code int) {
	body, err := json.Marshal(incident)
	if err != nil {
		return err, 0
	}
	req, err := http.NewRequest(http.MethodPost, fmt.Sprintf("%v/incidents", this.serverUrl), bytes.NewBuffer(body))
	if err != nil {
		return err, 0
	}
	return doVoid(token, req)
}

func (this *ClientImpl) SetOnIncidentHandler(token string, incident messages.OnIncident) (err error, code int) {
	body, err := json.Marshal(incident)
	if err != nil {
		return err, 0
	}
	req, err := http.NewRequest(http.MethodPut, fmt.Sprintf("%v/on-incident-handler", this.serverUrl), bytes.NewBuffer(body))
	if err != nil {
		return err, 0
	}
	return doVoid(token, req)
}

func (this *ClientImpl) DeleteIncidentByProcessInstanceId(token string, id string) (err error, code int) {
	req, err := http.NewRequest(http.MethodDelete, fmt.Sprintf("%v/process-instances/%v", this.serverUrl, url.PathEscape(id)), nil)
	if err != nil {
		return err, 0
	}
	return doVoid(token, req)
}

func (this *ClientImpl) DeleteIncidentByProcessDefinitionId(token string, id string) (err error, code int) {
	req, err := http.NewRequest(http.MethodDelete, fmt.Sprintf("%v/process-definitions/%v", this.serverUrl, url.PathEscape(id)), nil)
	if err != nil {
		return err, 0
	}
	return doVoid(token, req)
}

func do[T any](token string, req *http.Request) (result T, err error, code int) {
	req.Header.Set("Authorization", token)
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return result, err, http.StatusInternalServerError
	}
	defer resp.Body.Close()
	if resp.StatusCode > 299 {
		temp, _ := io.ReadAll(resp.Body) //read error response end ensure that resp.Body is read to EOF
		return result, fmt.Errorf("unexpected statuscode %v: %v", resp.StatusCode, string(temp)), resp.StatusCode
	}
	err = json.NewDecoder(resp.Body).Decode(&result)
	if err != nil {
		_, _ = io.ReadAll(resp.Body) //ensure resp.Body is read to EOF
		return result, err, http.StatusInternalServerError
	}
	return result, nil, resp.StatusCode
}

func doVoid(token string, req *http.Request) (err error, code int) {
	req.Header.Set("Authorization", token)
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return err, http.StatusInternalServerError
	}
	defer resp.Body.Close()
	if resp.StatusCode > 299 {
		temp, _ := io.ReadAll(resp.Body) //read error response end ensure that resp.Body is read to EOF
		return fmt.Errorf("unexpected statuscode %v: %v", resp.StatusCode, string(temp)), resp.StatusCode
	}
	return nil, resp.StatusCode
}
