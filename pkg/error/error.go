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

package error

// Define all errors that Faas will encounter, fully compatible with AWS Lambda
// e.g:
//
// HTTP/1.1 419
// x-cloud-RequestID: b0e91dc8-3807-11e2-83c6-5912bf8ad066
// x-cloud-ErrorType: TooManyRequestsException
// x-amzn-RequestID: b0e91dc8-3807-11e2-83c6-5912bf8ad066
// x-amzn-ErrorType: TooManyRequestsException
// Content-Type: application/json
// Content-Length: 124
// Date: Mon, 26 Nov 2012 20:27:25 GMT
//
// {
//     "Code": "TooManyRequestsException",
//     "Message": "Too many requests",
//     "Type": "User",
// }

import (
	"fmt"
	"net/http"
	"time"

	"github.com/baidu/openless/pkg/util/json"
	"github.com/baidu/openless/pkg/util/logs"
)

// ErrorType is the alias of string
type ErrorType string

// Error types is the brief introduction of error types,
// and it's the key to navigate the message below.
const (
	// ResourceNotFoundException = The resource (for example, a function or access policy
	// statement) specified in the request does not exist.
	ResourceNotFoundException ErrorType = "ResourceNotFoundException"
	UserNotFoundException     ErrorType = "UserNotFoundException"

	// TooManyRequestsException xxx
	TooManyRequestsException ErrorType = "TooManyRequestsException"

	// External error face to user
	InvalidParameterValueException ErrorType = "InvalidParameterValueException"
	InvalidRequestContentException ErrorType = "InvalidRequestContentException"
	PolicyLengthExceededException  ErrorType = "PolicyLengthExceededException"
	ServiceException               ErrorType = "ServiceException"
	UnrecognizedClientException    ErrorType = "UnrecognizedClientException"
	UnsupportedMediaTypeException  ErrorType = "UnsupportedMediaTypeException"
	ValidationException            ErrorType = "ValidationException"

	// service defined error
	NotImplementedException      ErrorType = "NotImplementedException"
	RequestTimeoutException      ErrorType = "RequestTimeoutException"
	AccountProblemException      ErrorType = "AccountProblemException"
	InvalidInvokeCallerException ErrorType = "InvalidInvokeCallerException"
)

// Define error message that will be returned in http body
var message = map[ErrorType]string{
	InvalidParameterValueException: "One of the parameters in the request is invalid",
	InvalidRequestContentException: "The request body could not be parsed as JSON",
	PolicyLengthExceededException:  "Function access policy is limited to 20 KB",
	ServiceException:               "Service encountered an internal error",
	UnsupportedMediaTypeException:  "The content type of the Invoke request body is not JSON",
	ValidationException:            "Validation exception",
}

// BasicError struct
type BasicError struct {
	Code    ErrorType `json:"Code,omitempty"`
	Cause   string    `json:"Cause,omitempty"`
	Message string    `json:"Message"`
	Status  int       `json:"Status,omitempty"`
	Type    string    `json:"Type,omitempty"`
}

// Error struct compatible with standard `error` interface
func (err BasicError) Error() string {
	return string(err.toJSON())
}

func (err BasicError) toJSON() []byte {
	b, e := json.Marshal(err)
	if e != nil {
		logs.Errorf("marshal error failed, err=%v, jsonerr=%s", err, e.Error())
	}
	return b
}

// FinalError preserve the error stack
type FinalError struct {
	BasicError
	Backtrace []BasicError `json:"Backtrace,omitempty"`
}

// WithWarnLog xxx
func (err FinalError) WithWarnLog() FinalError {
	logs.Warnf(err.Error())
	return err
}

// WithErrorLog xxx
func (err FinalError) WithErrorLog() FinalError {
	// logs.ErrorDepth(1, err.Error())
	logs.Error(err.Error())
	return err
}

func (f FinalError) Error() string {
	b, _ := json.Marshal(f)
	return string(b)
}

// MarshalJSON xxx
func (f FinalError) MarshalJSON() ([]byte, error) {
	s := ``
	// basic error
	x, err := json.Marshal(f.BasicError)
	if err != nil {
		return nil, err
	}
	s += string(x[0 : len(x)-1])
	// backtrace
	if f.Backtrace != nil {
		s += `,"Backtrace":[`
		for i, e := range f.Backtrace {
			x, err = json.Marshal(e)
			if err != nil {
				return nil, err
			}
			s += string(x)
			if i < len(f.Backtrace)-1 {
				s += `,`
			}
		}
		s += `]`
	}
	s += `}`
	return []byte(s), nil
}

func (err FinalError) toJSON() []byte {
	b, e := json.Marshal(err)
	if e != nil {
		logs.Errorf("marshal error failed, err=%v, jsonerr=%s", err, e.Error())
	}
	return b
}

