#!/bin/bash

echo "=========================================="
echo "🎬 RapidRTMP End-to-End Test"
echo "=========================================="
echo ""

# Generate token
echo "📝 Step 1: Generate Publish Token"
RESPONSE=$(curl -s -X POST http://localhost:8080/api/v1/publish \
  -H "Content-Type: application/json" \
  -d '{"streamKey":"test123"}')

echo "$RESPONSE" | python3 -m json.tool
echo ""

# Extract token
TOKEN=$(echo "$RESPONSE" | python3 -c "import sys, json; data = json.load(sys.stdin); print(data.get('token', data.get('data', {}).get('token', '')))")

if [ -z "$TOKEN" ]; then
    echo "❌ Failed to get token"
    exit 1
fi

echo "✅ Token: $TOKEN"
echo ""

# Show URLs
echo "=========================================="
echo "📡 Your Streaming Setup:"
echo "=========================================="
echo ""
echo "RTMP Publish URL:"
echo "  rtmp://localhost:1935/live/test123?token=$TOKEN"
echo ""
echo "HLS Playback URL:"
echo "  http://localhost:8080/live/test123/index.m3u8"
echo ""
echo "Stream Key: test123"
echo "Token: $TOKEN"
echo ""

# Check current streams
echo "=========================================="
echo "📊 Current Active Streams:"
echo "=========================================="
curl -s http://localhost:8080/api/v1/streams | python3 -m json.tool
echo ""

# Show how to test
echo "=========================================="
echo "🧪 How to Test:"
echo "=========================================="
echo ""
echo "Option 1: FFmpeg Test Pattern (no video file needed)"
echo "----------------------------------------------"
echo "ffmpeg -re -f lavfi -i testsrc=size=1280x720:rate=30 \\"
echo "  -f lavfi -i sine=frequency=1000 \\"
echo "  -c:v libx264 -preset veryfast -b:v 2500k \\"
echo "  -c:a aac -b:a 128k \\"
echo "  -f flv \"rtmp://localhost:1935/live/test123?token=$TOKEN\""
echo ""
echo "Option 2: OBS Studio"
echo "----------------------------------------------"
echo "Server: rtmp://localhost:1935/live"
echo "Stream Key: test123?token=$TOKEN"
echo ""
echo "Option 3: Watch in Browser"
echo "----------------------------------------------"
echo "open test-player.html"
echo "Enter URL: http://localhost:8080/live/test123/index.m3u8"
echo ""

echo "=========================================="
echo "✅ Server is ready for streaming!"
echo "=========================================="
