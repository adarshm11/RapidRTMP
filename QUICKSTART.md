# 🚀 Quick Start Guide - Testing Your Server

## ✅ Server Status: **WORKING!**

Your RapidRTMP server is running successfully on:
- **HTTP API**: http://localhost:8080
- **RTMP Ingest**: rtmp://localhost:1935
- **Metrics**: http://localhost:8080/metrics
- **Health**: http://localhost:8080/health

---

## 📋 Quick Tests

### 1. Check Server Health

```bash
# Health check
curl http://localhost:8080/health

# Readiness check
curl http://localhost:8080/ready

# Ping
curl http://localhost:8080/api/ping
```

**Expected Response:**
```json
{
  "status": "healthy",
  "time": 1760569630
}
```

---

### 2. Generate a Publish Token

```bash
# Create token for a stream
curl -X POST http://localhost:8080/api/v1/publish \
  -H "Content-Type: application/json" \
  -d '{
    "streamKey": "my-live-stream"
  }'
```

**Expected Response:**
```json
{
  "data": {
    "streamKey": "my-live-stream",
    "token": "tok_abc123...",
    "publishUrl": "rtmp://localhost:1935/live",
    "streamUrl": "my-live-stream?token=tok_abc123...",
    "playbackUrl": "http://localhost:8080/live/my-live-stream/index.m3u8",
    "expiresAt": "2025-10-15T17:06:39Z"
  }
}
```

**Save the token!** You'll need it to publish.

---

### 3. Start Streaming with FFmpeg

```bash
# First, get your token from step 2
TOKEN="your-token-here"

# Test with a video file
ffmpeg -re -i your-video.mp4 \
  -c:v libx264 -preset veryfast -b:v 2500k \
  -c:a aac -b:a 128k \
  -f flv "rtmp://localhost:1935/live/my-live-stream?token=$TOKEN"

# OR generate test pattern (no video file needed)
ffmpeg -re -f lavfi -i testsrc=size=1280x720:rate=30 \
  -f lavfi -i sine=frequency=1000:sample_rate=44100 \
  -c:v libx264 -preset veryfast -b:v 2500k \
  -c:a aac -b:a 128k \
  -f flv "rtmp://localhost:1935/live/my-live-stream?token=$TOKEN"
```

---

### 4. Check Active Streams

```bash
# List all streams
curl http://localhost:8080/api/v1/streams

# Get specific stream info
curl http://localhost:8080/api/v1/streams/my-live-stream
```

**Expected Response:**
```json
{
  "streams": [
    {
      "streamKey": "my-live-stream",
      "status": "live",
      "startedAt": "2025-10-15T16:10:00Z",
      "viewers": 0,
      "framesReceived": 1523,
      "droppedFrames": 0
    }
  ],
  "total": 1
}
```

---

### 5. Watch the Stream

#### Option A: Web Browser (Easiest)

Open `test-player.html` in your browser:
```bash
open test-player.html
```

Enter your stream URL:
```
http://localhost:8080/live/my-live-stream/index.m3u8
```

#### Option B: VLC Media Player

```bash
vlc http://localhost:8080/live/my-live-stream/index.m3u8
```

#### Option C: FFplay (Command Line)

```bash
ffplay http://localhost:8080/live/my-live-stream/index.m3u8
```

---

### 6. View Metrics

```bash
# View Prometheus metrics
curl http://localhost:8080/metrics | grep rapidrtmp

# Key metrics to watch:
# - rapidrtmp_active_streams
# - rapidrtmp_frames_received_total
# - rapidrtmp_frames_dropped_total
# - rapidrtmp_http_requests_total
```

---

### 7. Stop a Stream

```bash
curl -X POST http://localhost:8080/api/v1/streams/my-live-stream/stop
```

---

## 🎯 Complete End-to-End Test

Run this complete workflow:

```bash
#!/bin/bash

echo "Step 1: Generate token"
RESPONSE=$(curl -s -X POST http://localhost:8080/api/v1/publish \
  -H "Content-Type: application/json" \
  -d '{"streamKey":"test123"}')

echo "$RESPONSE" | python3 -m json.tool

# Extract token (requires jq or python)
TOKEN=$(echo "$RESPONSE" | python3 -c "import sys, json; print(json.load(sys.stdin)['data']['token'])")

echo ""
echo "Step 2: Your RTMP URL:"
echo "rtmp://localhost:1935/live/test123?token=$TOKEN"
echo ""
echo "Step 3: Your Playback URL:"
echo "http://localhost:8080/live/test123/index.m3u8"
echo ""
echo "Step 4: Start streaming with OBS or FFmpeg"
echo "Step 5: Open test-player.html in browser"
```

---

## 📊 Check Server Logs

```bash
# View live logs
tail -f /tmp/rapidrtmp.log

# Check for errors
grep -i error /tmp/rapidrtmp.log

# Check RTMP connections
grep -i rtmp /tmp/rapidrtmp.log
```

---

## 🔧 Troubleshooting

### Server won't start?

```bash
# Check if ports are in use
lsof -i :8080
lsof -i :1935

# Kill existing processes
pkill -f rapidrtmp

# Restart
./rapidrtmp
```

### Can't connect with OBS/FFmpeg?

1. **Check token**: Make sure it's not expired (1 hour default)
2. **Check URL format**: `rtmp://localhost:1935/live/STREAMKEY?token=TOKEN`
3. **Check firewall**: Ensure port 1935 is open

### Stream won't play?

1. **Wait 4-6 seconds** after starting to publish (segments need to generate)
2. **Check stream is active**: `curl http://localhost:8080/api/v1/streams`
3. **Check HLS files**: `ls -la data/streams/STREAMKEY/`

---

## 🎉 Success Checklist

- ✅ Server starts without errors
- ✅ Health checks return 200 OK
- ✅ Can generate publish tokens
- ✅ Can publish RTMP stream
- ✅ Can view stream in browser/VLC
- ✅ Metrics endpoint works
- ✅ Can list active streams

---

## 🚀 Next Steps

1. **Try with OBS Studio**: See [TESTING.md](TESTING.md)
2. **Set up monitoring**: Configure Prometheus
3. **Deploy with Docker**: `docker build -t rapidrtmp .`
4. **Configure GCS**: See [docs/GCS_SETUP.md](docs/GCS_SETUP.md)

---

## 💡 Pro Tips

1. **Use test pattern for testing** (no video file needed):
   ```bash
   ffmpeg -re -f lavfi -i testsrc -f lavfi -i sine \
     -c:v libx264 -c:a aac -f flv "rtmp://localhost:1935/live/test?token=$TOKEN"
   ```

2. **Monitor in real-time**:
   ```bash
   watch -n 1 'curl -s http://localhost:8080/api/v1/streams | python3 -m json.tool'
   ```

3. **Check HLS segments**:
   ```bash
   watch -n 1 'ls -lh data/streams/*/segments/'
   ```

---

## 📞 Need Help?

Check these files:
- `README.md` - Full documentation
- `TESTING.md` - OBS/FFmpeg setup
- `docs/GCS_SETUP.md` - Cloud storage
- `/tmp/rapidrtmp.log` - Server logs

**Your server is working! Happy streaming! 🎥✨**
