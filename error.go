package cbinsights

import (
	"encoding/json"
	"errors"
	"fmt"
)

// ErrInsights is the base error for any Operational Insights error that is not captured by a more specific error.
var ErrInsights = errors.New("operational insights error")

// ErrInvalidCredential occurs when invalid credentials are provided leading to errors in things like authentication.
var ErrInvalidCredential = errors.New("invalid credential")

// ErrTimeout occurs when a timeout is reached while waiting for a response.
// This is returned when a server timeout occurs, or an operation fails to be sent within the dispatch timeout.
var ErrTimeout = errors.New("timeout error")

// ErrQuery occurs when a server error is encountered while executing a query, excluding errors that caught by
// ErrInvalidCredential or ErrTimeout.
var ErrQuery = errors.New("query error")

// ErrInvalidArgument occurs when an invalid argument is provided to a function.
var ErrInvalidArgument = errors.New("invalid argument")

// ErrClosed occurs when an entity was used after it was closed.
var ErrClosed = errors.New("closed")

// ErrUnmarshal occurs when an entity could not be unmarshalled.
var ErrUnmarshal = errors.New("unmarshalling error")

// ErrServiceUnavailable occurs when the Operational Insights service, or a part of the system in the path to it, is unavailable.
var ErrServiceUnavailable = errors.New("service unavailable")

// ErrQueryNotFound occurs when a query handle or its results are not found,
// typically because they have been discarded or canceled.
var ErrQueryNotFound = errors.New("query not found")

type insightsErrorDesc struct {
	Code    uint32
	Message string
}

func (e insightsErrorDesc) MarshalJSON() ([]byte, error) {
	b, err := json.Marshal(struct {
		Code    uint32 `json:"code"`
		Message string `json:"msg"`
	}{
		Code:    e.Code,
		Message: e.Message,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to marshal operational insights error description: %s", err) // nolint: err113, errorlint
	}

	return b, nil
}

// InsightsError occurs when an error is encountered while interacting with the Operational Insights service.
type InsightsError struct {
	cause   error
	message string

	errors           []insightsErrorDesc
	statement        string
	endpoint         string
	httpResponseCode int
	retries          uint32
}

func newInsightsError(cause error, statement, endpoint string, statusCode int, retries uint32) InsightsError {
	if cause == nil {
		cause = ErrInsights
	}

	return InsightsError{
		cause:            cause,
		errors:           nil,
		statement:        statement,
		endpoint:         endpoint,
		message:          "",
		httpResponseCode: statusCode,
		retries:          retries,
	}
}

func (e InsightsError) withMessage(message string) *InsightsError {
	e.message = message

	return &e
}

// Error returns the string representation of an Operational Insights error.
func (e InsightsError) Error() string {
	errBytes, _ := json.Marshal(struct {
		Statement        string              `json:"statement,omitempty"`
		Errors           []insightsErrorDesc `json:"errors,omitempty"`
		Message          string              `json:"message,omitempty"`
		Endpoint         string              `json:"endpoint,omitempty"`
		HTTPResponseCode int                 `json:"status_code,omitempty"`
		Retries          uint32              `json:"retries,omitempty"`
	}{
		Statement:        e.statement,
		Errors:           e.errors,
		Message:          e.message,
		Endpoint:         e.endpoint,
		HTTPResponseCode: e.httpResponseCode,
		Retries:          e.retries,
	})

	return e.cause.Error() + " | " + string(errBytes)
}

// Unwrap returns the underlying reason for the error.
func (e InsightsError) Unwrap() error {
	if e.cause == nil {
		return ErrInsights
	}

	return e.cause
}

// QueryError occurs when an error is returned in the errors field of the response body of a response
// from the query server.
type QueryError struct {
	cause   *InsightsError
	code    int
	message string
}

// Code returns the error code from the server for this error.
func (e QueryError) Code() int {
	return e.code
}

// Message returns the error message from the server for this error.
func (e QueryError) Message() string {
	return e.message
}

// Error returns the string representation of a query error.
func (e QueryError) Error() string {
	return fmt.Errorf("%w", e.cause).Error()
}

// Unwrap returns the underlying reason for the error.
func (e QueryError) Unwrap() error {
	return e.cause
}

func (e QueryError) withErrors(errors []insightsErrorDesc) *QueryError {
	e.cause.errors = errors

	return &e
}

// nolint: unused
func newQueryError(cause error, statement, endpoint string, statusCode int, code int, message string, retries uint32) QueryError {
	if cause == nil {
		cause = ErrQuery
	}

	return QueryError{
		cause: &InsightsError{
			cause:            cause,
			errors:           nil,
			statement:        statement,
			endpoint:         endpoint,
			message:          "",
			httpResponseCode: statusCode,
			retries:          retries,
		},
		code:    code,
		message: message,
	}
}

type invalidArgumentError struct {
	ArgumentName string
	Reason       string
}

func (e invalidArgumentError) Error() string {
	return fmt.Sprintf("%s %s - %s", e.Unwrap(), e.ArgumentName, e.Reason)
}

func (e invalidArgumentError) Unwrap() error {
	return ErrInvalidArgument
}

type unmarshalError struct {
	Reason string
}

func (e unmarshalError) Error() string {
	return fmt.Sprintf("failed to unmarshal - %s", e.Reason)
}

func (e unmarshalError) Unwrap() error {
	return ErrUnmarshal
}
