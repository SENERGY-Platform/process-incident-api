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

package tests

import (
	"context"
	"github.com/SENERGY-Platform/process-incident-api/lib/configuration"
	"github.com/SENERGY-Platform/process-incident-api/lib/messages"
	"github.com/pkg/errors"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"testing"
	"time"
)

func createTestIncidents(t *testing.T, config configuration.Config, incidentMessages []messages.IncidentMessage) {
	for _, incident := range incidentMessages {
		createTestIncident(t, config, incident)
	}
}

func createTestIncident(t *testing.T, config configuration.Config, incident messages.IncidentMessage) {
	ctx, _ := context.WithTimeout(context.Background(), 2*time.Second)
	client, err := mongo.Connect(ctx, options.Client().ApplyURI(config.MongoUrl))
	if err != nil {
		err = errors.WithStack(err)
		t.Fatalf("ERROR: %+v", err)
		return
	}
	_, err = client.Database(config.MongoDatabaseName).Collection(config.MongoIncidentCollectionName).
		ReplaceOne(ctx, bson.M{"id": incident.Id}, incident, options.Replace().SetUpsert(true))
	if err != nil {
		t.Fatalf("ERROR: %+v", err)
	}
}
