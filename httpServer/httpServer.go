package httpServer

import (
	"fmt"
	"net/http"
	"strconv"
	"time"

	"rapidrtmp/internal/auth"
	"rapidrtmp/internal/metrics"
	"rapidrtmp/internal/segmenter"
	"rapidrtmp/internal/streammanager"
	"rapidrtmp/pkg/models"

	"github.com/gin-gonic/gin"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

// Server wraps the HTTP server with dependencies
type Server struct {
	router         *gin.Engine
	streamManager  *streammanager.Manager
	authManager    *auth.Manager
	segmenter      *segmenter.Segmenter
	metrics        *metrics.Metrics
	rtmpIngestAddr string // e.g., "rtmp://localhost:1935"
}

// New creates a new HTTP server
func New(streamManager *streammanager.Manager, authManager *auth.Manager, seg *segmenter.Segmenter, m *metrics.Metrics, rtmpIngestAddr string) *Server {
	s := &Server{
		streamManager:  streamManager,
		authManager:    authManager,
		segmenter:      seg,
		metrics:        m,
		rtmpIngestAddr: rtmpIngestAddr,
	}

	s.setupRoutes()
	return s
}

// setupRoutes configures all HTTP routes
func (s *Server) setupRoutes() {
	router := gin.Default()

	// Add metrics middleware
	router.Use(s.metricsMiddleware())

	// Observability endpoints
	router.GET("/metrics", gin.WrapH(promhttp.Handler()))
	router.GET("/health", s.handleHealth)
	router.GET("/ready", s.handleReady)

	api := router.Group("/api")
	{
		api.GET("/ping", s.handlePing)
		api.POST("/v1/publish", s.handlePublish)
		api.GET("/v1/streams", s.handleListStreams)
		api.GET("/v1/streams/:streamKey", s.handleGetStream)
		api.POST("/v1/streams/:streamKey/stop", s.handleStopStream)
	}

	live := router.Group("/live/:streamKey")
	{
		live.GET("/index.m3u8", s.handlePlaylist)
		live.HEAD("/index.m3u8", s.handlePlaylist) // respond to HEAD for players that probe
		live.GET("/init.mp4", s.handleInitSegment)
		live.HEAD("/init.mp4", s.handleInitSegment)
		live.GET("/:filename", s.handleMediaSegment)
		live.HEAD("/:filename", s.handleMediaSegment)
	}

	s.router = router
}

// Run starts the HTTP server
func (s *Server) Run(addr string) error {
	return s.router.Run(addr)
}

// Middleware

// metricsMiddleware records HTTP request metrics
func (s *Server) metricsMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		if s.metrics == nil {
			c.Next()
			return
		}

		start := time.Now()
		path := c.FullPath()
		if path == "" {
			path = "unknown"
		}

		c.Next()

		duration := time.Since(start).Seconds()
		status := c.Writer.Status()
		method := c.Request.Method

		s.metrics.RecordHTTPRequest(method, path, status, duration)
	}
}

// Handler implementations

func (s *Server) handleHealth(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"status": "healthy",
		"time":   time.Now().Unix(),
	})
}

func (s *Server) handleReady(c *gin.Context) {
	// Check if critical components are ready
	ready := true
	checks := make(map[string]string)

	// Check stream manager
	if s.streamManager != nil {
		checks["streamManager"] = "ok"
	} else {
		checks["streamManager"] = "not initialized"
		ready = false
	}

	// Check segmenter
	if s.segmenter != nil {
		checks["segmenter"] = "ok"
	} else {
		checks["segmenter"] = "not initialized"
		ready = false
	}

	status := http.StatusOK
	if !ready {
		status = http.StatusServiceUnavailable
	}

	c.JSON(status, gin.H{
		"ready":  ready,
		"checks": checks,
		"time":   time.Now().Unix(),
	})
}

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
	filename := c.Param("filename")

	// Only handle .m4s files
	if len(filename) < 5 || filename[len(filename)-4:] != ".m4s" {
		c.JSON(http.StatusNotFound, gin.H{"error": "not found"})
		return
	}

	// Remove .m4s extension
	filename = filename[:len(filename)-4]

	// Extract segment number from "segment_N"
	if len(filename) < 9 || filename[:8] != "segment_" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid segment format"})
		return
	}

	segmentNumStr := filename[8:]

	// Parse segment number
	segmentNum, err := strconv.ParseUint(segmentNumStr, 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": fmt.Sprintf("invalid segment number: %s", segmentNumStr)})
		return
	}

	// Get segment from segmenter
	segmentData, err := s.segmenter.GetSegment(streamKey, segmentNum)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "segment not found"})
		return
	}

	// If segment starts with full MP4 (ftyp/moov), trim to start at first moof box for fMP4 streaming
	if len(segmentData) >= 12 {
		// look for 'moof' (0x6d6f6f66)
		moofTag := []byte{'m', 'o', 'o', 'f'}
		if !(segmentData[4] == 'f' && segmentData[5] == 't' && segmentData[6] == 'y' && segmentData[7] == 'p') {
			// not starting with ftyp; leave as-is
		} else {
			// scan for moof
			idx := -1
			for i := 8; i <= len(segmentData)-4; i++ {
				if segmentData[i] == moofTag[0] && segmentData[i+1] == moofTag[1] && segmentData[i+2] == moofTag[2] && segmentData[i+3] == moofTag[3] {
					idx = i
					break
				}
			}
			if idx > 4 {
				// slice starting at size field preceding moof
				start := idx - 4
				if start >= 0 && start < len(segmentData) {
					segmentData = segmentData[start:]
				}
			}
		}
	}

	// Live segments: avoid caching to prevent stalling on stale fragments
	c.Header("Cache-Control", "no-cache, no-store, must-revalidate")
	c.Header("Pragma", "no-cache")
	c.Header("Expires", "0")
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
