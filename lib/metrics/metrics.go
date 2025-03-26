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

package metrics

import (
	"context"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"log"
	"net/http"
	"runtime/debug"
)

type Metrics struct {
	IncidentMessages prometheus.Counter
	httphandler      http.Handler
}

func New() *Metrics {
	reg := prometheus.NewRegistry()
	m := &Metrics{
		httphandler: promhttp.HandlerFor(
			reg,
			promhttp.HandlerOpts{
				Registry: reg,
			},
		),
		IncidentMessages: prometheus.NewCounter(prometheus.CounterOpts{
			Name: "incident_worker_incident_messages",
			Help: "count of incident messages received since startup",
		}),
	}

	reg.MustRegister(m.IncidentMessages)

	return m
}

func (this *Metrics) ServeHTTP(writer http.ResponseWriter, request *http.Request) {
	this.httphandler.ServeHTTP(writer, request)
}

func (this *Metrics) Serve(ctx context.Context, port string) *Metrics {
	if port == "" || port == "-" {
		return this
	}
	router := http.NewServeMux()

	router.Handle("/metrics", this)

	server := &http.Server{Addr: ":" + port, Handler: router}
	go func() {
		log.Println("listening on ", server.Addr, "for /metrics")
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			debug.PrintStack()
			log.Fatal("FATAL:", err)
		}
	}()
	go func() {
		<-ctx.Done()
		log.Println("metrics shutdown", server.Shutdown(context.Background()))
	}()
	return this
}

func (this *Metrics) NotifyIncidentMessage() {
	if this != nil && this.IncidentMessages != nil {
		this.IncidentMessages.Inc()
	}
}
