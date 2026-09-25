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
	"errors"
	"testing"

	"github.com/SENERGY-Platform/process-incident-api/lib/configuration"
)

func TestValidateConfig(t *testing.T) {
	cases := []struct {
		name    string
		config  configuration.Config
		wantErr error
	}{
		{
			name:    "empty database",
			config:  configuration.Config{MongoDatabase: ""},
			wantErr: errEmptyDatabase,
		},
		{
			name:    "user without password",
			config:  configuration.Config{MongoDatabase: "incidents", MongoUser: "svc"},
			wantErr: errMissingPassword,
		},
		{
			name:    "no user, no password",
			config:  configuration.Config{MongoDatabase: "incidents"},
			wantErr: nil,
		},
		{
			name:    "user with password",
			config:  configuration.Config{MongoDatabase: "incidents", MongoUser: "svc", MongoPassword: "pw"},
			wantErr: nil,
		},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			err := validateConfig(c.config)
			if !errors.Is(err, c.wantErr) {
				t.Errorf("validateConfig() = %v, want %v", err, c.wantErr)
			}
		})
	}
}

func TestClientOptionsAppliesCredentialsOnlyWhenUserSet(t *testing.T) {
	opts := clientOptions(configuration.Config{MongoUrl: "mongodb://localhost:27017"})
	if opts.Auth != nil {
		t.Errorf("expected no credentials without a user, got %+v", opts.Auth)
	}

	opts = clientOptions(configuration.Config{MongoUrl: "mongodb://localhost:27017", MongoUser: "svc", MongoPassword: "pw", MongoAuthSource: "admin"})
	if opts.Auth == nil {
		t.Fatal("expected credentials to be set")
	}
	if opts.Auth.Username != "svc" || opts.Auth.Password != "pw" || opts.Auth.AuthSource != "admin" {
		t.Errorf("unexpected credentials: %+v", opts.Auth)
	}
}
