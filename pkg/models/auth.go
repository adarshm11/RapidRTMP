package models

import "time"

// PublishToken represents a token for publishing to a stream
type PublishToken struct {
	Token       string    // The actual token string
	StreamKey   string    // Stream key this token is valid for
	CreatedAt   time.Time // When token was created
	ExpiresAt   time.Time // When token expires
	PublisherIP string    // IP address that requested the token
	IsUsed      bool      // Whether token has been used
}

// IsValid checks if the token is still valid
func (t *PublishToken) IsValid() bool {
	return !t.IsUsed && time.Now().Before(t.ExpiresAt)
}

// PublishRequest represents a request to create a publish token
type PublishRequest struct {
	StreamKey string `json:"streamKey" binding:"required"`
	ExpiresIn int    `json:"expiresIn"` // Seconds until expiration (default 3600)
}

// PublishResponse represents the response to a publish request
type PublishResponse struct {
	PublishURL string `json:"publishUrl"`
	StreamKey  string `json:"streamKey"`
	Token      string `json:"token"`
	ExpiresAt  string `json:"expiresAt"`
}

// StreamInfo represents stream metadata returned by the API
type StreamInfo struct {
	StreamKey  string                 `json:"streamKey"`
	Active     bool                   `json:"active"`
	State      string                 `json:"state"`
	Viewers    int                    `json:"viewers"`
	StartedAt  string                 `json:"startedAt,omitempty"`
	Duration   int                    `json:"duration,omitempty"` // seconds
	VideoCodec string                 `json:"videoCodec,omitempty"`
	AudioCodec string                 `json:"audioCodec,omitempty"`
	Resolution string                 `json:"resolution,omitempty"` // e.g., "1920x1080"
	Bitrate    int                    `json:"bitrate,omitempty"`
	Metadata   map[string]interface{} `json:"metadata,omitempty"`
}

// StreamListResponse represents a list of streams
type StreamListResponse struct {
	Streams []StreamInfo `json:"streams"`
	Total   int          `json:"total"`
}
