package handlers

import (
	"testing"
)

func TestNewSecretHandler(t *testing.T) {
	handler := NewSecretHandler(nil, nil)
	if handler == nil {
		t.Fatal("NewSecretHandler returned nil")
	}
	if handler.backend != nil {
		t.Error("Expected nil backend")
	}
	if handler.session != nil {
		t.Error("Expected nil session")
	}
}

func TestCreateSecretRequest(t *testing.T) {
	req := CreateSecretRequest{
		Name:  "test-secret",
		Value: "test-value",
	}
	
	if req.Name != "test-secret" {
		t.Errorf("Expected name 'test-secret', got %s", req.Name)
	}
	if req.Value != "test-value" {
		t.Errorf("Expected value 'test-value', got %s", req.Value)
	}
}

func TestUpdateSecretRequest(t *testing.T) {
	req := UpdateSecretRequest{
		Value: "new-value",
	}
	
	if req.Value != "new-value" {
		t.Errorf("Expected value 'new-value', got %s", req.Value)
	}
}

func TestSecretResponse(t *testing.T) {
	resp := SecretResponse{
		Name:  "test",
		Value: "value",
	}
	
	if resp.Name != "test" {
		t.Errorf("Expected name 'test', got %s", resp.Name)
	}
	if resp.Value != "value" {
		t.Errorf("Expected value 'value', got %s", resp.Value)
	}
}

func TestListSecretsResponse(t *testing.T) {
	resp := ListSecretsResponse{
		Secrets: []string{"secret1", "secret2"},
	}
	
	if len(resp.Secrets) != 2 {
		t.Errorf("Expected 2 secrets, got %d", len(resp.Secrets))
	}
	if resp.Secrets[0] != "secret1" {
		t.Errorf("Expected first secret 'secret1', got %s", resp.Secrets[0])
	}
}
