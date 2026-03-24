package handlers

import (
	"bytes"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
)

func init() {
	gin.SetMode(gin.TestMode)
}

// TestListSecrets_Success tests successful ListSecrets calls with varying data.
func TestListSecrets_Success(t *testing.T) {
	tests := []struct {
		name     string
		items    map[string]string
		expected []string
	}{
		{
			name:     "empty list",
			items:    map[string]string{},
			expected: []string{},
		},
		{
			name: "single item",
			items: map[string]string{
				"secret1": "value1",
			},
			expected: []string{"secret1"},
		},
		{
			name: "multiple items",
			items: map[string]string{
				"secret1": "value1",
				"secret2": "value2",
				"secret3": "value3",
			},
			expected: []string{"secret1", "secret2", "secret3"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			backend := NewMockBackend()
			for name, value := range tt.items {
				backend.CreateItem(nil, name, value, nil)
			}

			handler := NewSecretHandler(backend, nil)

			w := httptest.NewRecorder()
			c, _ := gin.CreateTestContext(w)
			c.Request = httptest.NewRequest(http.MethodGet, "/secrets", nil)

			handler.ListSecrets(c)

			if w.Code != http.StatusOK {
				t.Errorf("Expected status %d, got %d", http.StatusOK, w.Code)
			}

			var resp ListSecretsResponse
			if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
				t.Fatalf("Failed to unmarshal response: %v", err)
			}

			if len(resp.Secrets) != len(tt.expected) {
				t.Errorf("Expected %d secrets, got %d", len(tt.expected), len(resp.Secrets))
			}

			// Check all expected secrets are present (order may vary)
			secretMap := make(map[string]bool)
			for _, s := range resp.Secrets {
				secretMap[s] = true
			}
			for _, expected := range tt.expected {
				if !secretMap[expected] {
					t.Errorf("Expected secret %q not found in response", expected)
				}
			}
		})
	}
}

// TestListSecrets_BackendError tests ListSecrets error handling.
func TestListSecrets_BackendError(t *testing.T) {
	backend := NewMockBackend()
	backend.SetListError(errors.New("backend failure"))

	handler := NewSecretHandler(backend, nil)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodGet, "/secrets", nil)

	handler.ListSecrets(c)

	if w.Code != http.StatusInternalServerError {
		t.Errorf("Expected status %d, got %d", http.StatusInternalServerError, w.Code)
	}

	var resp map[string]interface{}
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("Failed to unmarshal response: %v", err)
	}

	if _, ok := resp["error"]; !ok {
		t.Error("Expected error field in response")
	}
}

// TestGetSecret_Success tests successful GetSecret retrieval.
func TestGetSecret_Success(t *testing.T) {
	backend := NewMockBackend()
	backend.CreateItem(nil, "test-secret", "test-value", nil)

	handler := NewSecretHandler(backend, nil)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodGet, "/secrets/test-secret", nil)
	c.Params = gin.Params{{Key: "name", Value: "test-secret"}}

	handler.GetSecret(c)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status %d, got %d", http.StatusOK, w.Code)
	}

	var resp SecretResponse
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("Failed to unmarshal response: %v", err)
	}

	if resp.Name != "test-secret" {
		t.Errorf("Expected name %q, got %q", "test-secret", resp.Name)
	}

	if resp.Value != "test-value" {
		t.Errorf("Expected value %q, got %q", "test-value", resp.Value)
	}
}

// TestGetSecret_NotFound tests GetSecret with non-existent secret.
func TestGetSecret_NotFound(t *testing.T) {
	backend := NewMockBackend()
	handler := NewSecretHandler(backend, nil)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodGet, "/secrets/nonexistent", nil)
	c.Params = gin.Params{{Key: "name", Value: "nonexistent"}}

	handler.GetSecret(c)

	if w.Code != http.StatusNotFound {
		t.Errorf("Expected status %d, got %d", http.StatusNotFound, w.Code)
	}

	var resp map[string]interface{}
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("Failed to unmarshal response: %v", err)
	}

	if resp["error"] != "secret not found" {
		t.Errorf("Expected error %q, got %q", "secret not found", resp["error"])
	}
}

// TestGetSecret_BackendError tests GetSecret backend failure.
func TestGetSecret_BackendError(t *testing.T) {
	backend := NewMockBackend()
	backend.SetGetError(errors.New("backend failure"))

	handler := NewSecretHandler(backend, nil)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodGet, "/secrets/test-secret", nil)
	c.Params = gin.Params{{Key: "name", Value: "test-secret"}}

	handler.GetSecret(c)

	if w.Code != http.StatusInternalServerError {
		t.Errorf("Expected status %d, got %d", http.StatusInternalServerError, w.Code)
	}

	var resp map[string]interface{}
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("Failed to unmarshal response: %v", err)
	}

	if _, ok := resp["error"]; !ok {
		t.Error("Expected error field in response")
	}
}

