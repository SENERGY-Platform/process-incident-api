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
	"github.com/SENERGY-Platform/incident-api/lib/configuration"
	"github.com/pkg/errors"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"go.mongodb.org/mongo-driver/mongo/readpref"
	"time"
)

const TIMEOUT = 2 * time.Second

type Mongo struct {
	config configuration.Config
	client *mongo.Client
}

func New(ctx context.Context, config configuration.Config) (result *Mongo, err error) {
	result = &Mongo{config: config}
	result.client, err = mongo.Connect(ctx, options.Client().ApplyURI(config.MongoUrl))
	if err != nil {
		err = errors.WithStack(err)
		return
	}
	ctx, _ = context.WithTimeout(context.Background(), TIMEOUT)
	err = result.client.Ping(ctx, readpref.Primary())
	if err != nil {
		err = errors.WithStack(err)
		return
	}
	return
}

func (this *Mongo) collection() *mongo.Collection {
	return this.client.Database(this.config.MongoDatabaseName).Collection(this.config.MongoIncidentCollectionName)
}
