/*
 * Copyright (c) 2026 InfAI (CC SES)
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
	"reflect"
	"testing"
	"time"

	"github.com/SENERGY-Platform/process-incident-api/lib/messages"
	"go.mongodb.org/mongo-driver/bson"
)

// The query itself needs a database; this checks, without one, that it matches the stored field names and sorts newest first.
func TestLastNotifiedQuery(t *testing.T) {
	filter, option := lastNotifiedQuery("definition", "user")
	expectedFilter := bson.M{"process_definition_id": "definition", "tenant_id": "user", "notified": true}
	if !reflect.DeepEqual(filter, expectedFilter) {
		t.Errorf("filter %v, expected %v", filter, expectedFilter)
	}
	if !reflect.DeepEqual(option.Sort, bson.D{{Key: "time", Value: -1}}) {
		t.Errorf("sort %v, expected time descending", option.Sort)
	}
}

func TestNotifiedIsStoredUnderTheQueriedName(t *testing.T) {
	stored, err := bson.Marshal(messages.Incident{Id: "id", Time: time.Now(), Notified: true})
	if err != nil {
		t.Fatal(err)
	}
	if v, err := bson.Raw(stored).LookupErr("notified"); err != nil || !v.Boolean() {
		t.Errorf("notified not stored as true: %v, %v", v, err)
	}
	unmarked, err := bson.Marshal(messages.Incident{Id: "id"})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := bson.Raw(unmarked).LookupErr("notified"); err == nil {
		t.Error("an incident without notification must not store the field")
	}
}
