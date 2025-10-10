package httpServer

import (
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"rapidrtmp/internal/auth"
	"rapidrtmp/internal/segmenter"
	"rapidrtmp/internal/streammanager"
	"rapidrtmp/pkg/models"

	"github.com/gin-gonic/gin"
)

// Server wraps the HTTP server with dependencies
type Server struct {
	router         *gin.Engine
	streamManager  *streammanager.Manager
	authManager    *auth.Manager
	segmenter      *segmenter.Segmenter
	rtmpIngestAddr string // e.g., "rtmp://localhost:1935"
}

// New creates a new HTTP server
func New(streamManager *streammanager.Manager, authManager *auth.Manager, seg *segmenter.Segmenter, rtmpIngestAddr string) *Server {
	s := &Server{
		streamManager:  streamManager,
		authManager:    authManager,
		segmenter:      seg,
		rtmpIngestAddr: rtmpIngestAddr,
	}

	s.setupRoutes()
	return s
}

// setupRoutes configures all HTTP routes
func (s *Server) setupRoutes() {
	router := gin.Default()

	api := router.Group("/api")
	{
		api.GET("/ping", s.handlePing)
		api.POST("/v1/publish", s.handlePublish)
		api.GET("/v1/streams", s.handleListStreams)
		api.GET("/v1/streams/:streamKey", s.handleGetStream)
		api.POST("/v1/streams/:streamKey/stop", s.handleStopStream)
	}

	live := router.Group("/live")
	{
		live.GET("/:streamKey/index.m3u8", s.handlePlaylist)
		live.GET("/:streamKey/init.mp4", s.handleInitSegment)
		live.GET("/:streamKey/:segment.m4s", s.handleMediaSegment)
	}

	s.router = router
}

// Run starts the HTTP server
func (s *Server) Run(addr string) error {
	return s.router.Run(addr)
}

// Handler implementations

func (s *Server) handlePing(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"message": "pong",
		"time":    time.Now().Unix(),
	})
}

func (s *Server) handlePublish(c *gin.Context) {
	var req models.PublishRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Default expiration to 1 hour
	if req.ExpiresIn == 0 {
		req.ExpiresIn = 3600
	}

	// Generate publish token
	clientIP := c.ClientIP()
	token, err := s.authManager.GeneratePublishToken(req.StreamKey, req.ExpiresIn, clientIP)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to generate token"})
		return
	}

	// Build publish URL
	publishURL := fmt.Sprintf("%s/live/%s?token=%s", s.rtmpIngestAddr, req.StreamKey, token.Token)

	c.JSON(http.StatusOK, models.PublishResponse{
		PublishURL: publishURL,
		StreamKey:  req.StreamKey,
		Token:      token.Token,
		ExpiresAt:  token.ExpiresAt.Format(time.RFC3339),
	})
}

func (s *Server) handleListStreams(c *gin.Context) {
	streams := s.streamManager.GetLiveStreams()

	streamInfos := make([]models.StreamInfo, len(streams))
	for i, stream := range streams {
		streamInfos[i] = s.streamToInfo(stream)
	}

	c.JSON(http.StatusOK, models.StreamListResponse{
		Streams: streamInfos,
		Total:   len(streamInfos),
	})
}

func (s *Server) handleGetStream(c *gin.Context) {
	streamKey := c.Param("streamKey")

	stream, exists := s.streamManager.GetStream(streamKey)
	if !exists {
		c.JSON(http.StatusNotFound, gin.H{"error": "stream not found"})
		return
	}

	c.JSON(http.StatusOK, s.streamToInfo(stream))
}

func (s *Server) handleStopStream(c *gin.Context) {
	streamKey := c.Param("streamKey")

	err := s.streamManager.StopStream(streamKey)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message":   "stream stopped",
		"streamKey": streamKey,
	})
}