// TestCreateSecret_Success tests successful secret creation.
func TestCreateSecret_Success(t *testing.T) {
	backend := NewMockBackend()
	handler := NewSecretHandler(backend, nil)

	reqBody := CreateSecretRequest{
		Name:  "new-secret",
		Value: "new-value",
	}
	body, _ := json.Marshal(reqBody)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodPost, "/secrets", bytes.NewReader(body))
	c.Request.Header.Set("Content-Type", "application/json")

	handler.CreateSecret(c)

	if w.Code != http.StatusCreated {
		t.Errorf("Expected status %d, got %d", http.StatusCreated, w.Code)
	}

	var resp SecretResponse
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("Failed to unmarshal response: %v", err)
	}

	if resp.Name != "new-secret" {
		t.Errorf("Expected name %q, got %q", "new-secret", resp.Name)
	}

	// Verify secret was actually created
	exists, _ := backend.ItemExists(nil, "new-secret", nil)
	if !exists {
		t.Error("Secret was not created in backend")
	}
}

// TestCreateSecret_MissingName tests CreateSecret with missing name field.
func TestCreateSecret_MissingName(t *testing.T) {
	backend := NewMockBackend()
	handler := NewSecretHandler(backend, nil)

	reqBody := map[string]interface{}{
		"value": "some-value",
	}
	body, _ := json.Marshal(reqBody)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodPost, "/secrets", bytes.NewReader(body))
	c.Request.Header.Set("Content-Type", "application/json")

	handler.CreateSecret(c)

	if w.Code != http.StatusBadRequest {
		t.Errorf("Expected status %d, got %d", http.StatusBadRequest, w.Code)
	}

	var resp map[string]interface{}
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("Failed to unmarshal response: %v", err)
	}

	if _, ok := resp["error"]; !ok {
		t.Error("Expected error field in response")
	}
}

// TestCreateSecret_MissingValue tests CreateSecret with missing value field.
func TestCreateSecret_MissingValue(t *testing.T) {
	backend := NewMockBackend()
	handler := NewSecretHandler(backend, nil)

	reqBody := map[string]interface{}{
		"name": "test-secret",
	}
	body, _ := json.Marshal(reqBody)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodPost, "/secrets", bytes.NewReader(body))
	c.Request.Header.Set("Content-Type", "application/json")

	handler.CreateSecret(c)

	if w.Code != http.StatusBadRequest {
		t.Errorf("Expected status %d, got %d", http.StatusBadRequest, w.Code)
	}

	var resp map[string]interface{}
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("Failed to unmarshal response: %v", err)
	}

	if _, ok := resp["error"]; !ok {
		t.Error("Expected error field in response")
	}
}

// TestCreateSecret_AlreadyExists tests CreateSecret with duplicate secret.
func TestCreateSecret_AlreadyExists(t *testing.T) {
	backend := NewMockBackend()
	backend.CreateItem(nil, "existing-secret", "existing-value", nil)

	handler := NewSecretHandler(backend, nil)

	reqBody := CreateSecretRequest{
		Name:  "existing-secret",
		Value: "new-value",
	}
	body, _ := json.Marshal(reqBody)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodPost, "/secrets", bytes.NewReader(body))
	c.Request.Header.Set("Content-Type", "application/json")

	handler.CreateSecret(c)

	if w.Code != http.StatusConflict {
		t.Errorf("Expected status %d, got %d", http.StatusConflict, w.Code)
	}

	var resp map[string]interface{}
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("Failed to unmarshal response: %v", err)
	}

	if resp["error"] != "secret already exists" {
		t.Errorf("Expected error %q, got %q", "secret already exists", resp["error"])
	}
}

// TestCreateSecret_BackendError tests CreateSecret with backend failure.
func TestCreateSecret_BackendError(t *testing.T) {
	backend := NewMockBackend()
	backend.SetCreateError(errors.New("backend failure"))

	handler := NewSecretHandler(backend, nil)

	reqBody := CreateSecretRequest{
		Name:  "new-secret",
		Value: "new-value",
	}
	body, _ := json.Marshal(reqBody)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodPost, "/secrets", bytes.NewReader(body))
	c.Request.Header.Set("Content-Type", "application/json")

	handler.CreateSecret(c)

	if w.Code != http.StatusInternalServerError {
		t.Errorf("Expected status %d, got %d", http.StatusInternalServerError, w.Code)
	}

	var resp map[string]interface{}
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("Failed to unmarshal response: %v", err)
	}

	if _, ok := resp["error"]; !ok {
		t.Error("Expected error field in response")
	}
}

