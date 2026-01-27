package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/blackwell-systems/vaultmux"
	_ "github.com/blackwell-systems/vaultmux/backends/awssecrets"
	_ "github.com/blackwell-systems/vaultmux/backends/azurekeyvault"
	_ "github.com/blackwell-systems/vaultmux/backends/gcpsecrets"
	"github.com/blackwell-systems/vaultmux-server/handlers"
	"github.com/blackwell-systems/vaultmux-server/middleware"
	"github.com/gin-gonic/gin"
)

func main() {
	port := getEnv("PORT", "8080")
	backendType := getEnv("VAULTMUX_BACKEND", "")
	
	if backendType == "" {
		log.Fatal("VAULTMUX_BACKEND environment variable is required (aws, gcp, or azure)")
	}
	
	supportedBackends := map[string]bool{
		"awssecrets":   true,
		"gcpsecrets":   true,
		"azurekeyvault": true,
	}
	
	if !supportedBackends[backendType] {
		log.Fatalf("Unsupported backend: %s. vaultmux-server supports: awssecrets, gcpsecrets, azurekeyvault. For other backends (pass, bitwarden, 1password), use the vaultmux library directly.", backendType)
	}
	
	backend, err := vaultmux.New(vaultmux.Config{
		Backend: vaultmux.BackendType(backendType),
		Prefix:  getEnv("VAULTMUX_PREFIX", "vaultmux"),
		Options: getBackendOptions(),
	})
	if err != nil {
		log.Fatalf("Failed to create vaultmux backend: %v", err)
	}
	defer backend.Close()

	ctx := context.Background()
	if err := backend.Init(ctx); err != nil {
		log.Fatalf("Failed to initialize backend: %v", err)
	}

	session, err := backend.Authenticate(ctx)
	if err != nil {
		log.Fatalf("Failed to authenticate: %v", err)
	}

	r := gin.Default()
	r.Use(middleware.Logger())
	r.Use(middleware.Recovery())

	handler := handlers.NewSecretHandler(backend, session)

	api := r.Group("/v1")
	{
		secrets := api.Group("/secrets")
		{
			secrets.GET("", handler.ListSecrets)
			secrets.GET("/:name", handler.GetSecret)
			secrets.POST("", handler.CreateSecret)
			secrets.PUT("/:name", handler.UpdateSecret)
			secrets.DELETE("/:name", handler.DeleteSecret)
		}
	}

	r.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{
			"status":  "healthy",
			"backend": backendType,
		})
	})

	srv := &http.Server{
		Addr:    ":" + port,
		Handler: r,
	}

	go func() {
		log.Printf("Starting vaultmux-server on :%s (backend: %s)", port, backendType)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("Failed to start server: %v", err)
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	log.Println("Shutting down server...")
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	
	if err := srv.Shutdown(ctx); err != nil {
		log.Fatalf("Server forced to shutdown: %v", err)
	}

	log.Println("Server exited")
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

func getBackendOptions() map[string]string {
	opts := make(map[string]string)
	
	if region := os.Getenv("AWS_REGION"); region != "" {
		opts["region"] = region
	}
	if endpoint := os.Getenv("AWS_ENDPOINT"); endpoint != "" {
		opts["endpoint"] = endpoint
	}
	if projectID := os.Getenv("GCP_PROJECT_ID"); projectID != "" {
		opts["project_id"] = projectID
	}
	
	return opts
}
