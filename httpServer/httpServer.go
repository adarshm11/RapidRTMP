package httpServer

import (
	"github.com/gin-gonic/gin"
)

func SetupRouter() *gin.Engine {
	router := gin.Default()

	api := router.Group("/api") // main api endpoints
	{

		api.GET("/ping", func(c *gin.Context) {
			c.JSON(200, gin.H{
				"message": "pong",
			})
		})

		api.POST("/v1/publish", func(c *gin.Context) {
			// request a short-lived publish token
			// request: { "streamKey": "myshow", "expiresIn"; 3600 }
			// response: { "publishUrl": "trmp://ingest.example.com/live/myshow?token=..." }
		})

		api.GET("/v1/streams", func(c *gin.Context) {
			// list live streams
		})

		api.GET("/v1/streams/:streamKey", func(c *gin.Context) {
			// stream metadata
		})

		api.POST("/v1/streams/:streamKey/stop", func(c *gin.Context) {
			// force stop
		})
	}

	live := router.Group("/live") // live streaming endpoints
	{
		live.GET("/:streamKey/index.m3u8", func(c *gin.Context) {
			// serve HLS playlist
		})

		live.GET("/:streamKey/init.mp4", func(c *gin.Context) {
			// serve mp4 init segment
		})

		live.GET("/:streamKey/:segment.m4s", func(c *gin.Context) {
			// serve mp4 media segment
		})
	}

	return router
}
