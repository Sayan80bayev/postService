package main

import (
	"context"
	"postService/internal/bootstrap"
	"postService/internal/metrics"

	"github.com/Sayan80bayev/go-project/pkg/logging"
	"github.com/prometheus/client_golang/prometheus/promhttp"

	"postService/internal/routes"

	"github.com/gin-gonic/gin"
)

var logger = logging.GetLogger()

func main() {
	ctx := context.Background()
	c, err := bootstrap.Init()
	if err != nil {
		logger.Errorf("bootstrap init err: %v", err)
	}

	go c.Consumer.Start(ctx)
	defer c.Consumer.Close()
	metrics.Init()

	gin.SetMode(gin.ReleaseMode)
	gin.DefaultWriter = logger.Out
	r := gin.New()
	r.Use(gin.Recovery())
	r.Use(logging.Middleware)
	r.GET("/metrics", gin.WrapH(promhttp.Handler()))

	routes.SetupRoutes(r, c)

	logger.Infof("Server is running on port %s", c.Config.Port)
	err = r.Run(":" + c.Config.Port)
	if err != nil {
		logger.Errorf("Error starting server: %v", err)
	}
}
