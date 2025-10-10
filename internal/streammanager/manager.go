package streammanager

import (
	"fmt"
	"rapidrtmp/pkg/models"
	"sync"
)

// Manager handles stream lifecycle and maintains in-memory registry
type Manager struct {
	streams map[string]*models.Stream // streamKey -> Stream
	mu      sync.RWMutex

	// Channels for pub/sub
	subscribers map[string][]chan *models.Frame // streamKey -> list of subscriber channels
	subMu       sync.RWMutex
}

// New creates a new stream manager
func New() *Manager {
	return &Manager{
		streams:     make(map[string]*models.Stream),
		subscribers: make(map[string][]chan *models.Frame),
	}
}

// CreateStream creates or retrieves a stream
func (m *Manager) CreateStream(streamKey string, publisherIP string) (*models.Stream, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	// Check if stream already exists
	if stream, exists := m.streams[streamKey]; exists {
		// If stream is already live, don't allow another publisher
		if stream.GetState() == models.StreamStateLive {
			return nil, fmt.Errorf("stream %s is already live", streamKey)
		}
	}

	// Create new stream
	stream := &models.Stream{
		Key:         streamKey,
		State:       models.StreamStateConnecting,
		PublisherIP: publisherIP,
		Metadata:    make(map[string]interface{}),
	}

	m.streams[streamKey] = stream
	return stream, nil
}

// GetStream retrieves a stream by key
func (m *Manager) GetStream(streamKey string) (*models.Stream, bool) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	stream, exists := m.streams[streamKey]
	return stream, exists
}

// GetAllStreams returns all streams
func (m *Manager) GetAllStreams() []*models.Stream {
	m.mu.RLock()
	defer m.mu.RUnlock()

	streams := make([]*models.Stream, 0, len(m.streams))
	for _, stream := range m.streams {
		streams = append(streams, stream)
	}

	return streams
}

// GetLiveStreams returns only live streams
func (m *Manager) GetLiveStreams() []*models.Stream {
	m.mu.RLock()
	defer m.mu.RUnlock()

	streams := make([]*models.Stream, 0)
	for _, stream := range m.streams {
		if stream.GetState() == models.StreamStateLive {
			streams = append(streams, stream)
		}
	}

	return streams
}

// StopStream stops a stream
func (m *Manager) StopStream(streamKey string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	stream, exists := m.streams[streamKey]
	if !exists {
		return fmt.Errorf("stream %s not found", streamKey)
	}

	stream.SetState(models.StreamStateStopped)

	// Close all subscriber channels
	m.closeSubscribers(streamKey)

	return nil
}

// DeleteStream removes a stream from the registry
func (m *Manager) DeleteStream(streamKey string) {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.closeSubscribers(streamKey)
	delete(m.streams, streamKey)
}

// PublishFrame publishes a frame to all subscribers
func (m *Manager) PublishFrame(frame *models.Frame) error {
	// Update stream stats
	stream, exists := m.GetStream(frame.StreamKey)
	if !exists {
		return fmt.Errorf("stream %s not found", frame.StreamKey)
	}

	stream.UpdateStats(frame)

	// Send frame to all subscribers
	m.subMu.RLock()
	subscribers, exists := m.subscribers[frame.StreamKey]
	m.subMu.RUnlock()

	if !exists || len(subscribers) == 0 {
		// No subscribers, frame is dropped
		return nil
	}

	// Send to all subscribers (non-blocking)
	for _, ch := range subscribers {
		select {
		case ch <- frame:
			// Frame sent successfully
		default:
			// Channel is full, drop frame
			stream.IncrementDroppedFrames()
		}
	}

	return nil
}

// Subscribe creates a subscription to a stream's frames
// Returns a channel that will receive frames and a cleanup function
func (m *Manager) Subscribe(streamKey string, bufferSize int) (<-chan *models.Frame, func()) {
	m.subMu.Lock()
	defer m.subMu.Unlock()

	// Create subscriber channel
	ch := make(chan *models.Frame, bufferSize)

	// Add to subscribers list
	if m.subscribers[streamKey] == nil {
		m.subscribers[streamKey] = make([]chan *models.Frame, 0)
	}
	m.subscribers[streamKey] = append(m.subscribers[streamKey], ch)

	// Return cleanup function
	cleanup := func() {
		m.unsubscribe(streamKey, ch)
	}

	return ch, cleanup
}

// unsubscribe removes a subscriber channel
func (m *Manager) unsubscribe(streamKey string, ch chan *models.Frame) {
	m.subMu.Lock()
	defer m.subMu.Unlock()

	subscribers, exists := m.subscribers[streamKey]
	if !exists {
		return
	}

	// Find and remove the channel
	for i, subCh := range subscribers {
		if subCh == ch {
			// Remove from slice
			m.subscribers[streamKey] = append(subscribers[:i], subscribers[i+1:]...)
			close(ch)
			break
		}
	}

	// Clean up empty subscriber lists
	if len(m.subscribers[streamKey]) == 0 {
		delete(m.subscribers, streamKey)
	}
}

// closeSubscribers closes all subscriber channels for a stream
func (m *Manager) closeSubscribers(streamKey string) {
	m.subMu.Lock()
	defer m.subMu.Unlock()

	subscribers, exists := m.subscribers[streamKey]
	if !exists {
		return
	}

	// Close all channels
	for _, ch := range subscribers {
		close(ch)
	}

	delete(m.subscribers, streamKey)
}

// GetStreamCount returns the total number of streams
func (m *Manager) GetStreamCount() int {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return len(m.streams)
}

// GetLiveStreamCount returns the number of live streams
func (m *Manager) GetLiveStreamCount() int {
	m.mu.RLock()
	defer m.mu.RUnlock()

	count := 0
	for _, stream := range m.streams {
		if stream.GetState() == models.StreamStateLive {
			count++
		}
	}
	return count
}
