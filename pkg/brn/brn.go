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

// Package brn provides a parser for interacting with BCE Resource Names.
package brn

import (
	"crypto/md5"
	"errors"
	"fmt"
	"io"
	"strings"
)

const (
	brnDelimiter = ":"
	brnSections  = 6
	brnPrefix    = "brn:"

	// zero-indexed
	sectionPartition = 1
	sectionService   = 2
	sectionRegion    = 3
	sectionAccountID = 4
	sectionResource  = 5

	// errors
	invalidPrefix   = "brn: invalid prefix"
	invalidSections = "brn: not enough sections"
)

// BRN captures the individual fields of an BCE Resource Name.
// See http://docs.aws.amazon.com/general/latest/gr/aws-arns-and-namespaces.html for more information.
type BRN struct {
	// The partition that the resource is in. For standard BCE regions, the partition is "cloud". If you have resources in
	// other partitions, the partition is "cloud-partitionname". For example, the partition for resources in the China
	// (Beijing) region is "cloud-cn".
	Partition string

	// The service namespace that identifies the BCE product. For a list of namespaces, see
	// http://docs.aws.amazon.com/general/latest/gr/aws-arns-and-namespaces.html#genref-aws-service-namespaces.
	Service string

	// The region the resource resides in. Note that the BRNs for some resources do not require a region, so this
	// component might be omitted.
	Region string

	// The ID of the BCE account that owns the resource, without the hyphens. For example, 123456789012. Note that the
	// BRNs for some resources don't require an account number, so this component might be omitted.
	AccountID string

	// The content of this part of the BRN varies by service. It often includes an indicator of the type of resource —
	// for example, an IAM user or BCE RDS database - followed by a slash (/) or a colon (:), followed by the
	// resource name itself. Some services allows paths for resource names, as described in
	// http://docs.aws.amazon.com/general/latest/gr/aws-arns-and-namespaces.html#arns-paths.
	Resource string
}

// Parse parses an BRN into its constituent parts.
//
// Some example BRNs:
// brn:cloud:elasticbeanstalk:us-east-1:123456789012:environment/My App/MyEnvironment
// brn:cloud:iam::123456789012:user/David
// brn:cloud:rds:eu-west-1:123456789012:db:mysql-db
// brn:cloud:s3:::my_corporate_bucket/exampleobject.png
func Parse(brn string) (BRN, error) {
	if !strings.HasPrefix(brn, brnPrefix) {
		return BRN{}, errors.New(invalidPrefix)
	}
	sections := strings.SplitN(brn, brnDelimiter, brnSections)
	if len(sections) != brnSections {
		return BRN{}, errors.New(invalidSections)
	}
	return BRN{
		Partition: sections[sectionPartition],
		Service:   sections[sectionService],
		Region:    sections[sectionRegion],
		AccountID: sections[sectionAccountID],
		Resource:  sections[sectionResource],
	}, nil
}

// String returns the canonical representation of the BRN
func (brn BRN) String() string {
	return brnPrefix +
		brn.Partition + brnDelimiter +
		brn.Service + brnDelimiter +
		brn.Region + brnDelimiter +
		brn.AccountID + brnDelimiter +
		brn.Resource
}

func GenerateCommonBrn(service, region, uid, resource, qualifier string) string {
	if len(qualifier) > 0 {
		resource = fmt.Sprintf("%s:%s", resource, qualifier)
	}

	b := &BRN{
		Partition: "cloud",
		Service:   service,
		Region:    region,
		AccountID: Md5BceUid(uid),
		Resource:  resource,
	}

	return b.String()
}

func Md5BceUid(uid string) string {
	h := md5.New()
	io.WriteString(h, uid+"cloud-cfc-2017")
	return fmt.Sprintf("%x", h.Sum(nil))
}
