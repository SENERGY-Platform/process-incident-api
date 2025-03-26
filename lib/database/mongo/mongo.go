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

package mongo

import (
	"context"
	"github.com/SENERGY-Platform/process-incident-api/lib/configuration"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"go.mongodb.org/mongo-driver/mongo/readpref"
	"log"
	"time"
)

const TIMEOUT = 2 * time.Second

type mongoclient struct {
	config configuration.Config
	client *mongo.Client
}

func New(ctx context.Context, config configuration.Config) (result *mongoclient, err error) {
	result = &mongoclient{config: config}
	ctx, cancel := context.WithCancel(ctx)
	result.client, err = mongo.Connect(ctx, options.Client().ApplyURI(config.MongoUrl))
	if err != nil {
		return nil, err
	}
	go func() {
		<-ctx.Done()
		log.Println("disconnect mongodb")
		disconnectCtx, _ := context.WithTimeout(context.Background(), TIMEOUT)
		result.client.Disconnect(disconnectCtx)
	}()
	pingCtx, _ := context.WithTimeout(context.Background(), TIMEOUT)
	err = result.client.Ping(pingCtx, readpref.Primary())
	if err != nil {
		cancel()
		return nil, err
	}
	err = result.init()
	if err != nil {
		cancel()
		return nil, err
	}
	return result, nil
}

func (this *mongoclient) getTimeoutContext() context.Context {
	ctx, _ := context.WithTimeout(context.Background(), TIMEOUT)
	return ctx
}
