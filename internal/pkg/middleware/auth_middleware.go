package middleware

import (
	"net/http"
	"postService/internal/model"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
)

func AuthMiddleware(jwtKey string) gin.HandlerFunc {
	return func(c *gin.Context) {
		authHeader := c.GetHeader("Authorization")
		if authHeader == "" {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Authorization header missing"})
			c.Abort()
			return
		}

		tokenString := strings.TrimPrefix(authHeader, "Bearer ")
		claims := &jwt.MapClaims{}

		token, err := jwt.ParseWithClaims(tokenString, claims, func(token *jwt.Token) (interface{}, error) {
			return []byte(jwtKey), nil
		})

		if err != nil || !token.Valid {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid token"})
			c.Abort()
			return
		}

		userID, okID := (*claims)["user_id"].(float64)
		userRole, okROLE := (*claims)["user_role"].(string)
		userActive, okActive := (*claims)["user_active"].(bool)
		if !okID || !okROLE || !okActive {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid token data"})
			c.Abort()
			return
		}
		c.Set("user_role", model.Role(userRole))
		c.Set("user_id", int(userID))
		c.Set("user_active", userActive)
		c.Next()
	}
}

func AdminMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		// Retrieve user role from context
		userRole, exists := c.Get("user_role")
		if !exists {
			msg := "Could not find user role"
			c.JSON(http.StatusUnauthorized, gin.H{"error": msg})
			c.Abort() // Prevent further request handling
			return
		}

		// Type assertion to ensure it's a string (or whatever type your roles use)
		role, ok := userRole.(model.Role)
		if !ok || role != model.RoleAdmin {
			msg := "You do not have access to this resource"
			c.JSON(http.StatusForbidden, gin.H{"error": msg})
			c.Abort()
			return
		}

		// Continue processing the request
		c.Next()
	}
}

func ModeratorMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		userRole, exists := c.Get("user_role")
		if !exists {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Could not find user role"})
			c.Abort()
			return
		}

		role, ok := userRole.(model.Role)
		if !ok || (role != model.RoleModerator && role != model.RoleAdmin) {
			c.JSON(http.StatusForbidden, gin.H{"error": "You do not have access to this resource"})
			c.Abort()
			return
		}

		c.Next()
	}
}

func ActiveMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		userActive, exists := c.Get("user_active")
		if !exists {
			msg := "Could not find user active"
			c.JSON(http.StatusUnauthorized, gin.H{"error": msg})
			c.Abort()
			return
		}

		if !userActive.(bool) {
			msg := "You are banned and cannot access this resource"
			c.JSON(http.StatusForbidden, gin.H{"error": msg})
			c.Abort()
			return
		}

		c.Next()
	}
}
