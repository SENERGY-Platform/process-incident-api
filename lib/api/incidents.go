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

package api

import (
	"encoding/json"
	"github.com/SENERGY-Platform/process-incident-api/lib/api/util"
	"github.com/SENERGY-Platform/process-incident-api/lib/configuration"
	"github.com/SENERGY-Platform/process-incident-api/lib/interfaces"
	"github.com/SENERGY-Platform/process-incident-api/lib/messages"
	"log"
	"net/http"
	"runtime/debug"
)

func init() {
	endpoints = append(endpoints, &IncidentsEndpoints{})
}

type IncidentsEndpoints struct{}

// GetIncident godoc
// @Summary      get incident
// @Description  get incident
// @Tags         incidents
// @Produce      json
// @Security Bearer
// @Param        id path string true "Incident Id"
// @Success      200 {object} messages.IncidentMessage
// @Failure      400
// @Failure      401
// @Failure      403
// @Failure      404
// @Failure      500
// @Router       /incidents/{id} [GET]
func (this *IncidentsEndpoints) GetIncident(config configuration.Config, ctrl interfaces.Controller, router *http.ServeMux) {
	router.HandleFunc("GET /incidents/{id}", func(writer http.ResponseWriter, request *http.Request) {
		id := request.PathValue("id")
		incident, err, code := ctrl.GetIncident(util.GetAuthToken(request), id)
		if err != nil {
			http.Error(writer, err.Error(), code)
			return
		}
		writer.Header().Set("Content-Type", "application/json; charset=utf-8")
		err = json.NewEncoder(writer).Encode(incident)
		if err != nil {
			debug.PrintStack()
			log.Println("ERROR: ", err)
		}
	})
}

// ListIncidents godoc
// @Summary      list incidents
// @Description  list incidents
// @Tags         incidents
// @Produce      json
// @Security Bearer
// @Param        limit query integer false "limits size of result; default 100"
// @Param        offset query integer false "offset to be used in combination with limit, default 0"
// @Param        sort query string false "default id.asc, sortable by id, external_task_id, process_instance_id, process_definition_id, time"
// @Param        process_definition_id query string false "filter by process_definition_id"
// @Param        process_instance_id query string false "filter by process_instance_id"
// @Param        external_task_id query string false "filter by external_task_id"
// @Success      200 {array}  messages.IncidentMessage
// @Failure      400
// @Failure      401
// @Failure      403
// @Failure      404
// @Failure      500
// @Router       /incidents [GET]
func (this *IncidentsEndpoints) ListIncidents(config configuration.Config, ctrl interfaces.Controller, router *http.ServeMux) {
	router.HandleFunc("GET /incidents", func(writer http.ResponseWriter, request *http.Request) {
		processDefinitionId := request.URL.Query().Get("process_definition_id")
		processInstanceId := request.URL.Query().Get("process_instance_id")
		taskId := request.URL.Query().Get("external_task_id")

		limit, err := util.ParseLimit(request.URL.Query().Get("limit"))
		if err != nil {
			http.Error(writer, err.Error(), http.StatusBadRequest)
			return
		}
		offset, err := util.ParseOffset(request.URL.Query().Get("offset"))
		if err != nil {
			http.Error(writer, err.Error(), http.StatusBadRequest)
			return
		}
		sortField, sortAsc, err := util.ParseSort(request.URL.Query().Get("sort"), []string{"id", "external_task_id", "process_instance_id", "process_definition_id", "time"})
		if err != nil {
			http.Error(writer, err.Error(), http.StatusBadRequest)
			return
		}

		incidents, err, code := ctrl.FindIncidents(util.GetAuthToken(request), taskId, processDefinitionId, processInstanceId, limit, offset, sortField, sortAsc)
		if err != nil {
			http.Error(writer, err.Error(), code)
			return
		}
		if incidents == nil {
			incidents = []messages.IncidentMessage{} //ensure json is '[]' and not 'null'
		}
		writer.Header().Set("Content-Type", "application/json; charset=utf-8")
		err = json.NewEncoder(writer).Encode(incidents)
		if err != nil {
			debug.PrintStack()
			log.Println("ERROR: ", err)
		}
	})
}

