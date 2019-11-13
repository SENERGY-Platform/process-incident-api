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
	"github.com/SENERGY-Platform/incident-api/lib/api"
	"github.com/SENERGY-Platform/incident-api/lib/configuration"
	"github.com/SENERGY-Platform/incident-api/lib/controller"
	"github.com/SENERGY-Platform/incident-api/lib/database"
	"github.com/SENERGY-Platform/incident-api/lib/interfaces"
)

func Start(ctx context.Context, config configuration.Config) (err error) {
	return StartWith(ctx, config, api.Factory, database.Factory)
}

func StartWith(parentCtx context.Context, config configuration.Config, api interfaces.ApiFactory, database interfaces.DatabaseFactory) (err error) {
	ctx, cancel := context.WithCancel(parentCtx)
	databaseInstance, err := database.Get(ctx, config)
	if err != nil {
		cancel()
		return err
	}
	ctrl := controller.New(ctx, config, databaseInstance)
	err = api.Start(ctx, config, ctrl)
	if err != nil {
		cancel()
		return err
	}
	return nil
}
