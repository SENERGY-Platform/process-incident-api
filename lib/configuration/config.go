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

package configuration

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"os"
	"reflect"
	"regexp"
	"runtime/debug"
	"strconv"
	"strings"
	"time"

	struct_logger "github.com/SENERGY-Platform/go-service-base/struct-logger"
)

type Config struct {
	MetricsPort                    string `json:"metrics_port"`
	ShardsDb                       string `json:"shards_db" config:"secret"`
	MongoUrl                       string `json:"mongo_url"`  // full connection string with scheme, credentials belong in MongoUser/MongoPassword
	MongoUser                      string `json:"mongo_user"` // empty: no authentication
	MongoPassword                  string `json:"mongo_password" config:"secret"`
	MongoAuthSource                string `json:"mongo_auth_source"`
	MongoDatabase                  string `json:"mongo_database"`
	MongoIncidentCollectionName    string `json:"mongo_incident_collection_name"`
	MongoOnIncidentCollectionName  string `json:"mongo_on_incident_collection_name"`
	ApiPort                        string `json:"api_port"`
	ApiLog                         bool   `json:"api_log"`
	Debug                          bool   `json:"debug"`
	NotificationUrl                string `json:"notification_url"`
	DeveloperNotificationUrl       string `json:"developer_notification_url"`
	CamundaIncidentRequestInterval string `json:"camunda_incident_request_interval"`
	NotificationThrottle           string `json:"notification_throttle"` // Go duration; "0" or empty: notify for every incident

	LogLevel string       `json:"log_level"`
	logger   *slog.Logger `json:"-"`
}

// loads config from json in location and used environment variables (e.g ZookeeperUrl --> ZOOKEEPER_URL)
func LoadConfig(location string) (config Config, err error) {
	file, err := os.Open(location)
	if err != nil {
		return config, err
	}
	err = json.NewDecoder(file).Decode(&config)
	if err != nil {
		return config, err
	}
	handleEnvironmentVars(&config)
	return config, nil
}

// NotificationThrottleWindow parses NotificationThrottle; an empty value or a zero duration switches throttling off.
func (this Config) NotificationThrottleWindow() (time.Duration, error) {
	if this.NotificationThrottle == "" {
		return 0, nil
	}
	window, err := time.ParseDuration(this.NotificationThrottle)
	if err != nil {
		return 0, fmt.Errorf("invalid notification_throttle %q: %w", this.NotificationThrottle, err)
	}
	if window < 0 {
		return 0, fmt.Errorf("invalid notification_throttle %q: must not be negative", this.NotificationThrottle)
	}
	return window, nil
}

func isSecret(field reflect.StructField) bool {
	return strings.Contains(field.Tag.Get("config"), "secret")
}

// plainConfig has none of Config's methods, so formatting it does not recurse.
type plainConfig Config

// masked returns a copy in which every non-empty field tagged config:"secret" is replaced.
func (this Config) masked() plainConfig {
	v := reflect.ValueOf(&this).Elem()
	for i := 0; i < v.NumField(); i++ {
		if isSecret(v.Type().Field(i)) && v.Field(i).Kind() == reflect.String && v.Field(i).String() != "" {
			v.Field(i).SetString("***")
		}
	}
	return plainConfig(this)
}

func (this Config) MarshalJSON() ([]byte, error) {
	return json.Marshal(this.masked())
}

func (this Config) String() string {
	return fmt.Sprintf("%+v", this.masked())
}

func (this Config) GoString() string {
	return fmt.Sprintf("%#v", this.masked())
}

var camel = regexp.MustCompile("(^[^A-Z]*|[A-Z]*)([A-Z][^A-Z]+|$)")

func fieldNameToEnvName(s string) string {
	var a []string
	for _, sub := range camel.FindAllStringSubmatch(s, -1) {
		if sub[1] != "" {
			a = append(a, sub[1])
		}
		if sub[2] != "" {
			a = append(a, sub[2])
		}
	}
	return strings.ToUpper(strings.Join(a, "_"))
}

// preparations for docker
func handleEnvironmentVars(config *Config) {
	configValue := reflect.Indirect(reflect.ValueOf(config))
	configType := configValue.Type()
	for index := 0; index < configType.NumField(); index++ {
		fieldName := configType.Field(index).Name
		envName := fieldNameToEnvName(fieldName)
		envValue := os.Getenv(envName)
		if envValue != "" {
			if isSecret(configType.Field(index)) {
				fmt.Println("use environment variable: ", envName)
			} else {
				fmt.Println("use environment variable: ", envName, " = ", envValue)
			}
			if configValue.FieldByName(fieldName).Kind() == reflect.Int64 {
				i, _ := strconv.ParseInt(envValue, 10, 64)
				configValue.FieldByName(fieldName).SetInt(i)
			}
			if configValue.FieldByName(fieldName).Kind() == reflect.String {
				configValue.FieldByName(fieldName).SetString(envValue)
			}
			if configValue.FieldByName(fieldName).Kind() == reflect.Bool {
				b, _ := strconv.ParseBool(envValue)
				configValue.FieldByName(fieldName).SetBool(b)
			}
			if configValue.FieldByName(fieldName).Kind() == reflect.Float64 {
				f, _ := strconv.ParseFloat(envValue, 64)
				configValue.FieldByName(fieldName).SetFloat(f)
			}
			if configValue.FieldByName(fieldName).Kind() == reflect.Slice {
				val := []string{}
				for _, element := range strings.Split(envValue, ",") {
					val = append(val, strings.TrimSpace(element))
				}
				configValue.FieldByName(fieldName).Set(reflect.ValueOf(val))
			}
			if configValue.FieldByName(fieldName).Kind() == reflect.Map {
				value := map[string]string{}
				for _, element := range strings.Split(envValue, ",") {
					keyVal := strings.Split(element, ":")
					key := strings.TrimSpace(keyVal[0])
					val := strings.TrimSpace(keyVal[1])
					value[key] = val
				}
				configValue.FieldByName(fieldName).Set(reflect.ValueOf(value))
			}
		}
	}
}

func (this *Config) GetLogger() *slog.Logger {
	if this.logger == nil {
		if this.Debug {
			this.LogLevel = "debug"
		}
		info, ok := debug.ReadBuildInfo()
		project := ""
		org := ""
		if ok {
			if parts := strings.Split(info.Main.Path, "/"); len(parts) > 2 {
				project = strings.Join(parts[2:], "/")
				org = strings.Join(parts[:2], "/")
			}
		}
		this.logger = struct_logger.New(
			struct_logger.Config{
				Handler:    struct_logger.JsonHandlerSelector,
				Level:      this.LogLevel,
				TimeFormat: time.RFC3339Nano,
				TimeUtc:    true,
				AddMeta:    true,
			},
			os.Stdout,
			org,
			project,
		)
		slog.SetDefault(this.logger)
		slog.SetLogLoggerLevel(slog.LevelInfo)
	}
	return this.logger
}
