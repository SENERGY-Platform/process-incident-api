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
	"errors"
	"github.com/SENERGY-Platform/process-incident-api/lib/messages"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"log"
)

func (this *mongoclient) collection() *mongo.Collection {
	return this.client.Database(this.config.MongoDatabaseName).Collection(this.config.MongoIncidentCollectionName)
}

func (this *mongoclient) GetIncidents(id string, user string) (incident messages.IncidentMessage, exists bool, err error) {
	result := this.collection().FindOne(this.getTimeoutContext(), bson.M{"id": id, "tenant_id": user})
	if errors.Is(err, mongo.ErrNoDocuments) {
		return incident, false, nil
	}
	err = result.Err()
	if err != nil {
		return incident, exists, err
	}
	err = result.Decode(&incident)
	if errors.Is(err, mongo.ErrNoDocuments) {
		return incident, false, nil
	}
	return incident, true, err
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
		SetSort(bson.D{{sortby, direction}})

	cursor, err := this.collection().Find(this.getTimeoutContext(), filter, option)
	if err != nil {
		return incidents, err
	}
	for cursor.Next(context.Background()) {
		incident := messages.IncidentMessage{}
		err = cursor.Decode(&incident)
		if err != nil {
			return nil, err
		}
		incidents = append(incidents, incident)
	}
	err = cursor.Err()
	return incidents, err
}

func (this *mongoclient) SaveIncident(incident messages.Incident) error {
	_, err := this.collection().ReplaceOne(this.getTimeoutContext(), bson.M{"id": incident.Id}, incident, options.Replace().SetUpsert(true))
	return err
}

func (this *mongoclient) DeleteByDefinitionId(id string) error {
	err := this.DeleteIncidentByDefinitionId(id)
	if err != nil {
		return err
	}
	return this.DeleteOnIncidentByDefinitionId(id)
}

func (this *mongoclient) DeleteIncidentByInstanceId(id string) error {
	_, err := this.collection().DeleteMany(this.getTimeoutContext(), bson.M{"process_instance_id": id})
	return err
}

func (this *mongoclient) DeleteIncidentByDefinitionId(id string) error {
	_, err := this.collection().DeleteMany(this.getTimeoutContext(), bson.M{"process_definition_id": id})
	return err
}
