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
	"context"
	"errors"
	"fmt"
	"github.com/SENERGY-Platform/process-incident-api/lib/api/util"
	"github.com/SENERGY-Platform/process-incident-api/lib/configuration"
	"github.com/SENERGY-Platform/process-incident-api/lib/interfaces"
	"github.com/SENERGY-Platform/service-commons/pkg/accesslog"
	"log"
	"net/http"
	"reflect"
	"runtime/debug"
	"time"
)

//go:generate go tool swag init -o ../../docs --parseDependency -d . -g api.go

type EndpointMethod = func(config configuration.Config, ctrl interfaces.Controller, router *http.ServeMux)

var endpoints = []interface{}{} //list of objects with EndpointMethod

type FactoryType struct{}

var Factory = &FactoryType{}

func (this *FactoryType) Start(ctx context.Context, config configuration.Config, ctrl interfaces.Controller) error {
	return Start(ctx, config, ctrl)
}

func Start(ctx context.Context, config configuration.Config, ctrl interfaces.Controller) (err error) {
	defer func() {
		if r := recover(); r != nil {
			err = errors.New(fmt.Sprint(r))
		}
	}()
	router := GetRouter(config, ctrl)
	server := &http.Server{Addr: ":" + config.ApiPort, Handler: router, WriteTimeout: 10 * time.Second, ReadTimeout: 2 * time.Second, ReadHeaderTimeout: 2 * time.Second}
	go func() {
		log.Println("listening on ", server.Addr)
		if err := server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			debug.PrintStack()
			log.Fatal("FATAL:", err)
		}
	}()
	go func() {
		<-ctx.Done()
		log.Println("api shutdown", server.Shutdown(context.Background()))
	}()
	return
}

// GetRouter doc
// @title         Incidents API
// @version       0.1
// @license.name  Apache 2.0
// @license.url   http://www.apache.org/licenses/LICENSE-2.0.html
// @BasePath  /
// @securityDefinitions.apikey Bearer
// @in header
// @name Authorization
// @description Type "Bearer" followed by a space and JWT token.
func GetRouter(config configuration.Config, control interfaces.Controller) http.Handler {
	router := http.NewServeMux()
	log.Println("add heart beat endpoint")
	router.HandleFunc("GET /{$}", func(writer http.ResponseWriter, request *http.Request) {
		writer.WriteHeader(http.StatusOK)
	})
	for _, e := range endpoints {
		for name, call := range getEndpointMethods(e) {
			log.Println("add endpoint " + name)
			call(config, control, router)
		}
	}
	log.Println("add cors")
	handler := util.NewCors(router)
	if config.ApiLog {
		log.Println("add logging")
		handler = accesslog.New(handler)
	}
	return handler
}

func getEndpointMethods(e interface{}) map[string]EndpointMethod {
	result := map[string]EndpointMethod{}
	objRef := reflect.ValueOf(e)
	methodCount := objRef.NumMethod()
	for i := 0; i < methodCount; i++ {
		m := objRef.Method(i)
		f, ok := m.Interface().(EndpointMethod)
		if ok {
			name := getTypeName(objRef.Type()) + "::" + objRef.Type().Method(i).Name
			result[name] = f
		}
	}
	return result
}

func getTypeName(t reflect.Type) (res string) {
	for t.Kind() == reflect.Ptr {
		t = t.Elem()
	}
	return t.Name()
}
