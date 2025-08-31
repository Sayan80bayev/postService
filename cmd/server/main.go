package main

import (
	"context"
	"github.com/Sayan80bayev/go-project/pkg/logging"
	"postService/internal/bootstrap"

	"github.com/gin-gonic/gin"
	"postService/internal/routes"
)

// Initialize logrus as the main logger
var logger = logging.GetLogger()

func main() {
	ctx := context.Background()
	c, err := bootstrap.Init()
	if err != nil {
		logger.Errorf("bootstrap init err: %v", err)
	}

	go c.Consumer.Start(ctx)
	defer c.Consumer.Close()

	gin.SetMode(gin.ReleaseMode)
	gin.DefaultWriter = logger.Out
	r := gin.New()

	r.Use(gin.Recovery())
	r.Use(logging.Middleware)

	routes.SetupRoutes(r, c)

	logger.Infof("🚀 Server is running on port %s", c.Config.Port)
	err = r.Run(":" + c.Config.Port)
	if err != nil {
		logger.Errorf("Error starting server: %v", err)
	}
}
