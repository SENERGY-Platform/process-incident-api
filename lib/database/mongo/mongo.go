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
	"go.mongodb.org/mongo-driver/x/bsonx"
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
		err = errors.WithStack(err)
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
		err = errors.WithStack(err)
		return nil, err
	}
	err = result.init()
	if err != nil {
		cancel()
		return nil, err
	}
	return result, nil
}

func (this *mongoclient) init() error {
	err := this.ensureIndex(this.collection(), "id_index", "id", true, true)
	if err != nil {
		return err
	}
	err = this.ensureIndex(this.collection(), "external_task_id_index", "external_task_id", true, false)
	if err != nil {
		return err
	}
	err = this.ensureIndex(this.collection(), "process_instance_id_index", "process_instance_id", true, false)
	if err != nil {
		return err
	}
	err = this.ensureIndex(this.collection(), "process_definition_id_index", "process_definition_id", true, false)
	if err != nil {
		return err
	}
	return nil
}

func (this *mongoclient) ensureIndex(collection *mongo.Collection, indexname string, indexKey string, asc bool, unique bool) error {
	ctx, _ := context.WithTimeout(context.Background(), TIMEOUT)
	var direction int32 = -1
	if asc {
		direction = 1
	}
	_, err := collection.Indexes().CreateOne(ctx, mongo.IndexModel{
		Keys:    bsonx.Doc{{indexKey, bsonx.Int32(direction)}},
		Options: options.Index().SetName(indexname).SetUnique(unique),
	})
	return errors.WithStack(err)
}

func (this *mongoclient) collection() *mongo.Collection {
	return this.client.Database(this.config.MongoDatabaseName).Collection(this.config.MongoIncidentCollectionName)
}
