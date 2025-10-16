# RapidRTMP

A high-performance RTMP streaming server written in Go, designed to accept live video streams from publishers (OBS, FFmpeg) and deliver them to viewers via HLS.

## 🚀 Current Status

**Phase 1: Core Infrastructure** ✅ **COMPLETE**  
**Phase 2: RTMP Ingest** ✅ **COMPLETE**  
**Phase 3: HLS Packaging** ✅ **COMPLETE**  
**Phase 4: Production Features** ✅ **COMPLETE**

### Implemented Features:

- ✅ **HTTP API Server** - REST API for stream management
- ✅ **RTMP Ingest Server** - Accept live streams from OBS/FFmpeg on port 1935
- ✅ **Stream Manager** - In-memory stream registry with pub/sub
- ✅ **Auth Manager** - Token-based authentication for publishers
- ✅ **Storage Layer** - Local filesystem storage (S3 ready)
- ✅ **Data Models** - Complete type system for streams, frames, segments
- ✅ **Configuration** - Environment-based config management
- ✅ **Frame Processing** - Extract H.264 video and AAC audio frames
- ✅ **Publisher Authentication** - Validate tokens from OBS/FFmpeg
- ✅ **HLS Segmenter** - Generate HLS playlists and segments
- ✅ **Segment Storage** - Automatic segment management with retention policy
- ✅ **HLS Playback** - Serve streams to viewers via HTTP
- ✅ **Prometheus Metrics** - Comprehensive metrics for monitoring
- ✅ **Health Checks** - Kubernetes-ready health and readiness endpoints
- ✅ **Docker Support** - Production-ready containerization
- ✅ **Kubernetes Manifests** - Complete K8s deployment configs
- ✅ **Google Cloud Storage** - Production storage backend with CDN support

**What's Working:**
- ✅ **Accept RTMP streams** from OBS Studio and FFmpeg
- ✅ **Generate publish tokens** via API
- ✅ **Real-time frame extraction** (video/audio)
- ✅ **H.264 codec data extraction** (SPS/PPS from AVC sequence headers)
- ✅ **AVCC to Annex-B conversion** for proper H.264 handling
- ✅ **Automatic HLS segmentation** (1-second segments for low latency)
- ✅ **FFmpeg-based fMP4/CMAF muxing** for browser-compatible segments
- ✅ **HLS playlist generation** (.m3u8 files with INDEPENDENT-SEGMENTS)
- ✅ **Serve HLS streams** to browsers (init.mp4 + CMAF segments)
- ✅ **List active streams** with metadata
- ✅ **Get stream info** (codec, resolution, bitrate)
- ✅ **Stop streams remotely**
- ✅ **Track stream statistics** (frames, viewers, dropped frames)
- ✅ **Web-based test player** with auto-recovery and cache-busting
- ✅ **Low-latency streaming** (~1-2 seconds glass-to-glass)
- ✅ **CORS support** for cross-origin playback
- ✅ **Prometheus metrics** (30+ metrics tracked)
- ✅ **Health/readiness endpoints** for K8s
- ✅ **Docker containerization** with multi-stage builds
- ✅ **Kubernetes deployment** manifests
- ✅ **Metrics middleware** for HTTP request tracking
- ✅ **GCS storage backend** with signed URLs and CDN integration

**Future Enhancements:**
- 🔨 Multi-bitrate transcoding (ABR/adaptive streaming)
- 🔨 WebRTC gateway for sub-second latency
- 🔨 DVR/VOD support with recording
- 🔨 AWS S3 storage backend
- 🔨 Azure Blob Storage backend
- 🔨 CloudFront/Fastly CDN integration
- 🔨 Distributed tracing (OpenTelemetry)
- 🔨 Audio-only streams and audio muxing
- 🔨 Stream overlays and watermarks

## 📋 Requirements

- **Go 1.24.3+**
- **FFmpeg** - Required for HLS segment muxing (install with `brew install ffmpeg` on macOS)
- No other external dependencies (Go modules handles everything)
- **Optional**: Google Cloud Platform account for GCS storage

