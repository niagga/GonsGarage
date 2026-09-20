package middleware

import (
	"net/http"

	"github.com/gaston-garcia-cegid/gonsgarage/internal/domain"
	"github.com/gin-gonic/gin"
)

// RequireStaffManagers allows only admin or manager (backoffice staff management).
func RequireStaffManagers() gin.HandlerFunc {
	return func(c *gin.Context) {
		v, ok := c.Get("userRole")
		if !ok {
			c.JSON(http.StatusForbidden, gin.H{"error": "forbidden"})
			c.Abort()
			return
		}
		role, ok := v.(string)
		if !ok || (role != domain.RoleAdmin && role != domain.RoleManager) {
			c.JSON(http.StatusForbidden, gin.H{"error": "forbidden"})
			c.Abort()
			return
		}
		c.Next()
	}
}

// RequireWorkshopStaff allows admin, manager, or employee (taller); blocks clients.
func RequireWorkshopStaff() gin.HandlerFunc {
	return func(c *gin.Context) {
		v, ok := c.Get("userRole")
		if !ok {
			c.JSON(http.StatusForbidden, gin.H{"error": "forbidden"})
			c.Abort()
			return
		}
		role, ok := v.(string)
		if !ok {
			c.JSON(http.StatusForbidden, gin.H{"error": "forbidden"})
			c.Abort()
			return
		}
		if role != domain.RoleAdmin && role != domain.RoleManager && role != domain.RoleEmployee {
			c.JSON(http.StatusForbidden, gin.H{"error": "forbidden"})
			c.Abort()
			return
		}
		c.Next()
	}
}

// RequireAccountingAccess allows only admin or manager (accounting/staff management).
func RequireAccountingAccess() gin.HandlerFunc {
	return RequireStaffManagers()
}
