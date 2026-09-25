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
	"fmt"
	"time"

	"github.com/SENERGY-Platform/process-incident-api/lib/configuration"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

const TIMEOUT = 2 * time.Second

// startupCheckTimeout bounds the authenticated check that replaces the unauthenticated ping.
const startupCheckTimeout = 10 * time.Second

var (
	errEmptyDatabase   = errors.New("mongo database name must not be empty")
	errMissingPassword = errors.New("mongo password must not be empty when a mongo user is set")
)

type mongoclient struct {
	config configuration.Config
	client *mongo.Client
}

func New(ctx context.Context, config configuration.Config) (result *mongoclient, err error) {
	if err = validateConfig(config); err != nil {
		return nil, err
	}
	result = &mongoclient{config: config}
	ctx, cancel := context.WithCancel(ctx)
	client, err := mongo.Connect(ctx, clientOptions(config))
	if err != nil {
		cancel()
		return nil, err
	}
	result.client = client
	go func() {
		// captures client, not result: a failure path below nils the named return result
		// while this goroutine is still live, which would otherwise dereference a nil result.
		<-ctx.Done()
		config.GetLogger().Info("disconnect mongodb")
		disconnectCtx, _ := context.WithTimeout(context.Background(), TIMEOUT)
		client.Disconnect(disconnectCtx)
	}()
	// listCollections requires authentication, unlike Ping, so wrong or missing credentials fail here.
	checkCtx, checkCancel := context.WithTimeout(ctx, startupCheckTimeout)
	_, err = client.Database(config.MongoDatabase).ListCollectionNames(checkCtx, bson.D{},
		options.ListCollections().SetNameOnly(true).SetAuthorizedCollections(true))
	checkCancel()
	if err != nil {
		cancel()
		return nil, fmt.Errorf("mongo startup check failed: %w", err)
	}
	err = result.init()
	if err != nil {
		cancel()
		return nil, err
	}
	return result, nil
}

func validateConfig(config configuration.Config) error {
	if config.MongoDatabase == "" {
		return errEmptyDatabase
	}
	if config.MongoUser != "" && config.MongoPassword == "" {
		return errMissingPassword
	}
	return nil
}

// clientOptions applies the credentials after the URI so they replace any given in MongoUrl.
func clientOptions(config configuration.Config) *options.ClientOptions {
	opts := options.Client().ApplyURI(config.MongoUrl)
	if config.MongoUser != "" {
		opts.SetAuth(options.Credential{
			Username:   config.MongoUser,
			Password:   config.MongoPassword,
			AuthSource: config.MongoAuthSource,
		})
	}
	return opts
}

func (this *mongoclient) getTimeoutContext() context.Context {
	ctx, _ := context.WithTimeout(context.Background(), TIMEOUT)
	return ctx
}
