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
	"github.com/SENERGY-Platform/incident-api/lib/api/util"
	"github.com/SENERGY-Platform/incident-api/lib/configuration"
	"github.com/SENERGY-Platform/incident-api/lib/interfaces"
	"github.com/julienschmidt/httprouter"
	"log"
	"net/http"
	"runtime/debug"
)

func init() {
	endpoints = append(endpoints, DeviceEndpoints)
}

func DeviceEndpoints(config configuration.Config, ctrl interfaces.Controller, router *httprouter.Router) {
	resource := "/incidents"

	router.GET(resource+"/:id", func(writer http.ResponseWriter, request *http.Request, params httprouter.Params) {
		id := params.ByName("id")
		incident, err, code := ctrl.GetIncident(id)
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

	/*
			query-parameter:
				- process_instance_id
				- process_definition_id
				- limit
		        - offset
				- sort
	*/
	router.GET(resource, func(writer http.ResponseWriter, request *http.Request, params httprouter.Params) {
		processDefinitionId := request.URL.Query().Get("process_definition_id")
		processInstanceId := request.URL.Query().Get("process_instance_id")

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
		sortField, sortAsc, err := util.ParseSort(request.URL.Query().Get("sort"), []string{"id"})
		if err != nil {
			http.Error(writer, err.Error(), http.StatusBadRequest)
			return
		}

		incidents, err := ctrl.FindIncidents(processDefinitionId, processInstanceId, limit, offset, sortField, sortAsc)
		if err != nil {
			http.Error(writer, err.Error(), http.StatusInternalServerError)
			return
		}
		writer.Header().Set("Content-Type", "application/json; charset=utf-8")
		err = json.NewEncoder(writer).Encode(incidents)
		if err != nil {
			debug.PrintStack()
			log.Println("ERROR: ", err)
		}
	})

}
