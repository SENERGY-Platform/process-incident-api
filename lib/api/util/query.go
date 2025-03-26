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

package util

import (
	"errors"
	"net/http"
	"runtime/debug"
	"strconv"
	"strings"
)

const DEFAULT_LIMIT = 100
const DEFAULT_OFFSET = 0

func GetAuthToken(req *http.Request) string {
	return req.Header.Get("Authorization")
}

func ParseLimit(str string) (limit int, err error) {
	if str == "" {
		return DEFAULT_LIMIT, nil
	}
	limit, err = strconv.Atoi(str)
	if err != nil {
		return limit, errors.New("unable to parse limit")
	}
	return
}

func ParseOffset(str string) (limit int, err error) {
	if str == "" {
		return DEFAULT_OFFSET, nil
	}
	limit, err = strconv.Atoi(str)
	if err != nil {
		return limit, errors.New("unable to parse offset")
	}
	return
}

func ParseSort(str string, fields []string) (field string, asc bool, err error) {
	if len(fields) == 0 {
		debug.PrintStack()
		return "", false, errors.New("missing fields")
	}
	if str == "" {
		return fields[0], false, nil
	}
	parts := strings.SplitN(str, ".", 2)
	if len(parts) >= 1 {
		field = parts[0]
	} else {
		field = fields[0]
	}

	fieldAllowed := false
	for _, allowedField := range fields {
		if field == allowedField {
			fieldAllowed = true
			break
		}
	}
	if !fieldAllowed {
		return "", false, errors.New("not sortable by given field")
	}

	direction := ""
	if len(parts) >= 2 {
		direction = parts[1]
	} else {
		direction = "asc"
	}
	if direction != "asc" && direction != "desc" {
		return "", false, errors.New("unknown sort direction")
	}
	return field, direction == "asc", nil
}
