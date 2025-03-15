package main

import (
	"postService/internal/bootstrap"

	"postService/pkg/logging"

	"github.com/gin-gonic/gin"
	"postService/internal/routes"
)

// Initialize logrus as the main logger
var logger = logging.GetLogger()

func main() {

	bs, err := bootstrap.Init()
	if err != nil {
		logger.Errorf("bootstrap init err: %v", err)
	}

	go bs.Consumer.Start()
	defer bs.Consumer.Close()

	r := gin.New()
	gin.SetMode(gin.ReleaseMode)
	gin.DefaultWriter = logger.Out

	r.Use(gin.Recovery())
	r.Use(logging.Middleware)

	routes.SetupRoutes(r, bs)

	logger.Infof("🚀 Server is running on port %s", bs.Config.Port)
	r.Run(":" + bs.Config.Port)
}
