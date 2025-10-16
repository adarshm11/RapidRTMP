package muxer

import (
	"bytes"
	"fmt"
	"io"
	"log"
	"os/exec"
	"sync"

	"rapidrtmp/pkg/models"
)

// FFmpegMuxer uses FFmpeg to mux H.264/AAC frames into fMP4 segments
type FFmpegMuxer struct {
	mu sync.Mutex
}

// NewFFmpegMuxer creates a new FFmpeg-based muxer
func NewFFmpegMuxer() *FFmpegMuxer {
	return &FFmpegMuxer{}
}

// CreateInitSegment creates an fMP4 initialization segment
// This contains the ftyp and moov boxes needed for CMAF/HLS
func (m *FFmpegMuxer) CreateInitSegment(videoCodecData, audioCodecData []byte) ([]byte, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	// Use FFmpeg to create a minimal fMP4 init segment
	// We'll feed it a tiny bit of data just to generate the headers
	cmd := exec.Command("ffmpeg",
		"-f", "h264", // Input format
		"-i", "pipe:0", // Read from stdin
		"-c", "copy", // Don't re-encode
		"-f", "mp4", // Output format
		// Create init with fragmented flags but without empty_moov in media segments
		"-movflags", "frag_keyframe+separate_moof+default_base_moof", // CMAF-style init
		"-frag_duration", "1000000", // 1 second fragments in microseconds
		"-t", "0.001", // Very short duration, we just want the init
		"pipe:1", // Write to stdout
	)

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	stdin, err := cmd.StdinPipe()
	if err != nil {
		return nil, fmt.Errorf("failed to get stdin pipe: %w", err)
	}

	if err := cmd.Start(); err != nil {
		return nil, fmt.Errorf("failed to start ffmpeg: %w", err)
	}

	// Write minimal H.264 data (just the codec data)
	if len(videoCodecData) > 0 {
		stdin.Write(videoCodecData)
	}
	stdin.Close()

	if err := cmd.Wait(); err != nil {
		log.Printf("FFmpeg init segment stderr: %s", stderr.String())
		// FFmpeg might return error due to very short duration, but still produce output
	}

	initData := stdout.Bytes()
	if len(initData) == 0 {
		return nil, fmt.Errorf("ffmpeg produced no output for init segment")
	}

	log.Printf("Created init segment: %d bytes", len(initData))
	return initData, nil
}

