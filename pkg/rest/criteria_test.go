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
	"net/url"
	"reflect"
	"testing"
)

type testCriteria struct {
	QueryCriteria
}

func newTestCriteria() *testCriteria {
	return &testCriteria{NewQueryCriteria()}
}

func TestBaseCriteria(t *testing.T) {
	c := NewQueryCriteria()
	c.AddCondition("con1", "val1")
	c.AddCondition("con1", "val2")
	c.AddCondition("con2", "val3")

	value := c.Value()
	if !reflect.DeepEqual(value["con1"], []string{"val1", "val2"}) {
		t.Fatalf("value[con1] shoudl be [val1, val2] but got %v", value["con1"])
	}

	if !reflect.DeepEqual(value["con2"], []string{"val3"}) {
		t.Fatalf("value[con2] shoudl be [val3] but got %v", value["con2"])
	}
}

func TestEmbeddedCriteria(t *testing.T) {
	c := newTestCriteria()
	c.AddCondition("con1", "val1")

	value := c.Value()
	if !reflect.DeepEqual(value["con1"], []string{"val1"}) {
		t.Fatalf("value[con1] shoudl be [val1] but got %v", value["con1"])
	}
}

func TestCriteriaForRequest(t *testing.T) {
	baseURL, _ := url.Parse("http://example.com/")
	r := NewRequest(nil, "GET", baseURL, "v1", ContentConfig{}, nil, 0)
	c := newTestCriteria()
	c.AddCondition("con1", "val1")
	r.Criteria(c)
	if r.URL().String() != "http://example.com/v1?con1=val1" {
		t.Fatalf("full url should be 'http://example.com/v1?con1=val1' but got %s", r.URL().String())
	}
}

func TestCriteriaAndParamForRequest(t *testing.T) {
	baseURL, _ := url.Parse("http://example.com/")
	r := NewRequest(nil, "GET", baseURL, "v1", ContentConfig{}, nil, 0)
	c := newTestCriteria()
	c.AddCondition("con1", "val1")
	r.Param("p1", "v1").Criteria(c)
	query := r.URL().Query()
	if !reflect.DeepEqual(query["p1"], []string{"v1"}) {
		t.Fatalf("param p1 should be v1 but got %v", query["p1"])
	}
	if !reflect.DeepEqual(query["con1"], []string{"val1"}) {
		t.Fatalf("param con1 should be val1 but got %v", query["p1"])
	}
}
