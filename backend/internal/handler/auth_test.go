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
