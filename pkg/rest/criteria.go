/*
 * Copyright (c) 2020 Baidu, Inc.
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

package rest

import (
	"net/http"
	"net/url"
)

type QueryCriteria interface {
	AddCondition(key, value string) QueryCriteria
	Value() url.Values
	ReadFromRequest(r *http.Request) QueryCriteria
}

type criteria struct {
	url.Values
}

// ensure type
var _ QueryCriteria = &criteria{}

func NewQueryCriteria() QueryCriteria {
	return &criteria{url.Values{}}
}

func (c *criteria) AddCondition(key, value string) QueryCriteria {
	c.Add(key, value)
	return c
}

func (c *criteria) Value() url.Values {
	return c.Values
}

func (c *criteria) ReadFromRequest(r *http.Request) QueryCriteria {
	query := r.URL.Query()
	for key, values := range query {
		for _, val := range values {
			c.Values.Add(key, val)
		}
	}
	return c
}