## 📚 Documentation

- **[GCS Setup Guide](docs/GCS_SETUP.md)** - Complete guide for Google Cloud Storage
- **[TESTING.md](TESTING.md)** - How to test with OBS/FFmpeg
- **[test-player.html](test-player.html)** - Web-based HLS player

## 🛠️ Installation

### Clone and Build

```bash
git clone https://github.com/yourusername/RapidRTMP.git
cd RapidRTMP
go build
```

### Run

#### Local Storage (Default)

```bash
./rapidrtmp
```

#### Google Cloud Storage

```bash
# Set up GCS credentials (see docs/GCS_SETUP.md)
export GOOGLE_APPLICATION_CREDENTIALS="path/to/service-account-key.json"

# Configure GCS
export STORAGE_TYPE="gcs"
export GCS_PROJECT_ID="your-project-id"
export GCS_BUCKET_NAME="your-bucket-name"

./rapidrtmp
```

The server will start with:
- **HTTP API**: `http://localhost:8080`
- **RTMP Ingest**: `rtmp://localhost:1935`
- **Metrics**: `http://localhost:8080/metrics`
- **Health**: `http://localhost:8080/health`

### Configuration

Configure via environment variables:

```bash
# HTTP Server
export HTTP_ADDR=":8080"

# RTMP Server
export RTMP_ADDR=":1935"
export RTMP_INGEST_ADDR="rtmp://localhost:1935"

# Storage - Local (default)
export STORAGE_TYPE="local"
export STORAGE_DIR="./data/streams"

# Storage - Google Cloud Storage (optional)
export STORAGE_TYPE="gcs"
export GCS_PROJECT_ID="your-project-id"
export GCS_BUCKET_NAME="your-bucket-name"
export GCS_BASE_DIR="streams"
export GOOGLE_APPLICATION_CREDENTIALS="path/to/key.json"

# HLS Settings
export HLS_SEGMENT_DURATION="2s"
export HLS_MAX_SEGMENTS="10"

# Auth
export DEFAULT_TOKEN_EXPIRATION="1h"
export MAX_TOKEN_EXPIRATION="24h"

# Limits
export MAX_CONCURRENT_STREAMS="100"
export MAX_VIEWERS_PER_STREAM="1000"
```

## 📡 API Endpoints

### Health Check
```bash
GET /api/ping
```

**Response:**
```json
{
  "message": "pong",
  "time": 1760078592
}
```

### Request Publish Token
```bash
POST /api/v1/publish
Content-Type: application/json

{
  "streamKey": "my-stream",
  "expiresIn": 3600
}
```

**Response:**
```json
{
  "publishUrl": "rtmp://localhost:1935/live/my-stream?token=abc123...",
  "streamKey": "my-stream",
  "token": "abc123...",
  "expiresAt": "2025-10-10T01:00:00Z"
}
```

### List Active Streams
```bash
GET /api/v1/streams
```

**Response:**
```json
{
  "streams": [
    {
      "streamKey": "my-stream",
      "active": true,
      "state": "live",
      "viewers": 42,
      "startedAt": "2025-10-10T00:00:00Z",
      "duration": 3600,
      "videoCodec": "h264",
      "audioCodec": "aac",
      "resolution": "1920x1080",
      "bitrate": 5000000
    }
  ],
  "total": 1
}
```

### Get Stream Info
```bash
GET /api/v1/streams/:streamKey
```

### Stop Stream
```bash
POST /api/v1/streams/:streamKey/stop
```

**Response:**
```json
{
  "message": "stream stopped",
  "streamKey": "my-stream"
}
```

### HLS Playback Endpoints
```bash
GET /live/:streamKey/index.m3u8     # HLS playlist
GET /live/:streamKey/init.mp4        # Initialization segment (fMP4)
GET /live/:streamKey/segment_N.m4s   # Media segment (CMAF fragment)
```

