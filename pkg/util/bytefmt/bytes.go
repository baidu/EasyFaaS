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

// Package bytefmt contains helper methods and constants for converting to and from a human-readable byte format.
//
//	bytefmt.ByteSize(100.5*bytefmt.Megabyte) // "100.5M"
//	bytefmt.ByteSize(uint64(1024)) // "1K"
//
package bytefmt

import (
	"errors"
	"fmt"
	"regexp"
	"strconv"
	"strings"
)

const (
	Byte     = 1.0
	Kilobyte = 1024 * Byte
	Megabyte = 1024 * Kilobyte
	Gigabyte = 1024 * Megabyte
	Terabyte = 1024 * Gigabyte
)

var bytesPattern = regexp.MustCompile(`(?i)^(-?\d+(?:\.\d+)?)([KMGT]B?|B)$`)

var invalidByteQuantityError = errors.New("Byte quantity must be a positive integer with a unit of measurement like M, MB, G, or GB")

// ByteSize returns a human-readable byte string of the form 10M, 12.5K, and so forth.  The following units are available:
//	T: Terabyte
//	G: Gigabyte
//	M: Megabyte
//	K: Kilobyte
//	B: Byte
// The unit that results in the smallest number greater than or equal to 1 is always chosen.
func ByteSize(bytes uint64) string {
	unit := ""
	value := float32(bytes)

	switch {
	case bytes >= Terabyte:
		unit = "T"
		value = value / Terabyte
	case bytes >= Gigabyte:
		unit = "G"
		value = value / Gigabyte
	case bytes >= Megabyte:
		unit = "M"
		value = value / Megabyte
	case bytes >= Kilobyte:
		unit = "K"
		value = value / Kilobyte
	case bytes >= Byte:
		unit = "B"
	case bytes == 0:
		return "0"
	}

	stringValue := fmt.Sprintf("%.1f", value)
	stringValue = strings.TrimSuffix(stringValue, ".0")
	return fmt.Sprintf("%s%s", stringValue, unit)
}

// ToMegabytes parses a string formatted by ByteSize as megabytes.
func ToMegabytes(s string) (uint64, error) {
	bytes, err := ToBytes(s)
	if err != nil {
		return 0, err
	}

	return bytes / Megabyte, nil
}

// ToBytes parses a string formatted by ByteSize as bytes.
func ToBytes(s string) (uint64, error) {
	parts := bytesPattern.FindStringSubmatch(strings.TrimSpace(s))
	if len(parts) < 3 {
		return 0, invalidByteQuantityError
	}

	value, err := strconv.ParseFloat(parts[1], 64)
	if err != nil || value <= 0 {
		return 0, invalidByteQuantityError
	}

	var bytes uint64
	unit := strings.ToUpper(parts[2])
	switch unit[:1] {
	case "T":
		bytes = uint64(value * Terabyte)
	case "G":
		bytes = uint64(value * Gigabyte)
	case "M":
		bytes = uint64(value * Megabyte)
	case "K":
		bytes = uint64(value * Kilobyte)
	case "B":
		bytes = uint64(value * Byte)
	default:
		bytes = uint64(value * Byte)
	}

	return bytes, nil
}
