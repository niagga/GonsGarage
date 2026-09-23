package main

import (
	"net/http"

	"github.com/gin-gonic/gin"

	fiscalsvc "github.com/gaston-garcia-cegid/gonsgarage/internal/service/fiscal"
)

func validateAPIFiscalComposition(cfg fiscalsvc.RuntimeConfig) error {
	return cfg.ValidateProductionComposition()
}

func registerCoreReadyRoute(router *gin.Engine, pingDB func() error, fiscalStatus func() fiscalsvc.FiscalDependencyStatus) {
	router.GET("/ready", func(c *gin.Context) {
		dbOK := pingDB() == nil
		var providerHealthy bool
		if fiscalStatus != nil {
			providerHealthy = fiscalStatus().ProviderHealthy
		}
		if !fiscalsvc.CoreAPIReady(dbOK, providerHealthy) {
			c.JSON(http.StatusServiceUnavailable, gin.H{"status": "not_ready"})
			return
		}
		c.JSON(http.StatusOK, gin.H{"status": "ready"})
	})
}

func registerFiscalDependencyRoute(router *gin.Engine, fiscalStatus func() fiscalsvc.FiscalDependencyStatus) {
	router.GET("/fiscal/dependency-status", func(c *gin.Context) {
		status := fiscalStatus()
		c.JSON(http.StatusOK, status)
	})
}
