package integration

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
)

// TestConfiguredUserLogin...
func TestConfiguredUserLogin(t *testing.T) {
	url := "/api/v1/login"
	method := "POST"

	payload := []byte(fmt.Sprintf(`{"email": "khalid@gmail.com", "password": "%s"}`, tempPass))
	req, err := http.NewRequest(method, url, bytes.NewReader(payload))
	if err != nil {
		t.Fatalf("Failed to read response body: %v", err)
	}
	req.Header.Add("Content-Type", "application/json")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)
	responseBody, err := io.ReadAll(w.Result().Body)
	if err != nil {
		t.Fatalf("Failed to read response body: %v", err)
	}
	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200 OK; got %v", w.Code)
		return
	}
	var resultMap map[string]interface{}
	if err := json.Unmarshal(responseBody, &resultMap); err != nil {
		t.Fatalf("Failed to unmarshal response body: %v", err)
	}
	token = resultMap["access_token"].(string)
	if resultMap["password_change_required"].(bool) == false {
		t.Errorf("password_change_required should be true")
	}
}

// TestConfiguredUserResetPassword...
func TestConfiguredUserResetPassword(t *testing.T) {
	url := "/api/v1/user/self/password"
	method := "PUT"

	payload := []byte(`{"password": "khalid123"}`)
	req, err := http.NewRequest(method, url, bytes.NewReader(payload))
	if err != nil {
		t.Fatalf("Failed to read response body: %v", err)
	}
	req.Header.Add("Content-Type", "application/json")
	req.Header.Add("Authorization", "Bearer "+token)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200 OK; got %v", w.Code)
	}
}