**Example:**
```bash
# Get playlist
curl http://localhost:8080/live/test/index.m3u8

# Get init segment
curl http://localhost:8080/live/test/init.mp4 -o init.mp4

# Get media segment
curl http://localhost:8080/live/test/segment_0.m4s -o segment_0.m4s
```

## 🏗️ Architecture

### Project Structure

```
RapidRTMP/
├── cmd/                    # (future) Multiple binaries
├── config/                 # Configuration management
│   └── config.go
├── httpServer/             # HTTP API server
│   └── httpServer.go
├── internal/               # Private application code
│   ├── auth/              # Authentication & authorization
│   │   └── auth.go
│   ├── remux/             # (TODO) RTMP → fMP4 remuxer
│   ├── rtmp/              # (TODO) RTMP protocol handler
│   ├── segmenter/         # (TODO) HLS segmentation
│   ├── storage/           # Storage abstraction (local/S3)
│   │   └── storage.go
│   ├── streammanager/     # Stream lifecycle & pub/sub
│   │   └── manager.go
│   └── webrtc/            # (TODO) WebRTC gateway
├── pkg/                    # Public/shared code
│   └── models/            # Data models
│       ├── auth.go
│       ├── frame.go
│       ├── segment.go
│       └── stream.go
├── main.go                # Application entry point
├── go.mod
└── README.md
```

### Component Overview

**Stream Manager**
- In-memory registry of active streams
- Pub/sub system for frame distribution
- Thread-safe stream state management
- Stream statistics tracking

**Auth Manager**
- Cryptographically secure token generation
- Token validation and expiration
- Automatic token cleanup

**Storage Layer**
- Abstract interface for storage backends
- Local filesystem implementation
- Designed for S3/Cloud Storage integration

**HTTP Server**
- RESTful API using Gin framework
- Stream management endpoints
- (Future) HLS playlist and segment serving

## 🧪 Testing

### Quick Start - Complete End-to-End Test

```bash
# 1. Start the server
./rapidrtmp

# 2. In a new terminal, start a test stream (2 minutes, 1-second keyframes)
pkill -9 ffmpeg 2>/dev/null
TOKEN=$(curl -s http://localhost:8080/api/v1/publish -H "Content-Type: application/json" -d '{"streamKey":"test"}' | python3 -c 'import sys,json; print(json.load(sys.stdin)["token"])')
nohup ffmpeg -re -t 120 -f lavfi -i testsrc=size=1280x720:rate=30 -f lavfi -i sine=frequency=1000 -pix_fmt yuv420p -profile:v high -level:v 4.1 -g 30 -keyint_min 30 -sc_threshold 0 -c:v libx264 -preset veryfast -b:v 2500k -c:a aac -b:a 128k -f flv "rtmp://localhost:1935/live/test?token=$TOKEN" </dev/null >/tmp/ffmpeg.log 2>&1 &

# 3. Verify stream is live
curl -s http://localhost:8080/api/v1/streams/test | python3 -m json.tool

# 4. Check playlist
curl -s http://localhost:8080/live/test/index.m3u8 | head -15

# 5. Open test-player.html in browser, enter "test" as stream key, click Load Stream
```

### Verify Stream Quality

```bash
# Check stream info
curl -s http://localhost:8080/api/v1/streams/test | python3 -m json.tool

# Monitor playlist updates (should increment every ~1 second)
watch -n 1 'curl -s http://localhost:8080/live/test/index.m3u8 | head -8'

# Check FFmpeg logs
tail -f /tmp/ffmpeg.log
```

### Test with OBS Studio

1. Generate a token via API
2. In OBS: Settings → Stream
3. Service: Custom
4. Server: `rtmp://localhost:1935/live`
5. Stream Key: `your-stream-key?token=YOUR_TOKEN`
6. Click "Start Streaming"
7. **Open `test-player.html` in your browser to watch!**

**See [TESTING.md](TESTING.md) for comprehensive testing guide.**

### Watch HLS Stream in Browser