// TestCreateSecret_ExistsCheckError tests CreateSecret when ItemExists check fails.
func TestCreateSecret_ExistsCheckError(t *testing.T) {
	backend := NewMockBackend()
	backend.SetExistsError(errors.New("exists check failure"))

	handler := NewSecretHandler(backend, nil)

	reqBody := CreateSecretRequest{
		Name:  "new-secret",
		Value: "new-value",
	}
	body, _ := json.Marshal(reqBody)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodPost, "/secrets", bytes.NewReader(body))
	c.Request.Header.Set("Content-Type", "application/json")

	handler.CreateSecret(c)

	if w.Code != http.StatusInternalServerError {
		t.Errorf("Expected status %d, got %d", http.StatusInternalServerError, w.Code)
	}
}

// TestUpdateSecret_Success tests successful secret update.
func TestUpdateSecret_Success(t *testing.T) {
	backend := NewMockBackend()
	backend.CreateItem(nil, "existing-secret", "old-value", nil)

	handler := NewSecretHandler(backend, nil)

	reqBody := UpdateSecretRequest{
		Value: "updated-value",
	}
	body, _ := json.Marshal(reqBody)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodPut, "/secrets/existing-secret", bytes.NewReader(body))
	c.Request.Header.Set("Content-Type", "application/json")
	c.Params = gin.Params{{Key: "name", Value: "existing-secret"}}

	handler.UpdateSecret(c)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status %d, got %d", http.StatusOK, w.Code)
	}

	var resp SecretResponse
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("Failed to unmarshal response: %v", err)
	}

	if resp.Name != "existing-secret" {
		t.Errorf("Expected name %q, got %q", "existing-secret", resp.Name)
	}

	// Verify secret was actually updated
	value, _ := backend.GetNotes(nil, "existing-secret", nil)
	if value != "updated-value" {
		t.Errorf("Expected value %q, got %q", "updated-value", value)
	}
}

// TestUpdateSecret_MissingValue tests UpdateSecret with missing value field.
func TestUpdateSecret_MissingValue(t *testing.T) {
	backend := NewMockBackend()
	handler := NewSecretHandler(backend, nil)

	reqBody := map[string]interface{}{}
	body, _ := json.Marshal(reqBody)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodPut, "/secrets/test-secret", bytes.NewReader(body))
	c.Request.Header.Set("Content-Type", "application/json")
	c.Params = gin.Params{{Key: "name", Value: "test-secret"}}

	handler.UpdateSecret(c)

	if w.Code != http.StatusBadRequest {
		t.Errorf("Expected status %d, got %d", http.StatusBadRequest, w.Code)
	}

	var resp map[string]interface{}
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("Failed to unmarshal response: %v", err)
	}

	if _, ok := resp["error"]; !ok {
		t.Error("Expected error field in response")
	}
}

// TestUpdateSecret_NotFound tests UpdateSecret with non-existent secret.
func TestUpdateSecret_NotFound(t *testing.T) {
	backend := NewMockBackend()
	handler := NewSecretHandler(backend, nil)

	reqBody := UpdateSecretRequest{
		Value: "new-value",
	}
	body, _ := json.Marshal(reqBody)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodPut, "/secrets/nonexistent", bytes.NewReader(body))
	c.Request.Header.Set("Content-Type", "application/json")
	c.Params = gin.Params{{Key: "name", Value: "nonexistent"}}

	handler.UpdateSecret(c)

	if w.Code != http.StatusNotFound {
		t.Errorf("Expected status %d, got %d", http.StatusNotFound, w.Code)
	}

	var resp map[string]interface{}
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("Failed to unmarshal response: %v", err)
	}

	if resp["error"] != "secret not found" {
		t.Errorf("Expected error %q, got %q", "secret not found", resp["error"])
	}
}

