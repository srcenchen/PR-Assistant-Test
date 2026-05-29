package api

import (
	"net/http"
	"op-status/pkg"
	"time"

	"github.com/gin-gonic/gin"
)

// Handler manages API endpoint handlers
type Handler struct {
	scheduler *pkg.Scheduler
}

// NewHandler creates a new API handler instance
func NewHandler(scheduler *pkg.Scheduler) *Handler {
	return &Handler{
		scheduler: scheduler,
	}
}

// RegisterRoutes registers all API routes with the Gin router
func (h *Handler) RegisterRoutes(router *gin.Engine) {
	// Enable CORS for all requests
	router.Use(corsMiddleware())

	v1 := router.Group("/api/v1")
	{
		v1.GET("/metrics", h.GetMetrics)
		v1.GET("/dhcp", h.GetDHCP)
		v1.GET("/refreshIP", h.RefreshPublicIP)
	}

	router.GET("/health", h.HealthCheck)
}

// GetMetrics returns all system metrics
func (h *Handler) GetMetrics(c *gin.Context) {
	metrics := h.scheduler.GetMetrics()
	c.JSON(http.StatusOK, metrics)
}

// GetDHCP returns DHCP lease information with device type identification
func (h *Handler) GetDHCP(c *gin.Context) {
	dhcpMetrics := h.scheduler.GetDHCPMetrics()
	if dhcpMetrics == nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to get DHCP leases",
		})
		return
	}
	c.JSON(http.StatusOK, dhcpMetrics)
}

// RefreshPublicIP forces an immediate refresh of public IP address
func (h *Handler) RefreshPublicIP(c *gin.Context) {
	result := h.scheduler.RefreshPublicIP()
	c.JSON(http.StatusOK, result)
}

// HealthCheck returns service health status
func (h *Handler) HealthCheck(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"status":    "ok",
		"timestamp": time.Now().Unix(),
	})
}

// corsMiddleware enables CORS support
func corsMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Writer.Header().Set("Access-Control-Allow-Origin", "*")
		c.Writer.Header().Set("Access-Control-Allow-Headers", "Content-Type, Content-Length, Accept-Encoding, X-CSRF-Token, Authorization, accept, origin, Cache-Control, X-Requested-With")
		c.Writer.Header().Set("Access-Control-Allow-Methods", "GET, OPTIONS")

		if c.Request.Method == "OPTIONS" {
			c.AbortWithStatus(http.StatusNoContent)
			return
		}

		c.Next()
	}
}
