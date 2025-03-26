/*
 * Copyright 2024 InfAI (CC SES)
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

package client

import (
	"encoding/json"
	"errors"
	"io"
	"log"
	"net/http"
	"net/url"
	"sync"
	"time"
)

// InternalAdminToken is expired and invalid. but because this service does not validate the received tokens,
// it may be used by trusted internal services which are within the same network (kubernetes cluster).
// requests with this token may not be routed over an ingres with token validation
const InternalAdminToken = `Bearer eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJleHAiOjEwMDAwMDAwMDAsImlhdCI6MTAwMDAwMDAwMCwiYXV0aF90aW1lIjoxMDAwMDAwMDAwLCJpc3MiOiJpbnRlcm5hbCIsImF1ZCI6W10sInN1YiI6ImRkNjllYTBkLWY1NTMtNDMzNi04MGYzLTdmNDU2N2Y4NWM3YiIsInR5cCI6IkJlYXJlciIsImF6cCI6ImZyb250ZW5kIiwicmVhbG1fYWNjZXNzIjp7InJvbGVzIjpbImFkbWluIiwiZGV2ZWxvcGVyIiwidXNlciJdfSwicmVzb3VyY2VfYWNjZXNzIjp7Im1hc3Rlci1yZWFsbSI6eyJyb2xlcyI6W119LCJCYWNrZW5kLXJlYWxtIjp7InJvbGVzIjpbXX0sImFjY291bnQiOnsicm9sZXMiOltdfX0sInJvbGVzIjpbImFkbWluIiwiZGV2ZWxvcGVyIiwidXNlciJdLCJuYW1lIjoiU2VwbCBBZG1pbiIsInByZWZlcnJlZF91c2VybmFtZSI6InNlcGwiLCJnaXZlbl9uYW1lIjoiU2VwbCIsImxvY2FsZSI6ImVuIiwiZmFtaWx5X25hbWUiOiJBZG1pbiIsImVtYWlsIjoic2VwbEBzZXBsLmRlIn0.HZyG6n-BfpnaPAmcDoSEh0SadxUx-w4sEt2RVlQ9e5I`

func NewTokenProvider(authEndpoint string, authClientId string, authClientSecret string) func() (string, error) {
	openid := OpenidToken{}
	mux := sync.Mutex{}
	return func() (string, error) {
		var err error
		defer func() {
			if err != nil {
				openid = OpenidToken{}
			}
		}()
		mux.Lock()
		defer mux.Unlock()
		duration := time.Now().Sub(openid.RequestTime).Seconds() + 10

		if openid.AccessToken != "" && openid.ExpiresIn > duration {
			return "Bearer " + openid.AccessToken, nil
		}

		if openid.RefreshToken != "" && openid.RefreshExpiresIn > duration {
			log.Println("refresh token", openid.RefreshExpiresIn, duration)
			openid, err = refreshOpenidToken(authEndpoint, authClientId, authClientSecret, openid)
			if err != nil {
				log.Println("WARNING: unable to use refresh token", err)
			} else {
				return "Bearer " + openid.AccessToken, nil
			}
		}
		openid, err = getOpenidToken(authEndpoint, authClientId, authClientSecret)
		if err != nil {
			return "", err
		}
		return "Bearer " + openid.AccessToken, nil
	}
}

func NewUserTokenProvider(authEndpoint string, authClientId string, authClientSecret string, userName string, pw string) func() (string, error) {
	openid := OpenidToken{}
	mux := sync.Mutex{}
	return func() (string, error) {
		var err error
		defer func() {
			if err != nil {
				openid = OpenidToken{}
			}
		}()
		mux.Lock()
		defer mux.Unlock()
		duration := time.Now().Sub(openid.RequestTime).Seconds() + 10

		if openid.AccessToken != "" && openid.ExpiresIn > duration {
			return "Bearer " + openid.AccessToken, nil
		}

		if openid.RefreshToken != "" && openid.RefreshExpiresIn > duration {
			log.Println("refresh token", openid.RefreshExpiresIn, duration)
			openid, err = refreshOpenidToken(authEndpoint, authClientId, authClientSecret, openid)
			if err != nil {
				log.Println("WARNING: unable to use refresh token", err)
			} else {
				return "Bearer " + openid.AccessToken, nil
			}
		}
		openid, err = getOpenidPasswordToken(authEndpoint, authClientId, authClientSecret, userName, pw)
		if err != nil {
			return "", err
		}
		return "Bearer " + openid.AccessToken, nil
	}
}

type OpenidToken struct {
	AccessToken      string    `json:"access_token"`
	ExpiresIn        float64   `json:"expires_in"`
	RefreshExpiresIn float64   `json:"refresh_expires_in"`
	RefreshToken     string    `json:"refresh_token"`
	TokenType        string    `json:"token_type"`
	RequestTime      time.Time `json:"-"`
}

func getOpenidToken(authEndpoint string, authClientId string, authClientSecret string) (openid OpenidToken, err error) {
	requesttime := time.Now()
	client := http.Client{
		Timeout: 5 * time.Second,
	}
	resp, err := client.PostForm(authEndpoint+"/auth/realms/master/protocol/openid-connect/token", url.Values{
		"client_id":     {authClientId},
		"client_secret": {authClientSecret},
		"grant_type":    {"client_credentials"},
	})
	if err != nil {
		return openid, err
	}
	if resp.StatusCode >= 300 {
		b, _ := io.ReadAll(resp.Body)
		err = errors.New(resp.Status + ": " + string(b))
		return
	}
	err = json.NewDecoder(resp.Body).Decode(&openid)
	openid.RequestTime = requesttime
	return
}

func refreshOpenidToken(authEndpoint string, authClientId string, authClientSecret string, oldOpenid OpenidToken) (openid OpenidToken, err error) {
	requesttime := time.Now()
	client := http.Client{
		Timeout: 5 * time.Second,
	}
	resp, err := client.PostForm(authEndpoint+"/auth/realms/master/protocol/openid-connect/token", url.Values{
		"client_id":     {authClientId},
		"client_secret": {authClientSecret},
		"refresh_token": {oldOpenid.RefreshToken},
		"grant_type":    {"refresh_token"},
	})

	if err != nil {
		return openid, err
	}
	if resp.StatusCode >= 300 {
		b, _ := io.ReadAll(resp.Body)
		err = errors.New(resp.Status + ": " + string(b))
		return
	}
	err = json.NewDecoder(resp.Body).Decode(&openid)
	openid.RequestTime = requesttime
	return
}

func getOpenidPasswordToken(authEndpoint string, authClientId string, authClientSecret string, username, password string) (token OpenidToken, err error) {
	requesttime := time.Now()
	client := http.Client{
		Timeout: 5 * time.Second,
	}
	resp, err := client.PostForm(authEndpoint+"/auth/realms/master/protocol/openid-connect/token", url.Values{
		"client_id":     {authClientId},
		"client_secret": {authClientSecret},
		"username":      {username},
		"password":      {password},
		"grant_type":    {"password"},
	})

	if err != nil {
		return token, err
	}
	if resp.StatusCode >= 300 {
		b, _ := io.ReadAll(resp.Body)
		err = errors.New(resp.Status + ": " + string(b))
		return
	}
	err = json.NewDecoder(resp.Body).Decode(&token)
	token.RequestTime = requesttime
	return
}
