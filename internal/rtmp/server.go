package rtmp

import (
	"fmt"
	"io"
	"log"
	"net"
	"sync"

	"github.com/yutopp/go-rtmp"
	rtmpmsg "github.com/yutopp/go-rtmp/message"

	"rapidrtmp/internal/auth"
	"rapidrtmp/internal/muxer"
	"rapidrtmp/internal/segmenter"
	"rapidrtmp/internal/streammanager"
	"rapidrtmp/pkg/models"
)

// Server represents the RTMP server
type Server struct {
	addr          string
	streamManager *streammanager.Manager
	authManager   *auth.Manager
	segmenter     *segmenter.Segmenter
	server        *rtmp.Server
	mu            sync.RWMutex
}

// New creates a new RTMP server
func New(addr string, streamManager *streammanager.Manager, authManager *auth.Manager, seg *segmenter.Segmenter) *Server {
	s := &Server{
		addr:          addr,
		streamManager: streamManager,
		authManager:   authManager,
		segmenter:     seg,
	}

	// Create RTMP server with handler
	s.server = rtmp.NewServer(&rtmp.ServerConfig{
		OnConnect: s.onConnect,
	})

	return s
}

// ListenAndServe starts the RTMP server
func (s *Server) ListenAndServe() error {
	log.Printf("Starting RTMP server on %s", s.addr)

	listener, err := net.Listen("tcp", s.addr)
	if err != nil {
		return fmt.Errorf("failed to listen on %s: %w", s.addr, err)
	}

	log.Printf("RTMP server listening on %s", s.addr)

	return s.server.Serve(listener)
}

// onConnect handles new RTMP connections
func (s *Server) onConnect(conn net.Conn) (io.ReadWriteCloser, *rtmp.ConnConfig) {
	log.Printf("New RTMP connection from %s", conn.RemoteAddr())

	handler := &ConnHandler{
		server:        s,
		streamManager: s.streamManager,
		authManager:   s.authManager,
		segmenter:     s.segmenter,
		conn:          conn,
	}

	return conn, &rtmp.ConnConfig{
		Handler: handler,

		ControlState: rtmp.StreamControlStateConfig{
			DefaultBandwidthWindowSize: 6 * 1024 * 1024, // 6MB
		},
	}
}

// Close gracefully shuts down the RTMP server
func (s *Server) Close() error {
	if s.server != nil {
		return s.server.Close()
	}
	return nil
}

// ConnHandler handles RTMP connection events
type ConnHandler struct {
	rtmp.DefaultHandler

	server        *Server
	streamManager *streammanager.Manager
	authManager   *auth.Manager
	segmenter     *segmenter.Segmenter
	conn          net.Conn
	streamKey     string
	stream        *models.Stream
	publishToken  string
	sps           [][]byte // H.264 Sequence Parameter Sets
	pps           [][]byte // H.264 Picture Parameter Sets
	naluLength    int      // NALU length size from AVCC
	mu            sync.RWMutex
}

// OnServe is called when the connection starts serving
func (h *ConnHandler) OnServe(conn *rtmp.Conn) {
	log.Printf("Connection started serving")
}

// OnConnect is called when RTMP connect command is received
func (h *ConnHandler) OnConnect(timestamp uint32, cmd *rtmpmsg.NetConnectionConnect) error {
	log.Printf("OnConnect: app=%s, tcUrl=%s", cmd.Command.App, cmd.Command.TCURL)

	// Extract app name (stream path)
	// The app is typically the path after the domain, e.g., "live" in rtmp://server/live/streamkey
	return nil
}

// OnCreateStream is called when createStream command is received
func (h *ConnHandler) OnCreateStream(timestamp uint32, cmd *rtmpmsg.NetConnectionCreateStream) error {
	log.Printf("OnCreateStream called")
	return nil
}

