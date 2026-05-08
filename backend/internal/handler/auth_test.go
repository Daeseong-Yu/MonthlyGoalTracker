package handler

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/Daeseong-Yu/MonthlyGoalTracker/backend/internal/service"
	"github.com/gin-gonic/gin"
)

func TestWriteAuthErrorDoesNotExposeExistingSignupEmail(t *testing.T) {
	gin.SetMode(gin.TestMode)
	recorder := httptest.NewRecorder()
	context, _ := gin.CreateTestContext(recorder)

	writeAuthError(context, service.ErrEmailAlreadyExists)

	if recorder.Code != http.StatusBadRequest {
		t.Fatalf("expected status %d, got %d", http.StatusBadRequest, recorder.Code)
	}

	var response struct {
		Error string `json:"error"`
	}
	decodeJSON(t, recorder, &response)
	if response.Error == "email already exists" {
		t.Fatal("expected duplicate email to stay hidden")
	}
	if response.Error != "signup failed" {
		t.Fatalf("expected generic signup failure, got %q", response.Error)
	}
}

func TestWriteAuthErrorMapsAuthFlowErrors(t *testing.T) {
	testCases := []struct {
		name       string
		err        error
		statusCode int
		message    string
	}{
		{
			name:       "unverified email",
			err:        service.ErrEmailNotVerified,
			statusCode: http.StatusForbidden,
			message:    "email not verified",
		},
		{
			name:       "invalid verification token",
			err:        service.ErrInvalidVerificationToken,
			statusCode: http.StatusBadRequest,
			message:    "invalid verification token",
		},
		{
			name:       "invalid password reset token",
			err:        service.ErrInvalidPasswordResetToken,
			statusCode: http.StatusBadRequest,
			message:    "invalid password reset token",
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			gin.SetMode(gin.TestMode)
			recorder := httptest.NewRecorder()
			context, _ := gin.CreateTestContext(recorder)

			writeAuthError(context, testCase.err)

			if recorder.Code != testCase.statusCode {
				t.Fatalf("expected status %d, got %d", testCase.statusCode, recorder.Code)
			}

			var response struct {
				Error string `json:"error"`
			}
			decodeJSON(t, recorder, &response)
			if response.Error != testCase.message {
				t.Fatalf("expected error %q, got %q", testCase.message, response.Error)
			}
		})
	}
}
