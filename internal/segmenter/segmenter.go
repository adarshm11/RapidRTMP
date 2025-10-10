package segmenter

import (
	"bytes"
	"fmt"
	"log"
	"sync"
	"time"

	"rapidrtmp/internal/storage"
	"rapidrtmp/internal/streammanager"
	"rapidrtmp/pkg/models"
)

// Segmenter handles HLS segmentation for streams
type Segmenter struct {
	storage       storage.Storage
	streamManager *streammanager.Manager
	playlists     map[string]*PlaylistManager
	mu            sync.RWMutex

	// Config
	segmentDuration time.Duration
	maxSegments     int
}

// New creates a new segmenter
func New(storage storage.Storage, streamManager *streammanager.Manager) *Segmenter {
	return &Segmenter{
		storage:         storage,
		streamManager:   streamManager,
		playlists:       make(map[string]*PlaylistManager),
		segmentDuration: 2 * time.Second,
		maxSegments:     10,
	}
}

// StartSegmenting starts segmentation for a stream
func (s *Segmenter) StartSegmenting(streamKey string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	// Check if already segmenting
	if _, exists := s.playlists[streamKey]; exists {
		return fmt.Errorf("already segmenting stream %s", streamKey)
	}

	// Create playlist manager
	pm := &PlaylistManager{
		streamKey:       streamKey,
		segmenter:       s,
		segments:        make([]*models.Segment, 0),
		targetDuration:  int(s.segmentDuration.Seconds()),
		maxSegments:     s.maxSegments,
		sequenceNumber:  0,
		currentSegment:  newSegmentBuffer(),
	}

	s.playlists[streamKey] = pm

	// Subscribe to stream frames
	frameChan, cleanup := s.streamManager.Subscribe(streamKey, 1000)
	pm.cleanup = cleanup

	// Start processing frames
	go pm.processFrames(frameChan)

	log.Printf("Started HLS segmentation for stream %s", streamKey)
	return nil
}

// StopSegmenting stops segmentation for a stream
func (s *Segmenter) StopSegmenting(streamKey string) {
	s.mu.Lock()
	defer s.mu.Unlock()

	pm, exists := s.playlists[streamKey]
	if !exists {
		return
	}

	// Cleanup
	if pm.cleanup != nil {
		pm.cleanup()
	}

	delete(s.playlists, streamKey)
	log.Printf("Stopped HLS segmentation for stream %s", streamKey)
}

// GetPlaylist returns the HLS playlist for a stream
func (s *Segmenter) GetPlaylist(streamKey string) (string, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	pm, exists := s.playlists[streamKey]
	if !exists {
		return "", fmt.Errorf("stream %s not found", streamKey)
	}

	return pm.generatePlaylist(), nil
}

// GetSegment returns a segment's data
func (s *Segmenter) GetSegment(streamKey string, segmentNum uint64) ([]byte, error) {
	path := fmt.Sprintf("%s/segment_%d.m4s", streamKey, segmentNum)
	return s.storage.Read(path)
}

// GetInitSegment returns the initialization segment
func (s *Segmenter) GetInitSegment(streamKey string) ([]byte, error) {
	path := fmt.Sprintf("%s/init.mp4", streamKey)
	return s.storage.Read(path)
}

// PlaylistManager manages playlist and segments for a stream
type PlaylistManager struct {
	streamKey      string
	segmenter      *Segmenter
	segments       []*models.Segment
	targetDuration int
	maxSegments    int
	sequenceNumber uint64
	currentSegment *SegmentBuffer
	cleanup        func()
	mu             sync.RWMutex
	hasInit        bool
}

// SegmentBuffer buffers frames for a segment
type SegmentBuffer struct {
	frames      []*models.Frame
	startTime   time.Time
	hasKeyFrame bool
	mu          sync.Mutex
}

func newSegmentBuffer() *SegmentBuffer {
	return &SegmentBuffer{
		frames:    make([]*models.Frame, 0),
		startTime: time.Now(),
	}
}

// processFrames processes incoming frames and creates segments
func (pm *PlaylistManager) processFrames(frameChan <-chan *models.Frame) {
	ticker := time.NewTicker(pm.segmenter.segmentDuration)
	defer ticker.Stop()

	for {
		select {
		case frame, ok := <-frameChan:
			if !ok {
				// Channel closed, finalize current segment
				pm.finalizeSegment()
				return
			}

			pm.addFrame(frame)

		case <-ticker.C:
			// Time to create a segment
			pm.finalizeSegment()
		}
	}
}

// addFrame adds a frame to the current segment
func (pm *PlaylistManager) addFrame(frame *models.Frame) {
	pm.mu.Lock()
	defer pm.mu.Unlock()

	pm.currentSegment.mu.Lock()
	defer pm.currentSegment.mu.Unlock()

	// Track if we have a keyframe
	if frame.IsVideo && frame.IsKeyFrame {
		pm.currentSegment.hasKeyFrame = true
	}

	pm.currentSegment.frames = append(pm.currentSegment.frames, frame)
}

