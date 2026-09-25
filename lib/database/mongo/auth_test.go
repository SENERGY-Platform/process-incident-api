/*
 * Copyright 2026 InfAI (CC SES)
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
	"crypto/rand"
	"encoding/hex"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/SENERGY-Platform/process-incident-api/lib/configuration"
	"github.com/SENERGY-Platform/process-incident-api/lib/messages"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

// TestStartAuthenticates needs a throwaway server with access control; MONGO_AUTH_TEST_USER and
// MONGO_AUTH_TEST_PASSWORD are root credentials, used to create and remove the test users.
func TestStartAuthenticates(t *testing.T) {
	url, rootUser, rootPassword := os.Getenv("MONGO_AUTH_TEST_URL"), os.Getenv("MONGO_AUTH_TEST_USER"), os.Getenv("MONGO_AUTH_TEST_PASSWORD")
	if testing.Short() || url == "" || rootUser == "" || rootPassword == "" {
		t.Skip("needs MONGO_AUTH_TEST_URL, MONGO_AUTH_TEST_USER and MONGO_AUTH_TEST_PASSWORD, not in -short")
	}
	ctx := context.Background()
	root, err := mongo.Connect(ctx, options.Client().ApplyURI(url).SetAuth(options.Credential{Username: rootUser, Password: rootPassword, AuthSource: "admin"}))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = root.Disconnect(ctx) })

	suffix := randomHex(t)
	testDB, otherDB := "process_incident_auth_test_"+suffix, "process_incident_auth_other_"+suffix
	svcUser, svcPassword := "incident-api-test-"+suffix, randomHex(t)
	otherUser, otherPassword := "incident-api-other-"+suffix, randomHex(t)
	createUser(t, root, svcUser, svcPassword, testDB)
	createUser(t, root, otherUser, otherPassword, otherDB)

	config := func(user, password string) configuration.Config {
		return configuration.Config{
			MongoUrl:                      url,
			MongoUser:                     user,
			MongoPassword:                 password,
			MongoAuthSource:               "admin",
			MongoDatabase:                 testDB,
			MongoIncidentCollectionName:   "incidents",
			MongoOnIncidentCollectionName: "on_incident",
		}
	}

	cases := []struct {
		name, user, password string
		wantErr              bool
	}{
		{"readWrite user starts fine", svcUser, svcPassword, false},
		{"wrong password fails", svcUser, svcPassword + "-wrong", true},
		{"user without rights on the database fails", otherUser, otherPassword, true},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			svcCtx, cancel := context.WithCancel(ctx)
			defer cancel()
			db, err := New(svcCtx, config(c.user, c.password))
			if c.wantErr {
				if err == nil {
					t.Fatal("expected an error")
				}
				if !strings.HasPrefix(err.Error(), "mongo startup check failed: ") {
					t.Errorf("unexpected error: %v", err)
				}
				for _, pw := range []string{svcPassword, otherPassword, rootPassword} {
					if strings.Contains(err.Error(), pw) {
						t.Error("error text contains a password")
					}
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if err = db.SaveOnIncident(messages.OnIncident{ProcessDefinitionId: "test", Restart: true}); err != nil {
				t.Errorf("query as the service user: %v", err)
			}
		})
	}
}

// createUser registers the removal first, so a creation that fails halfway leaves nothing behind.
func createUser(t *testing.T, root *mongo.Client, user, password, db string) {
	t.Helper()
	admin := root.Database("admin")
	t.Cleanup(func() {
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		_ = admin.RunCommand(ctx, bson.D{{Key: "dropUser", Value: user}}).Err()
		_ = root.Database(db).Drop(ctx)
	})
	cmd := bson.D{
		{Key: "createUser", Value: user},
		{Key: "pwd", Value: password},
		{Key: "roles", Value: bson.A{bson.D{{Key: "role", Value: "readWrite"}, {Key: "db", Value: db}}}},
	}
	if err := admin.RunCommand(context.Background(), cmd).Err(); err != nil {
		t.Fatalf("create user: %v", err)
	}
}

func randomHex(t *testing.T) string {
	t.Helper()
	b := make([]byte, 8)
	if _, err := rand.Read(b); err != nil {
		t.Fatal(err)
	}
	return hex.EncodeToString(b)
}
