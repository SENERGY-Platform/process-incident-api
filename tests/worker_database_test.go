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
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"reflect"
	"testing"
	"time"
)

func checkIncidentInDatabase(t *testing.T, config configuration.Config, expected messages.Incident) {
	ctx, _ := context.WithTimeout(context.Background(), 2*time.Second)
	client, err := mongo.Connect(ctx, options.Client().ApplyURI(config.MongoUrl))
	if err != nil {
		t.Fatalf("ERROR: %+v", err)
		return
	}
	result := client.Database(config.MongoDatabaseName).Collection(config.MongoIncidentCollectionName).FindOne(ctx, bson.M{"id": expected.Id})
	err = result.Err()
	if err != nil {
		t.Fatalf("ERROR: %+v", err)
		return
	}
	compare := messages.Incident{}
	err = result.Decode(&compare)
	if err != nil {
		t.Fatalf("ERROR: %+v", err)
		return
	}

	if expected.Time.Unix() != compare.Time.Unix() {
		t.Fatal(expected.Time.Unix(), compare.Time.Unix())
	}
	expected.Time = time.Time{}
	compare.Time = time.Time{}
	if !reflect.DeepEqual(expected, compare) {
		t.Fatal(expected, compare)
	}
}

func checkIncidentsInDatabase(t *testing.T, config configuration.Config, expected ...messages.Incident) {
	ctx, _ := context.WithTimeout(context.Background(), 2*time.Second)
	client, err := mongo.Connect(ctx, options.Client().ApplyURI(config.MongoUrl))
	if err != nil {
		t.Fatalf("ERROR: %+v", err)
		return
	}
	option := options.Find().
		SetSort(bson.D{
			{"id", 1},
		})

	incidents := []messages.Incident{}
	cursor, err := client.Database(config.MongoDatabaseName).Collection(config.MongoIncidentCollectionName).Find(ctx, bson.M{}, option)
	if err != nil {
		t.Fatalf("ERROR: %+v", err)
		return
	}
	for cursor.Next(context.Background()) {
		incident := messages.Incident{}
		err = cursor.Decode(&incident)
		if err != nil {
			t.Fatalf("ERROR: %+v", err)
			return
		}
		incident.Time = time.Time{}
		incidents = append(incidents, incident)
	}
	err = cursor.Err()
	if err != nil {
		t.Fatalf("ERROR: %+v", err)
		return
	}
	if !reflect.DeepEqual(expected, incidents) {
		t.Fatal(expected, incidents)
	}
}

func checkOnIncidentsInDatabase(t *testing.T, config configuration.Config, expected ...messages.OnIncident) {
	ctx, _ := context.WithTimeout(context.Background(), 2*time.Second)
	client, err := mongo.Connect(ctx, options.Client().ApplyURI(config.MongoUrl))
	if err != nil {
		t.Fatalf("ERROR: %+v", err)
		return
	}

	result := []messages.OnIncident{}
	cursor, err := client.Database(config.MongoDatabaseName).Collection(config.MongoOnIncidentCollectionName).Find(ctx, bson.M{}, nil)
	if err != nil {
		t.Fatalf("ERROR: %+v", err)
		return
	}
	for cursor.Next(context.Background()) {
		element := messages.OnIncident{}
		err = cursor.Decode(&element)
		if err != nil {
			t.Fatalf("ERROR: %+v", err)
			return
		}
		result = append(result, element)
	}
	err = cursor.Err()
	if err != nil {
		t.Fatalf("ERROR: %+v", err)
		return
	}
	if !reflect.DeepEqual(expected, result) {
		t.Fatal(expected, result)
	}
}
