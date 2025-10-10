package metrics

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

// Metrics holds all Prometheus metrics
type Metrics struct {
	// Stream metrics
	ActiveStreams    prometheus.Gauge
	TotalStreams     prometheus.Counter
	StreamsStarted   prometheus.Counter
	StreamsStopped   prometheus.Counter
	StreamDuration   prometheus.Histogram

	// Frame metrics
	FramesReceived   *prometheus.CounterVec
	FramesDropped    *prometheus.CounterVec
	FrameSize        *prometheus.HistogramVec
	KeyFrames        prometheus.Counter

	// Segment metrics
	SegmentsCreated  prometheus.Counter
	SegmentDuration  prometheus.Histogram
	SegmentSize      prometheus.Histogram

	// Viewer metrics
	ActiveViewers    prometheus.Gauge
	TotalViewers     prometheus.Counter
	ViewerSessions   prometheus.Counter

	// HTTP metrics
	HTTPRequests     *prometheus.CounterVec
	HTTPDuration     *prometheus.HistogramVec

	// RTMP metrics
	RTMPConnections  prometheus.Counter
	RTMPDisconnects  prometheus.Counter
	RTMPErrors       prometheus.Counter
	RTMPBytesReceived prometheus.Counter

	// System metrics
	BytesStored      prometheus.Gauge
	SegmentsStored   prometheus.Gauge
}

// New creates and registers all metrics
func New() *Metrics {
	m := &Metrics{
		// Stream metrics
		ActiveStreams: promauto.NewGauge(prometheus.GaugeOpts{
			Name: "rapidrtmp_active_streams",
			Help: "Number of currently active streams",
		}),
		TotalStreams: promauto.NewCounter(prometheus.CounterOpts{
			Name: "rapidrtmp_total_streams",
			Help: "Total number of streams since server start",
		}),
		StreamsStarted: promauto.NewCounter(prometheus.CounterOpts{
			Name: "rapidrtmp_streams_started_total",
			Help: "Total number of streams started",
		}),
		StreamsStopped: promauto.NewCounter(prometheus.CounterOpts{
			Name: "rapidrtmp_streams_stopped_total",
			Help: "Total number of streams stopped",
		}),
		StreamDuration: promauto.NewHistogram(prometheus.HistogramOpts{
			Name:    "rapidrtmp_stream_duration_seconds",
			Help:    "Duration of streams in seconds",
			Buckets: prometheus.ExponentialBuckets(10, 2, 10), // 10s to ~2.8h
		}),

		// Frame metrics
		FramesReceived: promauto.NewCounterVec(
			prometheus.CounterOpts{
				Name: "rapidrtmp_frames_received_total",
				Help: "Total number of frames received",
			},
			[]string{"stream_key", "type"}, // type: video or audio
		),
		FramesDropped: promauto.NewCounterVec(
			prometheus.CounterOpts{
				Name: "rapidrtmp_frames_dropped_total",
				Help: "Total number of frames dropped",
			},
			[]string{"stream_key", "reason"},
		),
		FrameSize: promauto.NewHistogramVec(
			prometheus.HistogramOpts{
				Name:    "rapidrtmp_frame_size_bytes",
				Help:    "Size of frames in bytes",
				Buckets: prometheus.ExponentialBuckets(1024, 2, 10), // 1KB to ~512KB
			},
			[]string{"type"},
		),
		KeyFrames: promauto.NewCounter(prometheus.CounterOpts{
			Name: "rapidrtmp_keyframes_total",
			Help: "Total number of keyframes received",
		}),

		// Segment metrics
		SegmentsCreated: promauto.NewCounter(prometheus.CounterOpts{
			Name: "rapidrtmp_segments_created_total",
			Help: "Total number of HLS segments created",
		}),
		SegmentDuration: promauto.NewHistogram(prometheus.HistogramOpts{
			Name:    "rapidrtmp_segment_duration_seconds",
			Help:    "Duration of HLS segments",
			Buckets: []float64{1, 2, 3, 4, 5, 10},
		}),
		SegmentSize: promauto.NewHistogram(prometheus.HistogramOpts{
			Name:    "rapidrtmp_segment_size_bytes",
			Help:    "Size of HLS segments in bytes",
			Buckets: prometheus.ExponentialBuckets(10240, 2, 10), // 10KB to ~5MB
		}),

		// Viewer metrics
		ActiveViewers: promauto.NewGauge(prometheus.GaugeOpts{
			Name: "rapidrtmp_active_viewers",
			Help: "Number of currently active viewers",
		}),
		TotalViewers: promauto.NewCounter(prometheus.CounterOpts{
			Name: "rapidrtmp_total_viewers",
			Help: "Total number of viewers since server start",
		}),
		ViewerSessions: promauto.NewCounter(prometheus.CounterOpts{
			Name: "rapidrtmp_viewer_sessions_total",
			Help: "Total number of viewer sessions",
		}),

		// HTTP metrics
		HTTPRequests: promauto.NewCounterVec(
			prometheus.CounterOpts{
				Name: "rapidrtmp_http_requests_total",
				Help: "Total number of HTTP requests",
			},
			[]string{"method", "path", "status"},
		),
		HTTPDuration: promauto.NewHistogramVec(
			prometheus.HistogramOpts{
				Name:    "rapidrtmp_http_request_duration_seconds",
				Help:    "Duration of HTTP requests",
				Buckets: prometheus.DefBuckets,
			},
			[]string{"method", "path"},
		),

		// RTMP metrics
		RTMPConnections: promauto.NewCounter(prometheus.CounterOpts{
			Name: "rapidrtmp_rtmp_connections_total",
			Help: "Total number of RTMP connections",
		}),
		RTMPDisconnects: promauto.NewCounter(prometheus.CounterOpts{
			Name: "rapidrtmp_rtmp_disconnects_total",
			Help: "Total number of RTMP disconnections",
		}),
		RTMPErrors: promauto.NewCounter(prometheus.CounterOpts{
			Name: "rapidrtmp_rtmp_errors_total",
			Help: "Total number of RTMP errors",
		}),
		RTMPBytesReceived: promauto.NewCounter(prometheus.CounterOpts{
			Name: "rapidrtmp_rtmp_bytes_received_total",
			Help: "Total bytes received via RTMP",
		}),

		// System metrics
		BytesStored: promauto.NewGauge(prometheus.GaugeOpts{
			Name: "rapidrtmp_bytes_stored",
			Help: "Total bytes stored on disk",
		}),
		SegmentsStored: promauto.NewGauge(prometheus.GaugeOpts{
			Name: "rapidrtmp_segments_stored",
			Help: "Number of segments currently stored",
		}),
	}

	return m
}

