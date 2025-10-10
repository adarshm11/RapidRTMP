package models

// Frame represents a single audio or video frame from RTMP ingest
type Frame struct {
	StreamKey  string                 // Unique identifier for the stream
	IsVideo    bool                   // true for video, false for audio
	Timestamp  uint32                 // PTS/DTS timestamp in milliseconds
	Payload    []byte                 // Raw NAL units (H.264) or AAC frames
	Codec      string                 // "h264", "h265", "aac", "mp3"
	IsKeyFrame bool                   // true if this is an IDR frame (video only)
	Metadata   map[string]interface{} // Additional codec-specific metadata
}

// CodecInfo contains initialization data for a codec
type CodecInfo struct {
	Codec       string  // "h264", "aac", etc.
	SPS         []byte  // H.264 Sequence Parameter Set (video)
	PPS         []byte  // H.264 Picture Parameter Set (video)
	VPS         []byte  // H.265 Video Parameter Set (optional)
	AudioConfig []byte  // AAC audio specific config
	Width       int     // Video width
	Height      int     // Video height
	FrameRate   float64 // Video frame rate
	SampleRate  int     // Audio sample rate
	Channels    int     // Audio channels
	Bitrate     int     // Bitrate in bps
}
