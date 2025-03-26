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
	"context"
	developerNotifications "github.com/SENERGY-Platform/developer-notifications/pkg/client"
	"github.com/SENERGY-Platform/process-incident-api/lib/configuration"
	"github.com/SENERGY-Platform/process-incident-api/lib/interfaces"
	"github.com/SENERGY-Platform/service-commons/pkg/cache"
	"log/slog"
	"os"
	"runtime/debug"
)

type Controller struct {
	config                configuration.Config
	db                    interfaces.Database
	camunda               interfaces.Camunda
	mux                   TopicMutex
	handledIncidentsCache *cache.Cache
	metrics               Metric
	devNotifications      developerNotifications.Client
	logger                *slog.Logger
}

type Metric interface {
	NotifyIncidentMessage()
}

func New(ctx context.Context, config configuration.Config, db interfaces.Database, camunda interfaces.Camunda, m Metric) (ctrl *Controller, err error) {
	logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelInfo}))
	if info, ok := debug.ReadBuildInfo(); ok {
		logger = logger.With("go-module", info.Path)
	}
	c, err := cache.New(cache.Config{}) //if the worker is scaled, the l2 must be configured with a shared memcached
	if err != nil {
		return nil, err
	}
	ctrl = &Controller{config: config, camunda: camunda, db: db, metrics: m, logger: logger, handledIncidentsCache: c}
	if config.DeveloperNotificationUrl != "" && config.DeveloperNotificationUrl != "-" {
		ctrl.devNotifications = developerNotifications.New(config.DeveloperNotificationUrl)
	}
	return ctrl, nil
}
