// +build go1.7

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

package brn

import (
	"errors"
	"testing"
)

func TestParseBRN(t *testing.T) {
	cases := []struct {
		input string
		brn   BRN
		err   error
	}{
		{
			input: "invalid",
			err:   errors.New(invalidPrefix),
		},
		{
			input: "brn:nope",
			err:   errors.New(invalidSections),
		},
		{
			input: "brn:cloud:ecr:us-west-2:123456789012:repository/foo/bar",
			brn: BRN{
				Partition: "cloud",
				Service:   "ecr",
				Region:    "us-west-2",
				AccountID: "123456789012",
				Resource:  "repository/foo/bar",
			},
		},
		{
			input: "brn:cloud:elasticbeanstalk:us-east-1:123456789012:environment/My App/MyEnvironment",
			brn: BRN{
				Partition: "cloud",
				Service:   "elasticbeanstalk",
				Region:    "us-east-1",
				AccountID: "123456789012",
				Resource:  "environment/My App/MyEnvironment",
			},
		},
		{
			input: "brn:cloud:iam::123456789012:user/David",
			brn: BRN{
				Partition: "cloud",
				Service:   "iam",
				Region:    "",
				AccountID: "123456789012",
				Resource:  "user/David",
			},
		},
		{
			input: "brn:cloud:rds:eu-west-1:123456789012:db:mysql-db",
			brn: BRN{
				Partition: "cloud",
				Service:   "rds",
				Region:    "eu-west-1",
				AccountID: "123456789012",
				Resource:  "db:mysql-db",
			},
		},
		{
			input: "brn:cloud:s3:::my_corporate_bucket/exampleobject.png",
			brn: BRN{
				Partition: "cloud",
				Service:   "s3",
				Region:    "",
				AccountID: "",
				Resource:  "my_corporate_bucket/exampleobject.png",
			},
		},
		{
			input: "brn:cloud:faas:bj:16:function:test-h:2",
			brn: BRN{
				Partition: "cloud",
				Service:   "faas",
				Region:    "bj",
				AccountID: "16",
				Resource:  "function:test-h:2",
			},
		},
	}
	for _, tc := range cases {
		t.Run(tc.input, func(t *testing.T) {
			spec, err := Parse(tc.input)
			if tc.brn != spec {
				t.Errorf("Expected %q to parse as %v, but got %v", tc.input, tc.brn, spec)
			}
			if err == nil && tc.err != nil {
				t.Errorf("Expected err to be %v, but got nil", tc.err)
			} else if err != nil && tc.err == nil {
				t.Errorf("Expected err to be nil, but got %v", err)
			} else if err != nil && tc.err != nil && err.Error() != tc.err.Error() {
				t.Errorf("Expected err to be %v, but got %v", tc.err, err)
			}
		})
	}
}