// TestUpdateSecret_BackendError tests UpdateSecret with backend failure.
func TestUpdateSecret_BackendError(t *testing.T) {
	backend := NewMockBackend()
	backend.CreateItem(nil, "existing-secret", "old-value", nil)
	backend.SetUpdateError(errors.New("backend failure"))

	handler := NewSecretHandler(backend, nil)

	reqBody := UpdateSecretRequest{
		Value: "new-value",
	}
	body, _ := json.Marshal(reqBody)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodPut, "/secrets/existing-secret", bytes.NewReader(body))
	c.Request.Header.Set("Content-Type", "application/json")
	c.Params = gin.Params{{Key: "name", Value: "existing-secret"}}

	handler.UpdateSecret(c)

	if w.Code != http.StatusInternalServerError {
		t.Errorf("Expected status %d, got %d", http.StatusInternalServerError, w.Code)
	}

	var resp map[string]interface{}
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("Failed to unmarshal response: %v", err)
	}

	if _, ok := resp["error"]; !ok {
		t.Error("Expected error field in response")
	}
}

// TestUpdateSecret_ExistsCheckError tests UpdateSecret when ItemExists check fails.
func TestUpdateSecret_ExistsCheckError(t *testing.T) {
	backend := NewMockBackend()
	backend.SetExistsError(errors.New("exists check failure"))

	handler := NewSecretHandler(backend, nil)

	reqBody := UpdateSecretRequest{
		Value: "new-value",
	}
	body, _ := json.Marshal(reqBody)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodPut, "/secrets/test-secret", bytes.NewReader(body))
	c.Request.Header.Set("Content-Type", "application/json")
	c.Params = gin.Params{{Key: "name", Value: "test-secret"}}

	handler.UpdateSecret(c)

	if w.Code != http.StatusInternalServerError {
		t.Errorf("Expected status %d, got %d", http.StatusInternalServerError, w.Code)
	}
}

// TestDeleteSecret_Success tests successful secret deletion.
func TestDeleteSecret_Success(t *testing.T) {
	backend := NewMockBackend()
	backend.CreateItem(nil, "existing-secret", "value", nil)

	handler := NewSecretHandler(backend, nil)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodDelete, "/secrets/existing-secret", nil)
	c.Params = gin.Params{{Key: "name", Value: "existing-secret"}}

	handler.DeleteSecret(c)

	if w.Code != http.StatusNoContent {
		t.Errorf("Expected status %d, got %d", http.StatusNoContent, w.Code)
	}

	// Verify secret was actually deleted
	exists, _ := backend.ItemExists(nil, "existing-secret", nil)
	if exists {
		t.Error("Secret still exists after deletion")
	}
}

// TestDeleteSecret_NotFound tests DeleteSecret with non-existent secret.
func TestDeleteSecret_NotFound(t *testing.T) {
	backend := NewMockBackend()
	handler := NewSecretHandler(backend, nil)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodDelete, "/secrets/nonexistent", nil)
	c.Params = gin.Params{{Key: "name", Value: "nonexistent"}}

	handler.DeleteSecret(c)

	if w.Code != http.StatusNotFound {
		t.Errorf("Expected status %d, got %d", http.StatusNotFound, w.Code)
	}

	var resp map[string]interface{}
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("Failed to unmarshal response: %v", err)
	}

	if resp["error"] != "secret not found" {
		t.Errorf("Expected error %q, got %q", "secret not found", resp["error"])
	}
}

// TestDeleteSecret_BackendError tests DeleteSecret with backend failure.
func TestDeleteSecret_BackendError(t *testing.T) {
	backend := NewMockBackend()
	backend.CreateItem(nil, "existing-secret", "value", nil)
	backend.SetDeleteError(errors.New("backend failure"))

	handler := NewSecretHandler(backend, nil)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodDelete, "/secrets/existing-secret", nil)
	c.Params = gin.Params{{Key: "name", Value: "existing-secret"}}

	handler.DeleteSecret(c)

	if w.Code != http.StatusInternalServerError {
		t.Errorf("Expected status %d, got %d", http.StatusInternalServerError, w.Code)
	}

	var resp map[string]interface{}
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("Failed to unmarshal response: %v", err)
	}

	if _, ok := resp["error"]; !ok {
		t.Error("Expected error field in response")
	}
}

// TestDeleteSecret_ExistsCheckError tests DeleteSecret when ItemExists check fails.
func TestDeleteSecret_ExistsCheckError(t *testing.T) {
	backend := NewMockBackend()
	backend.SetExistsError(errors.New("exists check failure"))

	handler := NewSecretHandler(backend, nil)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodDelete, "/secrets/test-secret", nil)
	c.Params = gin.Params{{Key: "name", Value: "test-secret"}}

	handler.DeleteSecret(c)

	if w.Code != http.StatusInternalServerError {
		t.Errorf("Expected status %d, got %d", http.StatusInternalServerError, w.Code)
	}
}