// WriteTo xxx
func (err FinalError) WriteTo(w http.ResponseWriter) FinalError {

	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("X-Amzn-ErrorType", string(err.Code))
	w.Header().Set("X-Bce-ErrorType", string(err.Code))

	w.WriteHeader(err.Status)

	b, _ := json.Marshal(err)
	w.Write(b)

	return err
}

// NewValidationException xxx
func NewValidationException(cause, message string, lasterr error) FinalError {
	return NewGenericException(BasicError{
		Code:    ValidationException,
		Cause:   cause,
		Message: message,
		Status:  http.StatusBadRequest,
	}, lasterr)
}

// NewInvalidParameterValueException xxx
func NewInvalidParameterValueException(cause string, lasterr error) FinalError {
	return NewGenericException(BasicError{
		Code:    InvalidParameterValueException,
		Cause:   cause,
		Message: "One of the parameters in the request is invalid",
		Status:  http.StatusBadRequest,
	}, lasterr)
}

// NewInvalidRequestContentException xxx
func NewInvalidRequestContentException(cause string, lasterr error) FinalError {
	return NewGenericException(BasicError{
		Code:    InvalidRequestContentException,
		Cause:   cause,
		Message: "The request body could not be parsed as JSON",
		Status:  http.StatusBadRequest,
	}, lasterr)
}

// NewServiceException xxx
func NewServiceException(cause string, lasterr error) FinalError {
	return NewGenericException(BasicError{
		Code:    ServiceException,
		Cause:   cause,
		Message: "Service encountered an internal error",
		Status:  http.StatusInternalServerError,
	}, lasterr)
}

// NewUnrecognizedClientException creates a UnrecognizedClientException
// HTTP status code is StatusForbidden (403)
func NewUnrecognizedClientException(cause string, lasterr error) FinalError {
	return NewGenericException(BasicError{
		Code:    UnrecognizedClientException,
		Cause:   cause,
		Message: "Access Denied",
		Status:  http.StatusForbidden,
	}, lasterr)
}

// NewRequestTimeoutException xxx
func NewRequestTimeoutException(cause string, timeout time.Duration, lasterr error) FinalError {
	return NewGenericException(BasicError{
		Code:    RequestTimeoutException,
		Cause:   cause,
		Message: fmt.Sprintf("Task timed out after %.2f seconds", timeout.Seconds()),
		Status:  http.StatusOK,
	}, lasterr)
}

// NewNotImplementedException xxx
func NewNotImplementedException() FinalError {
	return NewGenericException(BasicError{
		Code:    NotImplementedException,
		Message: "this method is not implemented",
		Status:  http.StatusInternalServerError,
	}, nil)
}

// NewGenericException xxx
func NewGenericException(err BasicError, lasterr error) FinalError {

	var returnerr FinalError

	if err.Status >= http.StatusBadRequest && err.Status < http.StatusInternalServerError {
		err.Type = "User"
	} else if err.Status >= http.StatusInternalServerError && err.Status <= http.StatusNetworkAuthenticationRequired {
		err.Type = "Server"
	}

	switch v := lasterr.(type) {
	case FinalError:
		returnerr = FinalError{
			BasicError: err,
			Backtrace:  append(v.Backtrace, v.BasicError),
		}
	case BasicError:
		returnerr = FinalError{
			BasicError: err,
			Backtrace:  []BasicError{v},
		}
	default:
		if lasterr == nil {
			returnerr = FinalError{
				BasicError: err,
				Backtrace:  nil,
			}
		} else {
			returnerr = FinalError{
				BasicError: err,
				Backtrace: []BasicError{
					BasicError{Message: lasterr.Error()},
				},
			}
		}
	}
	return returnerr
}

func GenericKunFinalError(err error) FinalError {
	switch v := err.(type) {
	case FinalError:
		return v
	default:
		e := NewServiceException("", v)
		return e
	}
}

// NewAccountProblemException xxx
func NewAccountProblemException(cause string, lasterr error) FinalError {
	return NewGenericException(BasicError{
		Code:    AccountProblemException,
		Cause:   cause,
		Message: "There is a problem with your account",
		Status:  http.StatusForbidden,
	}, lasterr)
}

// NewInvalidInvokeCallerException xxx
func NewInvalidInvokeCallerException(cause string, lasterr error) FinalError {
	return NewGenericException(BasicError{
		Code:    InvalidInvokeCallerException,
		Cause:   cause,
		Message: "Invalid caller id",
		Status:  http.StatusForbidden,
	}, lasterr)
}

// NewTooManyRequestsException xxx
func NewTooManyRequestsException(cause string, lasterr error) FinalError {
	return NewGenericException(BasicError{
		Code:    TooManyRequestsException,
		Cause:   cause,
		Message: "Too many requests",
		Status:  http.StatusTooManyRequests,
	}, lasterr)
}

func NewResourceNotFoundException(cause string, lasterr error) FinalError {
	return NewGenericException(BasicError{
		Code:    ResourceNotFoundException,
		Cause:   cause,
		Message: "The resource specified in the request does not exist",
		Status:  http.StatusNotFound,
	}, lasterr)
}
