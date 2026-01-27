package middleware

import (
	"testing"

	"github.com/gin-gonic/gin"
)

func TestLogger(t *testing.T) {
	gin.SetMode(gin.TestMode)
	
	middleware := Logger()
	if middleware == nil {
		t.Fatal("Logger() returned nil")
	}
}

func TestRecovery(t *testing.T) {
	gin.SetMode(gin.TestMode)
	
	middleware := Recovery()
	if middleware == nil {
		t.Fatal("Recovery() returned nil")
	}
}