// CreateMediaSegment muxes frames into an fMP4 media segment
func (m *FFmpegMuxer) CreateMediaSegment(frames []*models.Frame) ([]byte, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if len(frames) == 0 {
		return nil, fmt.Errorf("no frames to mux")
	}

	// Separate video and audio frames
	var videoFrames, audioFrames []*models.Frame
	for _, frame := range frames {
		if frame.IsVideo {
			videoFrames = append(videoFrames, frame)
		} else {
			audioFrames = append(audioFrames, frame)
		}
	}

	if len(videoFrames) == 0 {
		return nil, fmt.Errorf("no video frames in segment")
	}

	// Calculate approximate framerate and duration
	framerate := "30" // Default, could be detected from timestamps
	duration := fmt.Sprintf("%.3f", float64(len(videoFrames))/30.0)

	// For now, we'll just mux video. Audio sync is more complex.
	// Use FFmpeg to convert raw H.264 to fMP4
	cmd := exec.Command("ffmpeg",
		"-hide_banner",
		"-loglevel", "error", // Only show errors
		"-f", "h264", // Input is raw H.264
		"-r", framerate, // Set input framerate
		"-i", "pipe:0", // Read from stdin
		"-t", duration, // Duration
		"-c:v", "copy", // Don't re-encode
		"-f", "mp4", // Output as MP4
		// CMAF-style media fragments: moof+mdat only
		"-movflags", "frag_keyframe+separate_moof+default_base_moof+dash+omit_tfhd_offset",
		"-frag_duration", "1000000", // 1 second fragments in microseconds
		"-min_frag_duration", "1000000",
		"-reset_timestamps", "1", // Reset timestamps for each segment
		"-avoid_negative_ts", "make_zero", // Handle timestamp issues
		"-y",     // Overwrite output
		"pipe:1", // Write to stdout
	)

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	stdin, err := cmd.StdinPipe()
	if err != nil {
		return nil, fmt.Errorf("failed to get stdin pipe: %w", err)
	}

	if err := cmd.Start(); err != nil {
		return nil, fmt.Errorf("failed to start ffmpeg: %w", err)
	}

	// Write frames to FFmpeg (they should already be in Annex-B format from RTMP handler)
	writeErr := make(chan error, 1)
	go func() {
		defer stdin.Close()
		for _, frame := range videoFrames {
			// Frames are already in Annex-B format with SPS/PPS prepended to keyframes
			if _, err := stdin.Write(frame.Payload); err != nil {
				writeErr <- err
				return
			}
		}
		writeErr <- nil
	}()

	// Wait for write to complete or error
	if err := <-writeErr; err != nil {
		log.Printf("Error writing frames to ffmpeg: %v", err)
	}

	if err := cmd.Wait(); err != nil {
		errMsg := stderr.String()
		if len(errMsg) > 0 {
			log.Printf("FFmpeg error: %s", errMsg)
		}
		// Check if we got any output despite the error
		if stdout.Len() == 0 {
			return nil, fmt.Errorf("ffmpeg failed: %w", err)
		}
		// Sometimes FFmpeg returns error but still produces valid output
		log.Printf("FFmpeg returned error but produced %d bytes output, using it anyway", stdout.Len())
	}

	segmentData := stdout.Bytes()
	if len(segmentData) == 0 {
		return nil, fmt.Errorf("ffmpeg produced no output")
	}

	log.Printf("Created media segment: %d frames -> %d bytes", len(videoFrames), len(segmentData))
	return segmentData, nil
}

// MuxFramesToMP4 is a simpler interface that wraps CreateMediaSegment
func (m *FFmpegMuxer) MuxFramesToMP4(frames []*models.Frame) ([]byte, error) {
	return m.CreateMediaSegment(frames)
}

// CheckFFmpegAvailable checks if FFmpeg is installed and available
func CheckFFmpegAvailable() error {
	cmd := exec.Command("ffmpeg", "-version")
	var stderr bytes.Buffer
	cmd.Stderr = &stderr

	output, err := cmd.Output()
	if err != nil {
		return fmt.Errorf("ffmpeg not found or not working: %w\nStderr: %s", err, stderr.String())
	}

	if len(output) == 0 {
		return fmt.Errorf("ffmpeg produced no output")
	}

	log.Println("FFmpeg is available and working")
	return nil
}

// ExtractCodecData attempts to extract SPS/PPS from the first few video frames
func ExtractCodecData(frames []*models.Frame) []byte {
	for _, frame := range frames {
		if frame.IsVideo && frame.IsKeyFrame && len(frame.Payload) > 0 {
			// Look for SPS/PPS NAL units (type 7 and 8)
			// This is a simplified approach - proper parsing would use H.264 parser
			payload := frame.Payload

			// H.264 NAL units start with 0x00 0x00 0x00 0x01 or 0x00 0x00 0x01
			// We'll just return the first keyframe as it likely contains codec data
			if len(payload) > 100 {
				return payload[:100] // First 100 bytes likely contain SPS/PPS
			}
			return payload
		}
	}
	return nil
}

// WriteRawH264 is a helper to write raw H.264 data for debugging
func WriteRawH264(w io.Writer, frames []*models.Frame) error {
	for _, frame := range frames {
		if frame.IsVideo {
			if _, err := w.Write(frame.Payload); err != nil {
				return err
			}
		}
	}
	return nil
}
