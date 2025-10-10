package models

import (
	"sync"
	"time"
)

// StreamState represents the current state of a stream
type StreamState string

const (
	StreamStateIdle       StreamState = "idle"
	StreamStateConnecting StreamState = "connecting"
	StreamStateLive       StreamState = "live"
	StreamStateStopping   StreamState = "stopping"
	StreamStateStopped    StreamState = "stopped"
)

// Stream represents a live stream
type Stream struct {
	Key         string                 // Unique stream key
	State       StreamState            // Current state
	StartedAt   time.Time              // When stream went live
	StoppedAt   *time.Time             // When stream stopped (if stopped)
	PublisherIP string                 // IP of the publisher
	ViewerCount int                    // Current number of viewers
	VideoCodec  *CodecInfo             // Video codec information
	AudioCodec  *CodecInfo             // Audio codec information
	Metadata    map[string]interface{} // Additional metadata (from onMetaData)

	// Stats
	Stats StreamStats

	mu sync.RWMutex // Protects concurrent access
}

// StreamStats tracks stream statistics
type StreamStats struct {
	BytesReceived     uint64    // Total bytes received from publisher
	FramesReceived    uint64    // Total frames received
	KeyFramesReceived uint64    // Total keyframes received
	DroppedFrames     uint64    // Frames dropped due to errors/backpressure
	LastFrameTime     time.Time // Time of last frame received
	Bitrate           int       // Current bitrate in bps (rolling average)
}

// UpdateStats updates stream statistics
func (s *Stream) UpdateStats(frame *Frame) {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.Stats.FramesReceived++
	s.Stats.BytesReceived += uint64(len(frame.Payload))
	s.Stats.LastFrameTime = time.Now()

	if frame.IsKeyFrame {
		s.Stats.KeyFramesReceived++
	}
}

// IncrementViewers atomically increments the viewer count
func (s *Stream) IncrementViewers() {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.ViewerCount++
}

// DecrementViewers atomically decrements the viewer count
func (s *Stream) DecrementViewers() {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.ViewerCount > 0 {
		s.ViewerCount--
	}
}

// GetViewerCount safely returns the current viewer count
func (s *Stream) GetViewerCount() int {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.ViewerCount
}

// SetState safely updates the stream state
func (s *Stream) SetState(state StreamState) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.State = state

	if state == StreamStateLive && s.StartedAt.IsZero() {
		s.StartedAt = time.Now()
	} else if state == StreamStateStopped {
		now := time.Now()
		s.StoppedAt = &now
	}
}

// GetState safely returns the current stream state
func (s *Stream) GetState() StreamState {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.State
}

// IncrementDroppedFrames atomically increments the dropped frames counter
func (s *Stream) IncrementDroppedFrames() {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.Stats.DroppedFrames++
}
