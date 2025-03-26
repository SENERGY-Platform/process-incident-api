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

package server

import (
	"context"
	"github.com/SENERGY-Platform/process-incident-api/lib/camunda/cache"
	"github.com/SENERGY-Platform/process-incident-api/lib/camunda/shards"
	"github.com/SENERGY-Platform/process-incident-api/lib/configuration"
	"github.com/SENERGY-Platform/process-incident-api/tests/server/docker"
	"log"
	"net"
	"runtime/debug"
	"strconv"
	"sync"
	"time"
)

func New(ctx context.Context, wg *sync.WaitGroup, init configuration.Config) (config configuration.Config, err error) {
	config = init
	config.Debug = true

	whPort, err := getFreePort()
	if err != nil {
		log.Println("unable to find free port", err)
		return config, err
	}
	config.ApiPort = strconv.Itoa(whPort)

	_, ip, err := docker.Mongo(ctx, wg)
	if err != nil {
		log.Println("ERROR:", err)
		debug.PrintStack()
		return config, err
	}
	config.MongoUrl = "mongodb://" + ip + ":27017"

	_, pgIp, _, err := docker.PostgresWithNetwork(ctx, wg, "camunda")
	if err != nil {
		log.Println("ERROR:", err)
		debug.PrintStack()
		return config, err
	}

	camundaUrl, err := docker.Camunda(ctx, wg, pgIp, "5432")
	if err != nil {
		log.Println("ERROR:", err)
		debug.PrintStack()
		return config, err
	}

	shardsDb, err := docker.Postgres(ctx, wg, "shards")
	if err != nil {
		log.Println("ERROR:", err)
		debug.PrintStack()
		return config, err
	}
	config.ShardsDb = shardsDb

	s, err := shards.New(shardsDb, cache.None)
	if err != nil {
		log.Println("ERROR:", err)
		debug.PrintStack()
		return config, err
	}

	err = s.EnsureShard(camundaUrl)
	if err != nil {
		log.Println("ERROR:", err)
		debug.PrintStack()
		return config, err
	}

	_, err = s.EnsureShardForUser("")
	if err != nil {
		log.Println("ERROR:", err)
		debug.PrintStack()
		return config, err
	}

	time.Sleep(5 * time.Second)

	return config, nil
}

func getFreePort() (int, error) {
	addr, err := net.ResolveTCPAddr("tcp", "localhost:0")
	if err != nil {
		return 0, err
	}

	listener, err := net.ListenTCP("tcp", addr)
	if err != nil {
		return 0, err
	}
	defer listener.Close()
	return listener.Addr().(*net.TCPAddr).Port, nil
}
