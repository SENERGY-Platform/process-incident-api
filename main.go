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

package main

import (
	"context"
	"flag"
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/SENERGY-Platform/process-incident-api/lib"
	"github.com/SENERGY-Platform/process-incident-api/lib/configuration"
)

func main() {

	configLocation := flag.String("config", "config.json", "configuration file")
	flag.Parse()

	config, err := configuration.LoadConfig(*configLocation)
	if err != nil {
		log.Fatalf("FATAL: %+v", err)
	}

	err = lib.Start(context.Background(), config)
	if err != nil {
		config.GetLogger().Error("unable to start lib", "error", err)
		log.Fatalf("FATAL: %+v", err)
	}

	shutdown := make(chan os.Signal, 1)
	signal.Notify(shutdown, syscall.SIGINT, syscall.SIGTERM, syscall.SIGKILL)
	sig := <-shutdown
	config.GetLogger().Info("received shutdown signal", "signal", sig)
}