func (s *Server) handlePlaylist(c *gin.Context) {
	streamKey := c.Param("streamKey")

	stream, exists := s.streamManager.GetStream(streamKey)
	if !exists || stream.GetState() != models.StreamStateLive {
		c.JSON(http.StatusNotFound, gin.H{"error": "stream not found or not live"})
		return
	}

	// Get playlist from segmenter
	playlist, err := s.segmenter.GetPlaylist(streamKey)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "playlist not available"})
		return
	}

	// Set caching headers for low-latency
	c.Header("Cache-Control", "no-cache, no-store, must-revalidate")
	c.Header("Pragma", "no-cache")
	c.Header("Expires", "0")
	c.Header("Access-Control-Allow-Origin", "*")

	c.Data(http.StatusOK, "application/vnd.apple.mpegurl", []byte(playlist))
}

func (s *Server) handleInitSegment(c *gin.Context) {
	streamKey := c.Param("streamKey")

	stream, exists := s.streamManager.GetStream(streamKey)
	if !exists || stream.GetState() != models.StreamStateLive {
		c.JSON(http.StatusNotFound, gin.H{"error": "stream not found or not live"})
		return
	}

	// Get init segment from segmenter
	initData, err := s.segmenter.GetInitSegment(streamKey)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "init segment not available"})
		return
	}

	// Set caching headers (init segment can be cached longer)
	c.Header("Cache-Control", "public, max-age=3600")
	c.Header("Access-Control-Allow-Origin", "*")

	c.Data(http.StatusOK, "video/mp4", initData)
}

func (s *Server) handleMediaSegment(c *gin.Context) {
	streamKey := c.Param("streamKey")
	segmentParam := c.Param("segment")

	stream, exists := s.streamManager.GetStream(streamKey)
	if !exists || stream.GetState() != models.StreamStateLive {
		c.JSON(http.StatusNotFound, gin.H{"error": "stream not found or not live"})
		return
	}

	// Parse segment number from filename (e.g., "segment_5" -> 5)
	segmentNumStr := strings.TrimPrefix(segmentParam, "segment_")
	segmentNum, err := strconv.ParseUint(segmentNumStr, 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid segment number"})
		return
	}

	// Get segment from segmenter
	segmentData, err := s.segmenter.GetSegment(streamKey, segmentNum)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "segment not found"})
		return
	}

	// Set caching headers (segments can be cached)
	c.Header("Cache-Control", "public, max-age=60")
	c.Header("Access-Control-Allow-Origin", "*")

	c.Data(http.StatusOK, "video/mp4", segmentData)
}

// Helper functions

func (s *Server) streamToInfo(stream *models.Stream) models.StreamInfo {
	info := models.StreamInfo{
		StreamKey: stream.Key,
		Active:    stream.GetState() == models.StreamStateLive,
		State:     string(stream.GetState()),
		Viewers:   stream.GetViewerCount(),
		Metadata:  stream.Metadata,
	}

	if !stream.StartedAt.IsZero() {
		info.StartedAt = stream.StartedAt.Format(time.RFC3339)
		info.Duration = int(time.Since(stream.StartedAt).Seconds())
	}

	if stream.VideoCodec != nil {
		info.VideoCodec = stream.VideoCodec.Codec
		info.Resolution = fmt.Sprintf("%dx%d", stream.VideoCodec.Width, stream.VideoCodec.Height)
		info.Bitrate = stream.VideoCodec.Bitrate
	}

	if stream.AudioCodec != nil {
		info.AudioCodec = stream.AudioCodec.Codec
	}

	return info
}

// Legacy function for backward compatibility
func SetupRouter() *gin.Engine {
	// Create default dependencies
	streamManager := streammanager.New()
	authManager := auth.New()
	// Note: segmenter would need storage, which we don't have here
	// This function is mainly for backward compatibility

	server := &Server{
		streamManager:  streamManager,
		authManager:    authManager,
		rtmpIngestAddr: "rtmp://localhost:1935",
	}
	server.setupRoutes()
	return server.router
}