// CreateIncident godoc
// @Summary      create incident
// @Description  create incident, user must be admin
// @Tags         incidents
// @Produce      json
// @Security Bearer
// @Param        message body messages.IncidentMessage true "Incident"
// @Success      200
// @Failure      400
// @Failure      401
// @Failure      403
// @Failure      404
// @Failure      500
// @Router       /incidents [POST]
func (this *IncidentsEndpoints) CreateIncident(config configuration.Config, ctrl interfaces.Controller, router *http.ServeMux) {
	router.HandleFunc("POST /incidents", func(writer http.ResponseWriter, request *http.Request) {
		incident := messages.IncidentMessage{}
		err := json.NewDecoder(request.Body).Decode(&incident)
		if err != nil {
			http.Error(writer, err.Error(), http.StatusBadRequest)
			return
		}
		err, code := ctrl.CreateIncident(util.GetAuthToken(request), incident)
		if err != nil {
			http.Error(writer, err.Error(), code)
			return
		}
		writer.WriteHeader(http.StatusOK)
	})
}

// SetIncidentHandler godoc
// @Summary      set on incident handler
// @Description  set on incident handler, user must be admin
// @Tags         incidents
// @Produce      json
// @Security Bearer
// @Param        message body messages.OnIncident true "Incident-Handler"
// @Success      200
// @Failure      400
// @Failure      401
// @Failure      403
// @Failure      404
// @Failure      500
// @Router       /on-incident-handler [PUT]
func (this *IncidentsEndpoints) SetIncidentHandler(config configuration.Config, ctrl interfaces.Controller, router *http.ServeMux) {
	router.HandleFunc("PUT /on-incident-handler", func(writer http.ResponseWriter, request *http.Request) {
		handler := messages.OnIncident{}
		err := json.NewDecoder(request.Body).Decode(&handler)
		if err != nil {
			http.Error(writer, err.Error(), http.StatusBadRequest)
			return
		}
		err, code := ctrl.SetOnIncidentHandler(util.GetAuthToken(request), handler)
		if err != nil {
			http.Error(writer, err.Error(), code)
			return
		}
		writer.WriteHeader(http.StatusOK)
	})
}

// DeleteIncidentByProcessDefinitionId godoc
// @Summary      delete incidents by process-definition id
// @Description  delete incidents by process-definition id, user must be admin
// @Tags         incidents
// @Produce      json
// @Security Bearer
// @Param        id path string true "process-definition id"
// @Success      200
// @Failure      400
// @Failure      401
// @Failure      403
// @Failure      404
// @Failure      500
// @Router       /process-definitions/{id} [DELETE]
func (this *IncidentsEndpoints) DeleteIncidentByProcessDefinitionId(config configuration.Config, ctrl interfaces.Controller, router *http.ServeMux) {
	router.HandleFunc("DELETE /process-definitions/{id}", func(writer http.ResponseWriter, request *http.Request) {
		id := request.PathValue("id")
		err, code := ctrl.DeleteIncidentByProcessDefinitionId(util.GetAuthToken(request), id)
		if err != nil {
			http.Error(writer, err.Error(), code)
			return
		}
		writer.WriteHeader(http.StatusOK)
	})
}

// DeleteIncidentByProcessInstanceId godoc
// @Summary      delete incidents by process-instance id
// @Description  delete incidents by process-instance id, user must be admin
// @Tags         incidents
// @Produce      json
// @Security Bearer
// @Param        id path string true "process-instance id"
// @Success      200
// @Failure      400
// @Failure      401
// @Failure      403
// @Failure      404
// @Failure      500
// @Router       /process-instances/{id} [DELETE]
func (this *IncidentsEndpoints) DeleteIncidentByProcessInstanceId(config configuration.Config, ctrl interfaces.Controller, router *http.ServeMux) {
	router.HandleFunc("DELETE /process-instances/{id}", func(writer http.ResponseWriter, request *http.Request) {
		id := request.PathValue("id")
		err, code := ctrl.DeleteIncidentByProcessInstanceId(util.GetAuthToken(request), id)
		if err != nil {
			http.Error(writer, err.Error(), code)
			return
		}
		writer.WriteHeader(http.StatusOK)
	})
}