**Option 1: Built-in Test Player (Recommended)**

Open `test-player.html` in your browser:
```bash
# Serve via Python HTTP server for best results
python3 -m http.server 8888
# Then open: http://localhost:8888/test-player.html
```

Features:
- ✅ Auto cache-busting for fresh content
- ✅ Auto-recovery from buffering
- ✅ Live edge tracking
- ✅ Error reporting with details
- ✅ Status indicators

**Option 2: Direct HLS URL**

Use the playlist URL with any HLS player:
```
http://localhost:8080/live/your-stream-key/index.m3u8
```

Compatible with:
- VLC Media Player
- Safari (native HLS)
- hls.js
- video.js
- JW Player
- Shaka Player

### API Testing

```bash
# Health check
curl http://localhost:8080/api/ping

# Generate publish token
curl -X POST http://localhost:8080/api/v1/publish \
  -H "Content-Type: application/json" \
  -d '{"streamKey":"test","expiresIn":3600}'

# List streams
curl http://localhost:8080/api/v1/streams

# Get stream info
curl http://localhost:8080/api/v1/streams/test

# Stop stream
curl -X POST http://localhost:8080/api/v1/streams/test/stop
```

## 🎯 Roadmap

### Phase 1: Core Infrastructure ✅ **COMPLETE**
- [x] Project structure
- [x] Data models
- [x] Stream manager
- [x] Auth system
- [x] Storage layer
- [x] HTTP API
- [x] Configuration

### Phase 2: RTMP Ingest ✅ **COMPLETE**
- [x] RTMP server (TCP listener on port 1935)
- [x] RTMP protocol parser (using go-rtmp library)
- [x] H.264/AAC frame extraction
- [x] Publisher authentication via tokens
- [x] Integration with stream manager
- [x] Frame pub/sub to downstream consumers

### Phase 3: HLS Packaging ✅ **COMPLETE**
- [x] HLS segmenter (2-second segments, sliding window)
- [x] Automatic segmentation on stream start
- [x] HLS playlist generation (.m3u8)
- [x] Segment serving via HTTP
- [x] Init segment support (init.mp4)
- [x] Proper caching headers (low-latency optimized)
- [x] CORS support for web playback
- [x] Automatic cleanup and retention policy
- [x] Web-based test player (hls.js)

### Phase 4: Production Features ✅ **COMPLETE**
- [x] Prometheus metrics (30+ metrics)
- [x] Structured logging
- [x] Health and readiness endpoints
- [x] Docker containerization
- [x] Kubernetes manifests
- [x] GCS storage backend
- [x] Metrics middleware
- [ ] WebRTC gateway (future)
- [ ] Multi-bitrate transcoding (future)
- [ ] CDN integration (future)
- [ ] AWS S3 storage backend (future)

### Phase 5: Scale & Optimize
- [ ] Horizontal scaling
- [ ] Load balancing
- [ ] DVR/VOD support
- [ ] Admin dashboard
- [ ] Kubernetes deployment

## 🔧 Development

### Adding Dependencies

```bash
go get <package>
go mod tidy
```

### Building for Production

```bash
CGO_ENABLED=0 go build -ldflags="-s -w" -o rapidrtmp
```

### Running Tests (when added)

```bash
go test ./...
```

## 📝 Design Document

This project follows a detailed design document covering:
- RTMP protocol handling
- HLS/CMAF packaging
- Security & authentication
- Scaling strategies
- Observability

See the full design document for implementation details.

## 🤝 Contributing

Contributions welcome! Key areas:
- RTMP protocol implementation
- HLS segmentation
- Performance optimization
- Testing

## 📄 License

[Add your license here]

## 🙏 Acknowledgments

Built with:
- [Gin](https://github.com/gin-gonic/gin) - HTTP framework
- [Pion WebRTC](https://github.com/pion/webrtc) - (planned) WebRTC support
- [QUIC-Go](https://github.com/quic-go/quic-go) - QUIC protocol support