// RecordStreamStart records a stream starting
func (m *Metrics) RecordStreamStart() {
	m.ActiveStreams.Inc()
	m.TotalStreams.Inc()
	m.StreamsStarted.Inc()
}

// RecordStreamStop records a stream stopping
func (m *Metrics) RecordStreamStop(durationSeconds float64) {
	m.ActiveStreams.Dec()
	m.StreamsStopped.Inc()
	m.StreamDuration.Observe(durationSeconds)
}

// RecordFrame records a frame received
func (m *Metrics) RecordFrame(streamKey string, isVideo bool, size int) {
	frameType := "audio"
	if isVideo {
		frameType = "video"
	}
	m.FramesReceived.WithLabelValues(streamKey, frameType).Inc()
	m.FrameSize.WithLabelValues(frameType).Observe(float64(size))
}

// RecordKeyFrame records a keyframe
func (m *Metrics) RecordKeyFrame() {
	m.KeyFrames.Inc()
}

// RecordFrameDropped records a dropped frame
func (m *Metrics) RecordFrameDropped(streamKey, reason string) {
	m.FramesDropped.WithLabelValues(streamKey, reason).Inc()
}

// RecordSegment records a segment created
func (m *Metrics) RecordSegment(durationSeconds float64, sizeBytes int64) {
	m.SegmentsCreated.Inc()
	m.SegmentDuration.Observe(durationSeconds)
	m.SegmentSize.Observe(float64(sizeBytes))
	m.SegmentsStored.Inc()
}

// RecordSegmentDeleted records a segment deleted
func (m *Metrics) RecordSegmentDeleted() {
	m.SegmentsStored.Dec()
}

// RecordHTTPRequest records an HTTP request
func (m *Metrics) RecordHTTPRequest(method, path string, status int, durationSeconds float64) {
	m.HTTPRequests.WithLabelValues(method, path, m.statusCodeToString(status)).Inc()
	m.HTTPDuration.WithLabelValues(method, path).Observe(durationSeconds)
}

// RecordRTMPConnection records an RTMP connection
func (m *Metrics) RecordRTMPConnection() {
	m.RTMPConnections.Inc()
}

// RecordRTMPDisconnect records an RTMP disconnection
func (m *Metrics) RecordRTMPDisconnect() {
	m.RTMPDisconnects.Inc()
}

// RecordRTMPError records an RTMP error
func (m *Metrics) RecordRTMPError() {
	m.RTMPErrors.Inc()
}

// RecordRTMPBytes records bytes received via RTMP
func (m *Metrics) RecordRTMPBytes(bytes uint64) {
	m.RTMPBytesReceived.Add(float64(bytes))
}

// RecordViewer records a viewer
func (m *Metrics) RecordViewerStart() {
	m.ActiveViewers.Inc()
	m.TotalViewers.Inc()
	m.ViewerSessions.Inc()
}

// RecordViewerStop records a viewer stopping
func (m *Metrics) RecordViewerStop() {
	m.ActiveViewers.Dec()
}

// statusCodeToString converts an HTTP status code to a string
func (m *Metrics) statusCodeToString(code int) string {
	switch {
	case code >= 200 && code < 300:
		return "2xx"
	case code >= 300 && code < 400:
		return "3xx"
	case code >= 400 && code < 500:
		return "4xx"
	case code >= 500:
		return "5xx"
	default:
		return "unknown"
	}
}