// OnPublish is called when a client wants to publish a stream
func (h *ConnHandler) OnPublish(ctx *rtmp.StreamContext, timestamp uint32, cmd *rtmpmsg.NetStreamPublish) error {
	log.Printf("OnPublish: publishingName=%s, publishingType=%s", cmd.PublishingName, cmd.PublishingType)

	h.mu.Lock()
	defer h.mu.Unlock()

	// Parse stream key and token from publishing name
	// Format: "streamkey?token=xxx" or just "streamkey"
	streamKey, token := parseStreamKeyAndToken(cmd.PublishingName)
	h.streamKey = streamKey
	h.publishToken = token

	// Validate token if provided
	if token != "" {
		clientIP := h.conn.RemoteAddr().String()
		if err := h.authManager.ValidateToken(token, streamKey, clientIP); err != nil {
			log.Printf("Token validation failed for stream %s: %v", streamKey, err)
			return fmt.Errorf("authentication failed: %w", err)
		}

		// Mark token as used
		h.authManager.MarkTokenUsed(token)
		log.Printf("Token validated successfully for stream %s", streamKey)
	} else {
		log.Printf("Warning: No token provided for stream %s", streamKey)
		// For now, allow publishing without token for testing
		// In production, you should enforce token validation
	}

	// Create or get stream in stream manager
	clientIP := h.conn.RemoteAddr().String()
	stream, err := h.streamManager.CreateStream(streamKey, clientIP)
	if err != nil {
		log.Printf("Failed to create stream %s: %v", streamKey, err)
		return err
	}

	h.stream = stream
	stream.SetState(models.StreamStateLive)

	// Start HLS segmentation for this stream
	if h.segmenter != nil {
		if err := h.segmenter.StartSegmenting(streamKey); err != nil {
			log.Printf("Failed to start segmentation for stream %s: %v", streamKey, err)
		} else {
			log.Printf("Started HLS segmentation for stream %s", streamKey)
		}
	}

	log.Printf("Stream %s is now live from %s", streamKey, clientIP)

	return nil
}

// OnSetDataFrame is called when metadata is received
func (h *ConnHandler) OnSetDataFrame(timestamp uint32, data *rtmpmsg.NetStreamSetDataFrame) error {
	log.Printf("OnSetDataFrame received")

	h.mu.Lock()
	defer h.mu.Unlock()

	if h.stream != nil {
		// The payload structure in go-rtmp is complex, so we'll extract what we can
		// For now, just log that we received metadata
		log.Printf("Received metadata for stream %s", h.streamKey)
	}

	return nil
}

// OnAudio is called when audio data is received
func (h *ConnHandler) OnAudio(timestamp uint32, payload io.Reader) error {
	h.mu.RLock()
	stream := h.stream
	streamKey := h.streamKey
	h.mu.RUnlock()

	if stream == nil {
		return nil // Ignore audio before stream is created
	}

	// Read raw audio data
	audioData := make([]byte, 4096) // Buffer for audio data
	n, err := payload.Read(audioData)
	if err != nil && err != io.EOF {
		return err
	}

	if n > 0 {
		// Create frame and publish to stream manager
		frame := &models.Frame{
			StreamKey:  streamKey,
			IsVideo:    false,
			Timestamp:  timestamp,
			Payload:    audioData[:n],
			Codec:      "aac", // Assume AAC for now
			IsKeyFrame: false,
		}

		// Publish frame to subscribers
		if err := h.streamManager.PublishFrame(frame); err != nil {
			log.Printf("Failed to publish audio frame: %v", err)
		}
	}

	return nil
}

