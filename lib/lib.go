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

package lib

import (
	"context"
	"github.com/SENERGY-Platform/process-incident-api/lib/api"
	"github.com/SENERGY-Platform/process-incident-api/lib/camunda"
	"github.com/SENERGY-Platform/process-incident-api/lib/camundasource"
	"github.com/SENERGY-Platform/process-incident-api/lib/configuration"
	"github.com/SENERGY-Platform/process-incident-api/lib/controller"
	"github.com/SENERGY-Platform/process-incident-api/lib/database"
	"github.com/SENERGY-Platform/process-incident-api/lib/interfaces"
	"github.com/SENERGY-Platform/process-incident-api/lib/metrics"
)

func Start(ctx context.Context, config configuration.Config) (err error) {
	return StartWith(ctx, config, api.Factory, database.Factory, camunda.Factory)
}

func StartWith(parentCtx context.Context, config configuration.Config, api interfaces.ApiFactory, database interfaces.DatabaseFactory, camunda interfaces.CamundaFactory) (err error) {
	ctx, cancel := context.WithCancel(parentCtx)
	databaseInstance, err := database.Get(ctx, config)
	if err != nil {
		cancel()
		return err
	}
	camundaInstance, err := camunda.Get(ctx, config)
	if err != nil {
		cancel()
		return err
	}
	m := metrics.New().Serve(ctx, config.MetricsPort)
	ctrl, err := controller.New(ctx, config, databaseInstance, camundaInstance, m)
	if err != nil {
		cancel()
		return err
	}
	err = api.Start(ctx, config, ctrl)
	if err != nil {
		cancel()
		return err
	}
	err = camundasource.Start(ctx, config, camundaInstance, ctrl)
	if err != nil {
		cancel()
		return err
	}
	return nil
}
