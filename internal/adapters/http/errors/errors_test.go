package errors

import "testing"

func TestErrorContracts(t *testing.T) {
	codes := []string{
		CodeInvalidRequest,
		CodeValidation,
		CodeInvalidUserID,
		CodeEmailAlreadyExists,
		CodeInvalidCredentials,
		CodeUserNotFound,
		CodeUnauthorized,
		CodeInternal,
	}
	seen := make(map[string]struct{}, len(codes))
	for _, code := range codes {
		if code == "" {
			t.Fatal("error code is empty")
		}
		if _, exists := seen[code]; exists {
			t.Errorf("duplicate error code %q", code)
		}
		seen[code] = struct{}{}
	}

	messages := []string{
		MessageInvalidRequest,
		MessageInvalidUserID,
		MessageEmailAlreadyExists,
		MessageInvalidCredentials,
		MessageUserNotFound,
		MessageUnauthorized,
		MessageInternal,
	}
	for _, message := range messages {
		if message == "" {
			t.Fatal("error message is empty")
		}
	}
}
