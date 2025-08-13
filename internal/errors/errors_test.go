package errors

import (
	"errors"
	"net/http"
	"testing"
)

func TestHTTPError(t *testing.T) {
	err := NewHTTPError(404, "GET", "/api/clusters", "cluster not found")
	
	expectedMsg := "HTTP 404 GET /api/clusters: cluster not found"
	if err.Error() != expectedMsg {
		t.Errorf("Expected error message %q, got %q", expectedMsg, err.Error())
	}
	
	if !IsNotFound(err) {
		t.Error("Expected HTTPError with 404 to be identified as NotFound")
	}
	
	if !errors.Is(err, ErrNotFound) {
		t.Error("Expected HTTPError with 404 to match ErrNotFound")
	}
}

func TestHTTPErrorWithCause(t *testing.T) {
	cause := errors.New("network timeout")
	err := NewHTTPErrorWithCause(500, "POST", "/api/auth", "server error", cause)
	
	if !errors.Is(err, cause) {
		t.Error("Expected HTTPError to wrap the underlying cause")
	}
}

func TestHTTPErrorStatusMapping(t *testing.T) {
	tests := []struct {
		statusCode int
		expected   error
	}{
		{http.StatusNotFound, ErrNotFound},
		{http.StatusUnauthorized, ErrUnauthorized},
		{http.StatusForbidden, ErrUnauthorized},
		{http.StatusBadRequest, ErrInvalidInput},
	}
	
	for _, test := range tests {
		err := NewHTTPError(test.statusCode, "GET", "/test", "test error")
		if !errors.Is(err, test.expected) {
			t.Errorf("Expected HTTP %d to map to %v", test.statusCode, test.expected)
		}
	}
}

func TestAuthenticationError(t *testing.T) {
	cause := errors.New("invalid credentials")
	err := NewAuthenticationError("https://rancher.example.com", "github", "user123", cause)
	
	expectedMsg := "authentication failed for user 'user123' on server 'https://rancher.example.com' (github auth)"
	if err.Error() != expectedMsg {
		t.Errorf("Expected error message %q, got %q", expectedMsg, err.Error())
	}
	
	if !IsAuthentication(err) {
		t.Error("Expected AuthenticationError to be identified as authentication error")
	}
	
	if !errors.Is(err, ErrAuthentication) {
		t.Error("Expected AuthenticationError to match ErrAuthentication")
	}
	
	if !errors.Is(err, cause) {
		t.Error("Expected AuthenticationError to wrap the underlying cause")
	}
}

func TestConfigurationError(t *testing.T) {
	cause := errors.New("file not found")
	err := NewConfigurationError("config_path", "/invalid/path", "configuration file is missing", cause)
	
	expectedMsg := "configuration error in field 'config_path': configuration file is missing"
	if err.Error() != expectedMsg {
		t.Errorf("Expected error message %q, got %q", expectedMsg, err.Error())
	}
	
	if !IsConfiguration(err) {
		t.Error("Expected ConfigurationError to be identified as configuration error")
	}
	
	if !errors.Is(err, ErrConfiguration) {
		t.Error("Expected ConfigurationError to match ErrConfiguration")
	}
}

func TestValidationError(t *testing.T) {
	err := NewValidationError("auth_type", "invalid", "supported_values", "auth type must be one of: local, github, openldap")
	
	expectedMsg := "validation error in field 'auth_type': auth type must be one of: local, github, openldap"
	if err.Error() != expectedMsg {
		t.Errorf("Expected error message %q, got %q", expectedMsg, err.Error())
	}
	
	if !IsValidation(err) {
		t.Error("Expected ValidationError to be identified as validation error")
	}
	
	if !errors.Is(err, ErrInvalidInput) {
		t.Error("Expected ValidationError to match ErrInvalidInput")
	}
}

func TestMultiError(t *testing.T) {
	err1 := errors.New("first error")
	err2 := errors.New("second error")
	err3 := errors.New("third error")
	
	multiErr := NewMultiError([]error{err1, err2, err3})
	
	expectedMsg := "first error (and 2 more errors)"
	if multiErr.Error() != expectedMsg {
		t.Errorf("Expected error message %q, got %q", expectedMsg, multiErr.Error())
	}
	
	// Test Is method
	if !errors.Is(multiErr, err1) {
		t.Error("Expected MultiError to match first contained error")
	}
	
	if !errors.Is(multiErr, err2) {
		t.Error("Expected MultiError to match second contained error")
	}
	
	// Test Unwrap
	unwrapped := multiErr.Unwrap()
	if len(unwrapped) != 3 {
		t.Errorf("Expected 3 unwrapped errors, got %d", len(unwrapped))
	}
}

