package handler

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/Daeseong-Yu/MonthlyGoalTracker/backend/internal/principal"
	"github.com/gin-gonic/gin"
)

type requestOption func(*http.Request) *http.Request

func performRequest(t *testing.T, method string, path string, body string, register func(*gin.Engine), opts ...requestOption) *httptest.ResponseRecorder {
	t.Helper()

	gin.SetMode(gin.TestMode)

	router := gin.New()
	register(router)

	request := httptest.NewRequest(method, path, bytes.NewReader([]byte(body)))
	request.Header.Set("Content-Type", "application/json")
	for _, opt := range opts {
		request = opt(request)
	}

	recorder := httptest.NewRecorder()
	router.ServeHTTP(recorder, request)

	return recorder
}

func withPrincipal(current principal.Principal) requestOption {
	return func(request *http.Request) *http.Request {
		return request.WithContext(principal.WithContext(request.Context(), current))
	}
}

func decodeJSON(t *testing.T, recorder *httptest.ResponseRecorder, target any) {
	t.Helper()

	data := recorder.Body.Bytes()
	if err := json.Unmarshal(data, target); err != nil {
		t.Fatalf("failed to decode JSON %q: %v", string(data), err)
	}
}

func date(year int, month time.Month, day int) time.Time {
	return time.Date(year, month, day, 0, 0, 0, 0, time.UTC)
}
