# RapidRTMP

A high-performance RTMP streaming server written in Go, designed to accept live video streams from publishers (OBS, FFmpeg) and deliver them to viewers via HLS.

## 🚀 Current Status

**Phase 1: Core Infrastructure** ✅ **COMPLETE**

The following components are implemented and working:

- ✅ **HTTP API Server** - REST API for stream management
- ✅ **Stream Manager** - In-memory stream registry with pub/sub
- ✅ **Auth Manager** - Token-based authentication for publishers
- ✅ **Storage Layer** - Local filesystem storage (S3 ready)
- ✅ **Data Models** - Complete type system for streams, frames, segments
- ✅ **Configuration** - Environment-based config management

**What's Working:**
- Generate publish tokens via API
- List active streams
- Get stream metadata
- Stop streams remotely
- Health check endpoint

**What's Next (Phase 2):**
- 🔨 RTMP Ingest Server (accept streams from OBS/FFmpeg)
- 🔨 Remuxer (RTMP → fMP4/CMAF conversion)
- 🔨 HLS Segmenter (generate playlists and segments)
- 🔨 HLS Playback (serve to viewers)

## 📋 Requirements

- **Go 1.24.3+**
- No external dependencies needed for building (Go modules handles everything)

## 🛠️ Installation

### Clone and Build

```bash
git clone https://github.com/yourusername/RapidRTMP.git
cd RapidRTMP
go build
```

### Run

```bash
./rapidrtmp
```

The server will start on `http://localhost:8080` by default.

### Configuration

Configure via environment variables:

```bash
# HTTP Server
export HTTP_ADDR=":8080"

# RTMP Server (not yet implemented)
export RTMP_ADDR=":1935"
export RTMP_INGEST_ADDR="rtmp://localhost:1935"

# Storage
export STORAGE_DIR="./data/streams"

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

### HLS Endpoints (Not Yet Implemented)
```bash
GET /live/:streamKey/index.m3u8  # HLS playlist
GET /live/:streamKey/init.mp4    # Initialization segment
GET /live/:streamKey/:segment.m4s # Media segment
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

### Test the API

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

### Phase 2: RTMP Ingest (In Progress)
- [ ] RTMP server (TCP listener)
- [ ] RTMP protocol parser
- [ ] H.264/AAC frame extraction
- [ ] Publisher authentication
- [ ] Integration with stream manager

### Phase 3: HLS Packaging
- [ ] Remuxer (RTMP → fMP4)
- [ ] CMAF/fMP4 segmenter
- [ ] HLS playlist generation
- [ ] Segment serving via HTTP
- [ ] Proper caching headers

### Phase 4: Production Features
- [ ] Prometheus metrics
- [ ] Structured logging
- [ ] WebRTC gateway (optional)
- [ ] Multi-bitrate transcoding
- [ ] CDN integration
- [ ] S3 storage backend

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