func TestMultiErrorSingleError(t *testing.T) {
	singleErr := errors.New("only error")
	multiErr := NewMultiError([]error{singleErr})
	
	if multiErr.Error() != "only error" {
		t.Errorf("Expected single error message, got %q", multiErr.Error())
	}
}

func TestMultiErrorEmpty(t *testing.T) {
	multiErr := NewMultiError([]error{})
	
	if multiErr.Error() != "no errors" {
		t.Errorf("Expected 'no errors' message, got %q", multiErr.Error())
	}
}

func TestJoin(t *testing.T) {
	err1 := errors.New("error 1")
	err2 := errors.New("error 2")
	
	// Test joining multiple errors
	joined := Join(err1, err2)
	if joined == nil {
		t.Fatal("Expected non-nil joined error")
	}
	
	var multiErr *MultiError
	if !errors.As(joined, &multiErr) {
		t.Error("Expected joined error to be MultiError")
	}
	
	// Test joining single error
	single := Join(err1)
	if single != err1 {
		t.Error("Expected single error to be returned as-is")
	}
	
	// Test joining with nils
	withNils := Join(err1, nil, err2, nil)
	if !errors.As(withNils, &multiErr) {
		t.Error("Expected joined error with nils to be MultiError")
	}
	
	if len(multiErr.Errors) != 2 {
		t.Errorf("Expected 2 errors after filtering nils, got %d", len(multiErr.Errors))
	}
	
	// Test joining only nils
	onlyNils := Join(nil, nil)
	if onlyNils != nil {
		t.Error("Expected joining only nils to return nil")
	}
}

func TestAs(t *testing.T) {
	httpErr := NewHTTPError(404, "GET", "/test", "not found")
	authErr := NewAuthenticationError("server", "local", "user", httpErr)
	multiErr := NewMultiError([]error{authErr})
	
	// Test As with MultiError
	var extractedHTTPErr *HTTPError
	if !errors.As(multiErr, &extractedHTTPErr) {
		t.Error("Expected to extract HTTPError from MultiError")
	}
	
	var extractedAuthErr *AuthenticationError
	if !errors.As(multiErr, &extractedAuthErr) {
		t.Error("Expected to extract AuthenticationError from MultiError")
	}
}

func TestConvenienceFunctions(t *testing.T) {
	httpNotFound := NewHTTPError(404, "GET", "/test", "not found")
	httpUnauth := NewHTTPError(401, "GET", "/test", "unauthorized")
	authErr := NewAuthenticationError("server", "local", "user", nil)
	configErr := NewConfigurationError("field", "value", "message", nil)
	validationErr := NewValidationError("field", "value", "rule", "message")
	
	// Test IsNotFound
	if !IsNotFound(httpNotFound) {
		t.Error("Expected IsNotFound to identify 404 HTTP error")
	}
	
	// Test IsUnauthorized
	if !IsUnauthorized(httpUnauth) {
		t.Error("Expected IsUnauthorized to identify 401 HTTP error")
	}
	
	// Test IsAuthentication
	if !IsAuthentication(authErr) {
		t.Error("Expected IsAuthentication to identify AuthenticationError")
	}
	
	// Test IsConfiguration
	if !IsConfiguration(configErr) {
		t.Error("Expected IsConfiguration to identify ConfigurationError")
	}
	
	// Test IsValidation
	if !IsValidation(validationErr) {
		t.Error("Expected IsValidation to identify ValidationError")
	}
	
	// Test IsHTTPStatus
	if !IsHTTPStatus(httpNotFound, 404) {
		t.Error("Expected IsHTTPStatus to identify specific status code")
	}
	
	if IsHTTPStatus(httpNotFound, 500) {
		t.Error("Expected IsHTTPStatus to reject wrong status code")
	}
}

