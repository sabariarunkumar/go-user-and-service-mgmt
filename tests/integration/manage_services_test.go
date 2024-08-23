package integration

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strconv"
	"testing"
)

var (
	advancedUserID float64
)

// TestAdvancedUserAddService...
func TestAdvancedUserAddService(t *testing.T) {
	url := "/api/v1/service"
	method := "POST"

	payload := []byte(`{"name":"postman-4","description":"postman  sdf"}`)
	req, err := http.NewRequest(method, url, bytes.NewReader(payload))
	if err != nil {
		t.Fatalf("Failed to read response body: %v", err)
	}
	req.Header.Add("Content-Type", "application/json")
	req.Header.Add("Authorization", "Bearer "+token)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusCreated {
		t.Errorf("Expected status 201 Created; got %v", w.Code)
	}

	responseBody, err := io.ReadAll(w.Result().Body)
	if err != nil {
		t.Fatalf("Failed to read response body: %v", err)
	}
	var resultMap map[string]interface{}
	if err := json.Unmarshal(responseBody, &resultMap); err != nil {
		t.Fatalf("Failed to unmarshal response body: %v", err)
	}
	advancedUserID = resultMap["id"].(float64)
}

// TestAdvancedUserListServiceVersion...
func TestAdvancedUserListServiceVersion(t *testing.T) {
	url := "/api/v1/service/" + strconv.Itoa(int(advancedUserID)) + "/versions"
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
	if resultMap["TotalItems"].(float64) != 0 {
		t.Errorf("Expected services not found")
	}

}

// TestAdvancedUserAddsVersionV1...
func TestAdvancedUserAddsVersionV1(t *testing.T) {
	url := "/api/v1/service/" + strconv.Itoa(int(advancedUserID)) + "/version"
	method := "POST"

	payload := []byte(`{"tag":"1","info":"version added v1"}`)
	req, err := http.NewRequest(method, url, bytes.NewReader(payload))
	if err != nil {
		t.Fatalf("Failed to read response body: %v", err)
	}
	req.Header.Add("Content-Type", "application/json")
	req.Header.Add("Authorization", "Bearer "+token)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusCreated {
		t.Errorf("Expected status 201 CREATED; got %v", w.Code)
	}
}

// TestAdvancedUserAddsVersionV2...
func TestAdvancedUserAddsVersionV2(t *testing.T) {
	url := "/api/v1/service/" + strconv.Itoa(int(advancedUserID)) + "/version"
	method := "POST"

	payload := []byte(`{"tag":"2","info":"version added v2"}`)
	req, err := http.NewRequest(method, url, bytes.NewReader(payload))
	if err != nil {
		t.Fatalf("Failed to read response body: %v", err)
	}
	req.Header.Add("Content-Type", "application/json")
	req.Header.Add("Authorization", "Bearer "+token)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusCreated {
		t.Errorf("Expected status 201 Created; got %v", w.Code)
	}

}

// TestAdvancedUserListService...
func TestAdvancedUserListService(t *testing.T) {
	url := "/api/v1/services"
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
		t.Errorf("Expected services not found")
	}
	if resultMap["Data"].([]interface{})[0].(map[string]interface{})["versionCount"].(float64) != 2 {
		t.Errorf("Expected service version count not reflected")

	}
}

// TestAdvancedUserListServiceWithSortFilter...
func TestAdvancedUserListServiceWithSortFilter(t *testing.T) {
	url := "/api/v1/services?sort_by=name&name=postma"
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
		t.Errorf("Expected services not found")
	}
	if resultMap["Data"].([]interface{})[0].(map[string]interface{})["versionCount"].(float64) != 2 {
		t.Errorf("Expected service version count not reflected")

	}
}
