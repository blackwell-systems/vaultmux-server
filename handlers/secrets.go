package handlers

import (
	"net/http"

	"github.com/blackwell-systems/vaultmux"
	"github.com/gin-gonic/gin"
)

type SecretHandler struct {
	backend vaultmux.Backend
	session vaultmux.Session
}

func NewSecretHandler(backend vaultmux.Backend, session vaultmux.Session) *SecretHandler {
	return &SecretHandler{
		backend: backend,
		session: session,
	}
}

type CreateSecretRequest struct {
	Name  string `json:"name" binding:"required"`
	Value string `json:"value" binding:"required"`
}

type UpdateSecretRequest struct {
	Value string `json:"value" binding:"required"`
}

type SecretResponse struct {
	Name  string `json:"name"`
	Value string `json:"value,omitempty"`
}

type ListSecretsResponse struct {
	Secrets []string `json:"secrets"`
}

func (h *SecretHandler) ListSecrets(c *gin.Context) {
	items, err := h.backend.ListItems(c.Request.Context(), h.session)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	names := make([]string, len(items))
	for i, item := range items {
		names[i] = item.Name
	}

	c.JSON(http.StatusOK, ListSecretsResponse{Secrets: names})
}

func (h *SecretHandler) GetSecret(c *gin.Context) {
	name := c.Param("name")

	value, err := h.backend.GetNotes(c.Request.Context(), name, h.session)
	if err != nil {
		if err == vaultmux.ErrNotFound {
			c.JSON(http.StatusNotFound, gin.H{"error": "secret not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, SecretResponse{
		Name:  name,
		Value: value,
	})
}

func (h *SecretHandler) CreateSecret(c *gin.Context) {
	var req CreateSecretRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	exists, err := h.backend.ItemExists(c.Request.Context(), req.Name, h.session)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	if exists {
		c.JSON(http.StatusConflict, gin.H{"error": "secret already exists"})
		return
	}

	if err := h.backend.CreateItem(c.Request.Context(), req.Name, req.Value, h.session); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, SecretResponse{Name: req.Name})
}

func (h *SecretHandler) UpdateSecret(c *gin.Context) {
	name := c.Param("name")

	var req UpdateSecretRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	exists, err := h.backend.ItemExists(c.Request.Context(), name, h.session)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	if !exists {
		c.JSON(http.StatusNotFound, gin.H{"error": "secret not found"})
		return
	}

	if err := h.backend.UpdateItem(c.Request.Context(), name, req.Value, h.session); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, SecretResponse{Name: name})
}

func (h *SecretHandler) DeleteSecret(c *gin.Context) {
	name := c.Param("name")

	exists, err := h.backend.ItemExists(c.Request.Context(), name, h.session)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	if !exists {
		c.JSON(http.StatusNotFound, gin.H{"error": "secret not found"})
		return
	}

	if err := h.backend.DeleteItem(c.Request.Context(), name, h.session); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusNoContent, nil)
}