// finalizeSegment finalizes the current segment and creates a new one
func (pm *PlaylistManager) finalizeSegment() {
	pm.mu.Lock()
	defer pm.mu.Unlock()

	pm.currentSegment.mu.Lock()
	frameCount := len(pm.currentSegment.frames)
	hasKeyFrame := pm.currentSegment.hasKeyFrame
	frames := pm.currentSegment.frames
	pm.currentSegment.mu.Unlock()

	// Don't create segment if no frames or no keyframe
	if frameCount == 0 || !hasKeyFrame {
		return
	}

	// Create segment
	segmentNum := pm.sequenceNumber
	pm.sequenceNumber++

	// Convert frames to segment data (simplified for now)
	segmentData := pm.framesToSegmentData(frames)

	// Save segment to storage
	path := fmt.Sprintf("%s/segment_%d.m4s", pm.streamKey, segmentNum)
	if err := pm.segmenter.storage.Write(path, segmentData); err != nil {
		log.Printf("Failed to write segment %d for stream %s: %v", segmentNum, pm.streamKey, err)
		return
	}

	// Create segment metadata
	segment := &models.Segment{
		StreamKey:   pm.streamKey,
		SequenceNum: segmentNum,
		Duration:    float64(pm.segmenter.segmentDuration.Seconds()),
		FilePath:    path,
		FileSize:    int64(len(segmentData)),
		CreatedAt:   time.Now(),
		IsAvailable: true,
	}

	// Add to segments list
	pm.segments = append(pm.segments, segment)

	// Maintain sliding window
	if len(pm.segments) > pm.maxSegments {
		// Remove oldest segment
		oldSegment := pm.segments[0]
		pm.segments = pm.segments[1:]

		// Delete old segment file
		go pm.segmenter.storage.Delete(oldSegment.FilePath)
	}

	// Create init segment on first segment
	if !pm.hasInit {
		pm.createInitSegment(frames)
		pm.hasInit = true
	}

	// Reset current segment
	pm.currentSegment = newSegmentBuffer()

	log.Printf("Created segment %d for stream %s (%d frames, %.2f KB)",
		segmentNum, pm.streamKey, frameCount, float64(len(segmentData))/1024)
}

// framesToSegmentData converts frames to segment data
// This is a simplified version - in production you'd use a proper MP4 muxer
func (pm *PlaylistManager) framesToSegmentData(frames []*models.Frame) []byte {
	var buf bytes.Buffer

	// Simple concatenation of frame payloads
	// In production, this would be proper fMP4/CMAF packaging
	for _, frame := range frames {
		buf.Write(frame.Payload)
	}

	return buf.Bytes()
}

// createInitSegment creates the initialization segment
func (pm *PlaylistManager) createInitSegment(frames []*models.Frame) {
	// Create a simple init segment
	// In production, this would include proper MP4 headers (ftyp, moov boxes)
	initData := []byte("fMP4 init segment placeholder")

	path := fmt.Sprintf("%s/init.mp4", pm.streamKey)
	if err := pm.segmenter.storage.Write(path, initData); err != nil {
		log.Printf("Failed to write init segment for stream %s: %v", pm.streamKey, err)
		return
	}

	log.Printf("Created init segment for stream %s", pm.streamKey)
}

// generatePlaylist generates the HLS playlist
func (pm *PlaylistManager) generatePlaylist() string {
	pm.mu.RLock()
	defer pm.mu.RUnlock()

	var buf bytes.Buffer

	// HLS playlist header
	buf.WriteString("#EXTM3U\n")
	buf.WriteString("#EXT-X-VERSION:7\n")
	buf.WriteString(fmt.Sprintf("#EXT-X-TARGETDURATION:%d\n", pm.targetDuration))

	// Media sequence (first segment number in playlist)
	if len(pm.segments) > 0 {
		buf.WriteString(fmt.Sprintf("#EXT-X-MEDIA-SEQUENCE:%d\n", pm.segments[0].SequenceNum))
	} else {
		buf.WriteString("#EXT-X-MEDIA-SEQUENCE:0\n")
	}

	// Map (init segment)
	if pm.hasInit {
		buf.WriteString(fmt.Sprintf("#EXT-X-MAP:URI=\"init.mp4\"\n"))
	}

	// Segments
	for _, seg := range pm.segments {
		buf.WriteString(fmt.Sprintf("#EXTINF:%.3f,\n", seg.Duration))
		buf.WriteString(fmt.Sprintf("segment_%d.m4s\n", seg.SequenceNum))
	}

	// Note: We don't add #EXT-X-ENDLIST because it's a live stream

	return buf.String()
}

