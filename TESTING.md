# Testing RapidRTMP with OBS Studio and FFmpeg

This guide will help you test the RTMP ingest functionality with real streaming software.

## Prerequisites

- RapidRTMP server running (`./rapidrtmp`)
- OBS Studio or FFmpeg installed
- A video file or webcam for testing

## Step 1: Generate a Publish Token

First, generate a publish token via the API:

```bash
curl -X POST http://localhost:8080/api/v1/publish \
  -H "Content-Type: application/json" \
  -d '{"streamKey":"my-test-stream","expiresIn":7200}'
```

**Response:**
```json
{
  "publishUrl": "rtmp://localhost:1935/live/my-test-stream?token=abc123...",
  "streamKey": "my-test-stream",
  "token": "abc123...",
  "expiresAt": "2025-10-10T02:00:00Z"
}
```

**Save the `publishUrl` - you'll need it!**

---

## Option 1: Test with OBS Studio

### Configure OBS

1. **Open OBS Studio**

2. **Go to Settings → Stream**

3. **Select "Custom..." as Service**

4. **Enter Server and Stream Key:**
   - **Server:** `rtmp://localhost:1935/live`
   - **Stream Key:** `my-test-stream?token=YOUR_TOKEN_HERE`
   
   *(Replace `YOUR_TOKEN_HERE` with the token from Step 1)*

5. **Click Apply, then OK**

### Start Streaming

1. Add a **Video Source** (webcam, screen capture, or video file)

2. Click **Start Streaming**

3. **Check the server logs:**
   ```bash
   tail -f /tmp/rapidrtmp.log
   ```

4. **Verify stream is live:**
   ```bash
   curl http://localhost:8080/api/v1/streams
   ```

### Expected Output

You should see logs like:
```
New RTMP connection from 127.0.0.1:xxxxx
OnConnect: app=live
OnPublish: publishingName=my-test-stream?token=...
Token validated successfully for stream my-test-stream
Stream my-test-stream is now live from 127.0.0.1:xxxxx
```

---

## Option 2: Test with FFmpeg

### Using a Video File

```bash
# Replace YOUR_TOKEN with the token from Step 1
ffmpeg -re -i input.mp4 \
  -c:v libx264 -preset veryfast -b:v 2500k -maxrate 2500k -bufsize 5000k \
  -c:a aac -b:a 128k \
  -f flv "rtmp://localhost:1935/live/my-test-stream?token=YOUR_TOKEN"
```

### Using Webcam (macOS)

```bash
# List available devices
ffmpeg -f avfoundation -list_devices true -i ""

# Stream from webcam (adjust device index)
ffmpeg -f avfoundation -i "0:0" \
  -c:v libx264 -preset veryfast -b:v 2500k \
  -c:a aac -b:a 128k \
  -f flv "rtmp://localhost:1935/live/my-test-stream?token=YOUR_TOKEN"
```

### Using Test Pattern

```bash
# Generate a test pattern (no external video needed)
ffmpeg -f lavfi -i testsrc=size=1280x720:rate=30 \
  -f lavfi -i sine=frequency=1000:sample_rate=48000 \
  -c:v libx264 -preset veryfast -b:v 2500k \
  -c:a aac -b:a 128k \
  -f flv "rtmp://localhost:1935/live/my-test-stream?token=YOUR_TOKEN"
```

---

## Step 2: Verify Stream Status

### Check Active Streams

```bash
curl http://localhost:8080/api/v1/streams | python3 -m json.tool
```

**Expected Response:**
```json
{
  "streams": [
    {
      "streamKey": "my-test-stream",
      "active": true,
      "state": "live",
      "viewers": 0,
      "startedAt": "2025-10-10T00:00:00Z",
      "duration": 120,
      "videoCodec": "h264",
      "audioCodec": "aac"
    }
  ],
  "total": 1
}
```

### Get Specific Stream Info

```bash
curl http://localhost:8080/api/v1/streams/my-test-stream | python3 -m json.tool
```

### Stop Stream Remotely

```bash
curl -X POST http://localhost:8080/api/v1/streams/my-test-stream/stop
```

---

## Step 3: Monitor Server Logs

Watch real-time logs:

```bash
tail -f /tmp/rapidrtmp.log
```

You should see:
- **Connection events**: When OBS/FFmpeg connects
- **Authentication**: Token validation success/failure
- **Frame reception**: Audio and video frames being received
- **Stream state**: Stream going live

---

## Troubleshooting

### "Connection refused" or "Failed to connect"

- ✅ Check server is running: `ps aux | grep rapidrtmp`
- ✅ Check port 1935 is open: `lsof -i :1935`
- ✅ Verify server logs: `tail -f /tmp/rapidrtmp.log`

### "Authentication failed"

- ✅ Generate a fresh token
- ✅ Make sure token is included in stream key: `streamkey?token=xxx`
- ✅ Token hasn't expired (check `expiresAt` field)

### Stream connects but no frames received

- ✅ Check codec settings (should be H.264 video, AAC audio)
- ✅ Verify bitrate isn't too high
- ✅ Check FFmpeg command output for errors

### OBS can't connect

- ✅ Use "Custom..." service, not a preset
- ✅ Server should be just `rtmp://localhost:1935/live` (without stream key)
- ✅ Put stream key + token in the "Stream Key" field

---

## Performance Testing

### Multiple Streams

Generate tokens for multiple streams and start them simultaneously:

```bash
# Stream 1
curl -X POST http://localhost:8080/api/v1/publish \
  -d '{"streamKey":"stream1","expiresIn":3600}' | jq -r '.publishUrl'

# Stream 2
curl -X POST http://localhost:8080/api/v1/publish \
  -d '{"streamKey":"stream2","expiresIn":3600}' | jq -r '.publishUrl'
```

Then start multiple FFmpeg instances or OBS with different stream keys.

### Check Server Stats

```bash
curl http://localhost:8080/api/v1/streams
```

Look for:
- Total active streams
- Frame counts
- Dropped frames (should be 0 or very low)

---

## Next Steps

Once RTMP ingest is working:

1. **Phase 3:** Implement HLS packaging (remuxer + segmenter)
2. **Add viewer support:** Serve HLS playlists and segments
3. **Test playback:** Use video.js or hls.js in browser
4. **Add transcoding:** Multiple bitrates for ABR streaming

---

## Quick Test Commands

```bash
# 1. Generate token
TOKEN_URL=$(curl -s -X POST http://localhost:8080/api/v1/publish \
  -H "Content-Type: application/json" \
  -d '{"streamKey":"test","expiresIn":3600}' | jq -r '.publishUrl')

echo "Publish URL: $TOKEN_URL"

# 2. Start test stream (test pattern)
ffmpeg -re -f lavfi -i testsrc=size=1280x720:rate=30 \
  -f lavfi -i sine=frequency=1000 \
  -c:v libx264 -preset veryfast -b:v 2500k \
  -c:a aac -b:a 128k \
  -f flv "$TOKEN_URL"

# 3. In another terminal, check status
curl http://localhost:8080/api/v1/streams | python3 -m json.tool
```

---

## Success Criteria

✅ **RTMP Connection:** OBS/FFmpeg connects successfully  
✅ **Authentication:** Token is validated  
✅ **Stream Goes Live:** Status shows "live" in API  
✅ **Frames Received:** Video and audio frames are logged  
✅ **No Dropped Frames:** Stats show minimal/no drops  
✅ **Clean Disconnection:** Stream stops cleanly when publisher disconnects

**If all criteria pass, Phase 2 is complete! 🎉**

