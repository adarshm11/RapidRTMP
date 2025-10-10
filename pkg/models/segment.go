package models

import "time"

// Segment represents an HLS media segment
type Segment struct {
	StreamKey   string    // Stream this segment belongs to
	SequenceNum uint64    // Segment sequence number
	Duration    float64   // Duration in seconds
	FilePath    string    // Path to segment file (local or S3)
	FileSize    int64     // Size in bytes
	CreatedAt   time.Time // When segment was created
	IsAvailable bool      // Whether segment is ready for serving
}

// Playlist represents an HLS playlist state
type Playlist struct {
	StreamKey       string     // Stream this playlist belongs to
	TargetDuration  int        // EXT-X-TARGETDURATION
	MediaSequence   uint64     // EXT-X-MEDIA-SEQUENCE
	Segments        []*Segment // List of segments in playlist
	InitSegmentPath string     // Path to init.mp4
	MaxSegments     int        // Max segments to keep in playlist (sliding window)
	LastUpdated     time.Time  // Last time playlist was updated
}

// AddSegment adds a new segment to the playlist and maintains the sliding window
func (p *Playlist) AddSegment(seg *Segment) {
	p.Segments = append(p.Segments, seg)
	p.LastUpdated = time.Now()

	// Maintain sliding window
	if len(p.Segments) > p.MaxSegments {
		// Remove oldest segment
		p.Segments = p.Segments[1:]
		p.MediaSequence++
	}
}

// GetM3U8Content generates the HLS playlist content
func (p *Playlist) GetM3U8Content() string {
	// This will be implemented when we build the HLS packager
	// For now, return empty string
	return ""
}
