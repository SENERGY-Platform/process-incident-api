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

package notification

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"log"
	"net/http"
	"time"
)

func Send(notificationUrl string, message Message) error {
	if notificationUrl == "" {
		return nil
	}
	log.Println("send notification", notificationUrl, message)
	b := new(bytes.Buffer)
	err := json.NewEncoder(b).Encode(message)
	if err != nil {
		return err
	}
	req, err := http.NewRequest("POST", notificationUrl+"/notifications?ignore_duplicates_within_seconds=3600", b)
	if err != nil {
		log.Println("ERROR: unable to send notification", err)
		return err
	}
	ctx, _ := context.WithTimeout(context.Background(), 5*time.Second)
	req.WithContext(ctx)
	req.Header.Set("Content-Type", "application/json")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		log.Println("ERROR: unable to send notification", err)
		return err
	}
	if resp.StatusCode >= 300 {
		respMsg, _ := io.ReadAll(resp.Body)
		log.Println("ERROR: unexpected response status from notifier", resp.StatusCode, string(respMsg))
		return errors.New("unexpected response status from notifier " + resp.Status)
	}
	return nil
}

type Message struct {
	UserId  string `json:"userId" bson:"userId"`
	Title   string `json:"title" bson:"title"`
	Message string `json:"message" bson:"message"`
	Topic   string `json:"topic" bson:"topic"`
}

const Topic = "incident"