func TestErrorChaining(t *testing.T) {
	baseErr := errors.New("base error")
	httpErr := NewHTTPErrorWithCause(500, "GET", "/test", "server error", baseErr)
	authErr := NewAuthenticationError("server", "local", "user", httpErr)
	
	// Test that we can unwrap through the chain
	if !errors.Is(authErr, baseErr) {
		t.Error("Expected to be able to unwrap through error chain")
	}
	
	if !errors.Is(authErr, ErrAuthentication) {
		t.Error("Expected authentication error to match ErrAuthentication")
	}
	
	// Test extracting specific error types from chain
	var extractedHTTPErr *HTTPError
	if !errors.As(authErr, &extractedHTTPErr) {
		t.Error("Expected to extract HTTPError from wrapped error")
	}
	
	if extractedHTTPErr.StatusCode != 500 {
		t.Errorf("Expected status code 500, got %d", extractedHTTPErr.StatusCode)
	}
}

// Additional tests to improve coverage

func TestConfigurationError_EmptyField(t *testing.T) {
	// Test configuration error without field (should hit different Error() path)
	err := NewConfigurationError("", "value", "generic configuration error", nil)
	
	expectedMsg := "configuration error: generic configuration error"
	if err.Error() != expectedMsg {
		t.Errorf("Expected error message %q, got %q", expectedMsg, err.Error())
	}
}

func TestConfigurationError_Unwrap(t *testing.T) {
	// Test Unwrap method for configuration error
	cause := errors.New("underlying error")
	err := NewConfigurationError("field", "value", "message", cause)
	
	unwrapped := err.Unwrap()
	if unwrapped != cause {
		t.Errorf("Expected unwrapped error to be %v, got %v", cause, unwrapped)
	}
}

func TestConfigurationError_UnwrapNil(t *testing.T) {
	// Test Unwrap when no underlying error
	err := NewConfigurationError("field", "value", "message", nil)
	
	unwrapped := err.Unwrap()
	if unwrapped != nil {
		t.Errorf("Expected unwrapped error to be nil, got %v", unwrapped)
	}
}

func TestHTTPError_EmptyMessage(t *testing.T) {
	// Test HTTPError without message (should hit different Error() path)
	err := NewHTTPError(404, "GET", "/api/test", "")
	
	expectedMsg := "HTTP 404 GET /api/test"
	if err.Error() != expectedMsg {
		t.Errorf("Expected error message %q, got %q", expectedMsg, err.Error())
	}
}

func TestIsHTTPStatus_NonHTTPError(t *testing.T) {
	// Test IsHTTPStatus with non-HTTP error (should return false)
	regularErr := errors.New("not an HTTP error")
	
	if IsHTTPStatus(regularErr, 404) {
		t.Error("Expected IsHTTPStatus to return false for non-HTTP error")
	}
}

func TestValidationError_EmptyField(t *testing.T) {
	// Test validation error without field (should hit different Error() path)
	err := NewValidationError("", "value", "rule", "generic validation error")
	
	expectedMsg := "validation error: generic validation error"
	if err.Error() != expectedMsg {
		t.Errorf("Expected error message %q, got %q", expectedMsg, err.Error())
	}
}

func TestMultiError_IsNonContained(t *testing.T) {
	// Test MultiError Is method with error not in the collection
	err1 := errors.New("error 1")
	err2 := errors.New("error 2")
	nonContained := errors.New("not contained")
	
	multiErr := NewMultiError([]error{err1, err2})
	
	if errors.Is(multiErr, nonContained) {
		t.Error("Expected MultiError to not match non-contained error")
	}
}

func TestMultiError_AsNonMatch(t *testing.T) {
	// Test MultiError As method with type that doesn't match
	err1 := errors.New("regular error")
	err2 := errors.New("another regular error")
	
	multiErr := NewMultiError([]error{err1, err2})
	
	var httpErr *HTTPError
	if errors.As(multiErr, &httpErr) {
		t.Error("Expected MultiError to not match HTTPError when it doesn't contain one")
	}
}

func TestIsNetwork(t *testing.T) {
	// Test IsNetwork function (currently 0% coverage)
	// First create a network error that would match ErrNetwork
	networkErr := &HTTPError{
		StatusCode: 502,
		Method:     "GET",
		URL:        "/test",
		Message:    "network error",
		Err:        ErrNetwork,
	}
	
	if !IsNetwork(networkErr) {
		t.Error("Expected IsNetwork to identify network error")
	}
	
	// Test with non-network error
	regularErr := errors.New("not a network error")
	if IsNetwork(regularErr) {
		t.Error("Expected IsNetwork to return false for non-network error")
	}
}