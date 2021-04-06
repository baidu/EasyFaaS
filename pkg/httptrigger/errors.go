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

package httptrigger

import (
	"net/http"

	kunErr "github.com/baidu/openless/pkg/error"
)

const (
	InvalidRequestException kunErr.ErrorType = "InvalidRequestException"
	NotFountException       kunErr.ErrorType = "NotFountException"
	InternalException       kunErr.ErrorType = "InternalException"
	BadGatewayException     kunErr.ErrorType = "BadGatewayException"
)

var message = map[kunErr.ErrorType]string{
	InvalidRequestException: "Invalid Request",
	NotFountException:       "Not Found",
	InternalException:       "Internal Error",
	BadGatewayException:     "Bad Gateway",
}

func NewInvalidRequestException(cause string, lasterr error) kunErr.FinalError {
	return kunErr.NewGenericException(kunErr.BasicError{
		Code:    InvalidRequestException,
		Cause:   cause,
		Message: message[InvalidRequestException],
		Status:  http.StatusBadRequest,
	}, lasterr)
}

func NewNotFountException(cause string, lasterr error) kunErr.FinalError {
	return kunErr.NewGenericException(kunErr.BasicError{
		Code:    NotFountException,
		Cause:   cause,
		Message: message[NotFountException],
		Status:  http.StatusNotFound,
	}, lasterr)
}

func NewInternalException(cause string, lasterr error) kunErr.FinalError {
	return kunErr.NewGenericException(kunErr.BasicError{
		Code:    InternalException,
		Cause:   cause,
		Message: message[InternalException],
		Status:  http.StatusInternalServerError,
	}, lasterr)
}

func NewBadGatewayException(cause string, lasterr error) kunErr.FinalError {
	return kunErr.NewGenericException(kunErr.BasicError{
		Code:    BadGatewayException,
		Cause:   cause,
		Message: message[BadGatewayException],
		Status:  http.StatusBadGateway,
	}, lasterr)
}
