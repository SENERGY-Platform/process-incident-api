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
	"github.com/SENERGY-Platform/process-incident-api/lib/messages"
	"github.com/pkg/errors"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"go.mongodb.org/mongo-driver/x/bsonx"
	"log"
)

func (this *mongoclient) GetIncidents(id string, user string) (incident messages.IncidentMessage, exists bool, err error) {
	ctx, _ := context.WithTimeout(context.Background(), TIMEOUT)
	result := this.collection().FindOne(ctx, bson.M{"id": id, "tenant_id": user})
	err = errors.WithStack(result.Err())
	if err != nil {
		return incident, exists, err
	}
	err = result.Decode(&incident)
	if err == mongo.ErrNoDocuments {
		return incident, false, nil
	}
	return incident, true, errors.WithStack(err)
}

func (this *mongoclient) FindIncidents(externalTaskId string, processDefinitionId string, processInstanceId string, limit int, offset int, sortby string, asc bool, user string) (incidents []messages.IncidentMessage, err error) {
	if this.config.Debug {
		log.Println("DEBUG: FindIncidents()", externalTaskId, processDefinitionId, processInstanceId)
	}
	filter := bson.M{"tenant_id": user}
	if processDefinitionId != "" {
		filter["process_definition_id"] = processDefinitionId
	}
	if processInstanceId != "" {
		filter["process_instance_id"] = processInstanceId
	}
	if externalTaskId != "" {
		filter["external_task_id"] = externalTaskId
	}
	if this.config.Debug {
		log.Println("DEBUG: FindIncidents() filter = ", filter)
	}

	direction := int32(1)
	if !asc {
		direction = int32(-1)
	}

	option := options.Find().
		SetSkip(int64(offset)).
		SetLimit(int64(limit)).
		SetSort(bsonx.Doc{
			{sortby, bsonx.Int32(direction)},
		})

	ctx, _ := context.WithTimeout(context.Background(), TIMEOUT)
	cursor, err := this.collection().Find(ctx, filter, option)
	if err != nil {
		return incidents, errors.WithStack(err)
	}
	for cursor.Next(context.Background()) {
		incident := messages.IncidentMessage{}
		err = cursor.Decode(&incident)
		if err != nil {
			return nil, errors.WithStack(err)
		}
		incidents = append(incidents, incident)
	}
	err = cursor.Err()
	return incidents, errors.WithStack(err)
}
