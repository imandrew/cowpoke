// Package errors provides custom error types and utilities for cowpoke.
//
// This package provides error handling for various operations including:
// - Authentication errors
// - Configuration errors
// - HTTP errors
// - Validation errors
// - Multi-error handling
package errors

import (
	"errors"
	"fmt"
	"net/http"
)

// Error categories for cowpoke operations
var (
	ErrNotFound       = errors.New("resource not found")
	ErrUnauthorized   = errors.New("unauthorized")
	ErrInvalidInput   = errors.New("invalid input")
	ErrNetwork        = errors.New("network error")
	ErrConfiguration  = errors.New("configuration error")
	ErrAuthentication = errors.New("authentication error")
)

// AuthenticationError represents authentication-related errors
type AuthenticationError struct {
	ServerURL string
	AuthType  string
	Username  string
	Err       error
}

func (e *AuthenticationError) Error() string {
	return fmt.Sprintf("authentication failed for user '%s' on server '%s' (%s auth)", e.Username, e.ServerURL, e.AuthType)
}

func (e *AuthenticationError) Unwrap() error {
	return e.Err
}

func (e *AuthenticationError) Is(target error) bool {
	return errors.Is(target, ErrAuthentication)
}

// NewAuthenticationError creates a new authentication error
func NewAuthenticationError(serverURL, authType, username string, err error) *AuthenticationError {
	return &AuthenticationError{
		ServerURL: serverURL,
		AuthType:  authType,
		Username:  username,
		Err:       err,
	}
}

// IsAuthentication checks if an error is authentication-related
func IsAuthentication(err error) bool {
	return errors.Is(err, ErrAuthentication)
}

// ConfigurationError represents configuration-related errors
type ConfigurationError struct {
	Field   string
	Value   string
	Message string
	Err     error
}

func (e *ConfigurationError) Error() string {
	if e.Field != "" {
		return fmt.Sprintf("configuration error in field '%s': %s", e.Field, e.Message)
	}
	return fmt.Sprintf("configuration error: %s", e.Message)
}

func (e *ConfigurationError) Unwrap() error {
	return e.Err
}

func (e *ConfigurationError) Is(target error) bool {
	return errors.Is(target, ErrConfiguration)
}

// NewConfigurationError creates a new configuration error
func NewConfigurationError(field, value, message string, err error) *ConfigurationError {
	return &ConfigurationError{
		Field:   field,
		Value:   value,
		Message: message,
		Err:     err,
	}
}

// IsConfiguration checks if an error is configuration-related
func IsConfiguration(err error) bool {
	return errors.Is(err, ErrConfiguration)
}

// ValidationError represents input validation errors
type ValidationError struct {
	Field   string
	Value   string
	Rule    string
	Message string
}

func (e *ValidationError) Error() string {
	if e.Field != "" {
		return fmt.Sprintf("validation error in field '%s': %s", e.Field, e.Message)
	}
	return fmt.Sprintf("validation error: %s", e.Message)
}

func (e *ValidationError) Is(target error) bool {
	return errors.Is(target, ErrInvalidInput)
}

// NewValidationError creates a new validation error
func NewValidationError(field, value, rule, message string) *ValidationError {
	return &ValidationError{
		Field:   field,
		Value:   value,
		Rule:    rule,
		Message: message,
	}
}

// IsValidation checks if an error is validation-related
func IsValidation(err error) bool {
	return errors.Is(err, ErrInvalidInput)
}

// HTTPError represents an HTTP-related error
type HTTPError struct {
	StatusCode int
	Method     string
	URL        string
	Message    string
	Err        error
}

func (e *HTTPError) Error() string {
	if e.Message != "" {
		return fmt.Sprintf("HTTP %d %s %s: %s", e.StatusCode, e.Method, e.URL, e.Message)
	}
	return fmt.Sprintf("HTTP %d %s %s", e.StatusCode, e.Method, e.URL)
}

func (e *HTTPError) Unwrap() error {
	return e.Err
}

func (e *HTTPError) Is(target error) bool {
	switch e.StatusCode {
	case http.StatusNotFound:
		return errors.Is(target, ErrNotFound)
	case http.StatusUnauthorized, http.StatusForbidden:
		return errors.Is(target, ErrUnauthorized)
	case http.StatusBadRequest:
		return errors.Is(target, ErrInvalidInput)
	default:
		return false
	}
}

// NewHTTPError creates a new HTTP error
func NewHTTPError(statusCode int, method, url, message string) *HTTPError {
	return &HTTPError{
		StatusCode: statusCode,
		Method:     method,
		URL:        url,
		Message:    message,
	}
}

// NewHTTPErrorWithCause creates a new HTTP error with an underlying cause
func NewHTTPErrorWithCause(statusCode int, method, url, message string, err error) *HTTPError {
	return &HTTPError{
		StatusCode: statusCode,
		Method:     method,
		URL:        url,
		Message:    message,
		Err:        err,
	}
}

// IsHTTPStatus checks if an error represents a specific HTTP status
func IsHTTPStatus(err error, statusCode int) bool {
	var httpErr *HTTPError
	if errors.As(err, &httpErr) {
		return httpErr.StatusCode == statusCode
	}
	return false
}

// MultiError represents multiple errors that occurred together
type MultiError struct {
	Errors []error
}

func (e *MultiError) Error() string {
	if len(e.Errors) == 0 {
		return "no errors"
	}
	if len(e.Errors) == 1 {
		return e.Errors[0].Error()
	}
	return fmt.Sprintf("%s (and %d more errors)", e.Errors[0].Error(), len(e.Errors)-1)
}

func (e *MultiError) Unwrap() []error {
	return e.Errors
}

func (e *MultiError) Is(target error) bool {
	for _, err := range e.Errors {
		if errors.Is(err, target) {
			return true
		}
	}
	return false
}

func (e *MultiError) As(target any) bool {
	for _, err := range e.Errors {
		if errors.As(err, target) {
			return true
		}
	}
	return false
}

// NewMultiError creates a new multi-error from a slice of errors
func NewMultiError(errs []error) *MultiError {
	var filteredErrors []error
	for _, err := range errs {
		if err != nil {
			filteredErrors = append(filteredErrors, err)
		}
	}
	return &MultiError{Errors: filteredErrors}
}

// Join creates a MultiError from multiple errors, filtering out nils
func Join(errs ...error) error {
	var nonNilErrors []error
	for _, err := range errs {
		if err != nil {
			nonNilErrors = append(nonNilErrors, err)
		}
	}

	if len(nonNilErrors) == 0 {
		return nil
	}
	if len(nonNilErrors) == 1 {
		return nonNilErrors[0]
	}

	return NewMultiError(nonNilErrors)
}

// IsNotFound checks if an error represents a "not found" condition
func IsNotFound(err error) bool {
	return errors.Is(err, ErrNotFound) || IsHTTPStatus(err, http.StatusNotFound)
}

// IsUnauthorized checks if an error represents an authorization failure
func IsUnauthorized(err error) bool {
	return errors.Is(err, ErrUnauthorized) ||
		IsHTTPStatus(err, http.StatusUnauthorized) ||
		IsHTTPStatus(err, http.StatusForbidden)
}

// IsNetwork checks if an error is network-related
func IsNetwork(err error) bool {
	return errors.Is(err, ErrNetwork)
}
