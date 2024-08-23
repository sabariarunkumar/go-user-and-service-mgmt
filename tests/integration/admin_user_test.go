package integration

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

var (
	token    string
	router   *gin.Engine
	tempPass string
)

// truncateTables...
func truncateTables(db *gorm.DB) (err error) {
	err = db.Exec("TRUNCATE TABLE service CASCADE").Error
	if err != nil {
		return
	}
	err = db.Exec(`TRUNCATE TABLE "user"`).Error
	if err != nil {
		return
	}
	err = db.Exec(`TRUNCATE TABLE "user_role"`).Error
	if err != nil {
		return
	}
	err = db.Exec(`REFRESH MATERIALIZED VIEW name_sorted_service`).Error
	if err != nil {
		return
	}
	return
}

// TestAdminLogin...
func TestAdminLogin(t *testing.T) {
	url := "/api/v1/login"
	method := "POST"

	payload := []byte(`{"email": "admin@mgmtportal.com", "password": "admin123"}`)
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

// TestAdminResetPassword...
func TestAdminResetPassword(t *testing.T) {
	url := "/api/v1/user/self/password"
	method := "PUT"

	payload := []byte(`{"password": "admin123"}`)
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

// TestAdminAddUser...
func TestAdminAddUser(t *testing.T) {
	url := "/api/v1/user"
	method := "POST"

	payload := []byte(`{"name":"khalid","email":"khalid@gmail.com","role":"advanced"}`)
	req, err := http.NewRequest(method, url, bytes.NewReader(payload))
	if err != nil {
		t.Fatalf("Failed to read response body: %v", err)
	}
	req.Header.Add("Content-Type", "application/json")
	req.Header.Add("Authorization", "Bearer "+token)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusCreated {
		t.Errorf("Expected status 201 code; got %v", w.Code)
	}
	responseBody, err := io.ReadAll(w.Result().Body)
	if err != nil {
		t.Fatalf("Failed to read response body: %v", err)
	}
	var resultMap map[string]interface{}
	if err := json.Unmarshal(responseBody, &resultMap); err != nil {
		t.Fatalf("Failed to unmarshal response body: %v", err)
	}
	tempPass = resultMap["temporary_password"].(string)
}

// TestAdminListUsers...
func TestAdminListUsers(t *testing.T) {
	url := "/api/v1/users"
	method := "GET"

	req, err := http.NewRequest(method, url, nil)
	if err != nil {
		t.Fatalf("Failed to read response body: %v", err)
	}
	req.Header.Add("Authorization", "Bearer "+token)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200 OK; got %v", w.Code)
	}
	responseBody, err := io.ReadAll(w.Result().Body)
	if err != nil {
		t.Fatalf("Failed to read response body: %v", err)
	}
	var resultMap map[string]interface{}
	if err := json.Unmarshal(responseBody, &resultMap); err != nil {
		t.Fatalf("Failed to unmarshal response body: %v", err)
	}
	if resultMap["TotalItems"].(float64) != 1 {
		t.Errorf("Admin User is not being listed")
	}
}
