package tests

import (
	"context"
	"github.com/SENERGY-Platform/process-incident-api/lib"
	"github.com/SENERGY-Platform/process-incident-api/lib/api"
	"github.com/SENERGY-Platform/process-incident-api/lib/camunda"
	"github.com/SENERGY-Platform/process-incident-api/lib/client"
	"github.com/SENERGY-Platform/process-incident-api/lib/configuration"
	"github.com/SENERGY-Platform/process-incident-api/lib/controller"
	"github.com/SENERGY-Platform/process-incident-api/lib/database"
	"github.com/SENERGY-Platform/process-incident-api/lib/messages"
	"github.com/SENERGY-Platform/process-incident-api/lib/metrics"
	"github.com/SENERGY-Platform/process-incident-api/tests/resources"
	"github.com/SENERGY-Platform/process-incident-api/tests/server"
	"github.com/SENERGY-Platform/process-incident-api/tests/server/docker"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"io"
	"log"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
	"time"
)

func TestScriptIncident(t *testing.T) {
	wg := &sync.WaitGroup{}
	defer wg.Wait()
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	defaultConfig, err := configuration.LoadConfig("../config.json")
	if err != nil {
		t.Error(err)
		return
	}
	defaultConfig.Debug = true

	defaultConfig.MetricsPort, err = docker.GetFreePortStr()
	if err != nil {
		t.Error(err)
		return
	}

	log.Println("start docker")
	config, err := server.New(ctx, wg, defaultConfig)
	if err != nil {
		t.Error(err)
		return
	}

	mux := sync.Mutex{}
	notificationCount := 0

	log.Println("start notify mock")
	notificationTestServer := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		msg, _ := io.ReadAll(request.Body)
		t.Log("notification:", request.URL.String(), string(msg))
		mux.Lock()
		defer mux.Unlock()
		notificationCount = notificationCount + 1
	}))
	config.NotificationUrl = notificationTestServer.URL

	log.Println("start lib")
	err = lib.StartWith(ctx, config, api.Factory, database.Factory, camunda.Factory)
	if err != nil {
		t.Error(err)
		return
	}

	processId := ""

	t.Run("deploy process", func(t *testing.T) {
		processId, err = deployProcessWithInfo(config, "test", resources.ScriptErrBpmn, resources.SvgExample, "testuser")
		if err != nil {
			t.Error(err)
			return
		}
	})

	t.Run("set incident handler", func(t *testing.T) {
		camundaInstance, err := camunda.Factory.Get(ctx, config)
		if err != nil {
			t.Error(err)
			return
		}
		databaseInstance, err := database.Factory.Get(ctx, config)
		if err != nil {
			t.Error(err)
			return
		}
		ctrl, err := controller.New(ctx, config, databaseInstance, camundaInstance, metrics.New())
		if err != nil {
			t.Error(err)
			return
		}

		err, _ = ctrl.SetOnIncidentHandler(client.InternalAdminToken, messages.OnIncident{
			ProcessDefinitionId: processId,
			Restart:             true,
			Notify:              true,
		})
		if err != nil {
			t.Error(err)
			return
		}
	})

	t.Run("start process", func(t *testing.T) {
		c, err := camunda.Factory.Get(ctx, config)
		if err != nil {
			t.Error(err)
			return
		}
		err = c.StartProcess(processId, "testuser")
		if err != nil {
			t.Error(err)
			return
		}
	})

	time.Sleep(1 * time.Minute)

	t.Run("check database", func(t *testing.T) {
		ctx, _ := context.WithTimeout(context.Background(), 2*time.Second)
		client, err := mongo.Connect(ctx, options.Client().ApplyURI(config.MongoUrl))
		if err != nil {
			t.Errorf("ERROR: %+v", err)
			return
		}
		c, err := client.Database(config.MongoDatabaseName).Collection(config.MongoIncidentCollectionName).Find(ctx, bson.M{})
		if err != nil {
			t.Errorf("ERROR: %+v", err)
			return
		}
		defer c.Close(ctx)
		counter := 0

		duplicateInstance := map[string]bool{}

		for c.Next(ctx) {
			incident := messages.Incident{}
			err = c.Decode(&incident)
			if err != nil {
				t.Error(err)
				return
			}
			if incident.TenantId != "testuser" {
				t.Errorf("%#v", incident)
				return
			}
			counter = counter + 1
			if duplicateInstance[incident.ProcessInstanceId] {
				t.Error("duplicate process instance found")
			}
			duplicateInstance[incident.ProcessInstanceId] = true
		}
		t.Log("log: incident count =", counter)
		if counter < 2 {
			t.Error("expected at least two incidents")
		}
	})

	t.Run("check notifications", func(t *testing.T) {
		mux.Lock()
		defer mux.Unlock()
		t.Log("log: notificationCount =", notificationCount)
		if notificationCount < 2 {
			t.Error("expected at least two incidents")
		}
	})
}
