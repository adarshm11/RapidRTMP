package config

import (
	"os"
	"strconv"
	"time"
)

// Config holds all application configuration
type Config struct {
	// HTTP Server
	HTTPAddr string

	// RTMP Server
	RTMPAddr       string
	RTMPIngestAddr string // Public RTMP URL for publishers

	// Storage
	StorageDir string

	// HLS
	HLSSegmentDuration time.Duration
	HLSMaxSegments     int

	// Auth
	DefaultTokenExpiration time.Duration
	MaxTokenExpiration     time.Duration

	// Limits
	MaxConcurrentStreams int
	MaxViewersPerStream  int
}

// Load loads configuration from environment variables with defaults
func Load() *Config {
	return &Config{
		HTTPAddr:               getEnv("HTTP_ADDR", ":8080"),
		RTMPAddr:               getEnv("RTMP_ADDR", ":1935"),
		RTMPIngestAddr:         getEnv("RTMP_INGEST_ADDR", "rtmp://localhost:1935"),
		StorageDir:             getEnv("STORAGE_DIR", "./data/streams"),
		HLSSegmentDuration:     getDurationEnv("HLS_SEGMENT_DURATION", 2*time.Second),
		HLSMaxSegments:         getIntEnv("HLS_MAX_SEGMENTS", 10),
		DefaultTokenExpiration: getDurationEnv("DEFAULT_TOKEN_EXPIRATION", 1*time.Hour),
		MaxTokenExpiration:     getDurationEnv("MAX_TOKEN_EXPIRATION", 24*time.Hour),
		MaxConcurrentStreams:   getIntEnv("MAX_CONCURRENT_STREAMS", 100),
		MaxViewersPerStream:    getIntEnv("MAX_VIEWERS_PER_STREAM", 1000),
	}
}

// Helper functions to get environment variables with defaults

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

func getIntEnv(key string, defaultValue int) int {
	if value := os.Getenv(key); value != "" {
		if intValue, err := strconv.Atoi(value); err == nil {
			return intValue
		}
	}
	return defaultValue
}

func getDurationEnv(key string, defaultValue time.Duration) time.Duration {
	if value := os.Getenv(key); value != "" {
		if duration, err := time.ParseDuration(value); err == nil {
			return duration
		}
	}
	return defaultValue
}