// OnVideo is called when video data is received
func (h *ConnHandler) OnVideo(timestamp uint32, payload io.Reader) error {
	h.mu.RLock()
	stream := h.stream
	streamKey := h.streamKey
	h.mu.RUnlock()

	if stream == nil {
		return nil // Ignore video before stream is created
	}

	// Read raw video data
	videoData := make([]byte, 65536) // Larger buffer for video data
	n, err := payload.Read(videoData)
	if err != nil && err != io.EOF {
		return err
	}

	if n == 0 {
		return nil
	}

	// Parse FLV video packet
	isSequenceHeader, isKeyFrame, avcData, err := muxer.ParseFLVVideoPacket(videoData[:n])
	if err != nil {
		log.Printf("Failed to parse FLV video packet: %v", err)
		return nil // Don't fail, just skip this packet
	}

	// Handle AVC sequence header (contains SPS/PPS)
	if isSequenceHeader {
		log.Printf("Received AVC sequence header for stream %s (%d bytes)", streamKey, len(avcData))

		// Parse AVCDecoderConfigurationRecord to extract SPS/PPS
		avcConfig, err := muxer.ParseAVCDecoderConfigurationRecord(avcData)
		if err != nil {
			log.Printf("Failed to parse AVCDecoderConfigurationRecord: %v", err)
			return nil
		}

		// Store SPS/PPS for later use
		h.mu.Lock()
		h.sps = avcConfig.SPS
		h.pps = avcConfig.PPS
		h.naluLength = int(avcConfig.NALUnitLength)
		h.mu.Unlock()

		log.Printf("Stored SPS/PPS for stream %s: %d SPS, %d PPS, NALU length=%d",
			streamKey, len(avcConfig.SPS), len(avcConfig.PPS), avcConfig.NALUnitLength)

		// Don't send sequence header as a frame, it's just configuration
		return nil
	}

	// Convert AVCC to Annex-B
	annexBData, err := muxer.ConvertAVCCToAnnexB(avcData)
	if err != nil {
		log.Printf("Failed to convert AVCC to Annex-B: %v", err)
		// Fall back to original data
		annexBData = avcData
	}

	// For keyframes, prepend SPS/PPS
	var frameData []byte
	if isKeyFrame {
		h.mu.RLock()
		sps := h.sps
		pps := h.pps
		h.mu.RUnlock()

		if len(sps) > 0 && len(pps) > 0 {
			// Prepend SPS/PPS to keyframe
			frameData = muxer.PrependSPSPPSAnnexB(annexBData, sps, pps)
			log.Printf("Prepended SPS/PPS to keyframe for stream %s (total size: %d bytes)", streamKey, len(frameData))
		} else {
			log.Printf("Warning: Keyframe received but no SPS/PPS stored for stream %s", streamKey)
			frameData = annexBData
		}
	} else {
		frameData = annexBData
	}

	// Create frame and publish to stream manager
	frame := &models.Frame{
		StreamKey:  streamKey,
		IsVideo:    true,
		Timestamp:  timestamp,
		Payload:    frameData,
		Codec:      "h264",
		IsKeyFrame: isKeyFrame,
	}

	// Publish frame to subscribers
	if err := h.streamManager.PublishFrame(frame); err != nil {
		log.Printf("Failed to publish video frame: %v", err)
	}

	return nil
}

// OnClose is called when the connection is closed
func (h *ConnHandler) OnClose() {
	log.Printf("Connection closed: %s", h.conn.RemoteAddr())

	h.mu.Lock()
	defer h.mu.Unlock()

	if h.stream != nil && h.streamKey != "" {
		log.Printf("Stopping stream %s", h.streamKey)

		// Stop segmentation
		if h.segmenter != nil {
			h.segmenter.StopSegmenting(h.streamKey)
		}

		h.streamManager.StopStream(h.streamKey)
	}
}

// Helper functions

func parseStreamKeyAndToken(publishingName string) (streamKey, token string) {
	// Parse format: "streamkey?token=xxx"
	// Find the '?' separator
	for i, c := range publishingName {
		if c == '?' {
			streamKey = publishingName[:i]
			// Parse query string for token
			query := publishingName[i+1:]
			// Simple parsing: look for "token="
			if len(query) > 6 && query[:6] == "token=" {
				token = query[6:]
			}
			return
		}
	}

	// No token, just stream key
	streamKey = publishingName
	return
}
