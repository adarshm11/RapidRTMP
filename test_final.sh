#!/bin/bash

echo "🎬 Final Test - Init Segment Fix"
echo "=================================="
echo ""

# Kill old processes
pkill -9 ffmpeg 2>/dev/null
sleep 1

# Clean old data
rm -rf data/streams/demo 2>/dev/null
mkdir -p data/streams/demo

# Generate token
echo "1. Generating token..."
TOKEN=$(curl -s http://localhost:8080/api/v1/publish \
  -H "Content-Type: application/json" \
  -d '{"streamKey":"demo"}' | python3 -c "import sys, json; print(json.load(sys.stdin)['token'])")

if [ -z "$TOKEN" ]; then
  echo "❌ Failed to generate token"
  exit 1
fi

echo "✅ Token: ${TOKEN:0:20}..."
echo ""

# Start streaming
echo "2. Starting FFmpeg stream (10 seconds)..."
ffmpeg -re -t 10 -f lavfi -i testsrc=size=1280x720:rate=30 \
  -f lavfi -i sine=frequency=1000 \
  -c:v libx264 -preset veryfast -b:v 2500k \
  -c:a aac -b:a 128k \
  -f flv "rtmp://localhost:1935/live/demo?token=$TOKEN" \
  > /tmp/ffmpeg_test.log 2>&1

echo "✅ FFmpeg finished"
echo ""

# Check results
echo "3. Checking results..."
echo ""

if [ -f "data/streams/demo/init.mp4" ]; then
  INIT_SIZE=$(ls -lh data/streams/demo/init.mp4 | awk '{print $5}')
  echo "✅ Init segment created: $INIT_SIZE"
  
  # Check if it's a real MP4
  if file data/streams/demo/init.mp4 | grep -q "ISO Media"; then
    echo "   ✅ Valid MP4 file!"
  else
    echo "   ❌ Not a valid MP4 ($(file data/streams/demo/init.mp4 | cut -d: -f2))"
  fi
else
  echo "❌ No init segment created"
fi

echo ""

# Check segments
SEG_COUNT=$(ls data/streams/demo/segment_*.m4s 2>/dev/null | wc -l)
echo "✅ Created $SEG_COUNT media segments"

if [ $SEG_COUNT -gt 0 ]; then
  FIRST_SEG=$(ls data/streams/demo/segment_0.m4s 2>/dev/null)
  if [ -n "$FIRST_SEG" ]; then
    SEG_SIZE=$(ls -lh "$FIRST_SEG" | awk '{print $5}')
    echo "   First segment: $SEG_SIZE"
    
    # Check if it's a real MP4
    if file "$FIRST_SEG" | grep -q "ISO Media"; then
      echo "   ✅ Valid MP4 segment!"
    else
      echo "   ❌ Not valid MP4 ($(file "$FIRST_SEG" | cut -d: -f2))"
    fi
  fi
fi

echo ""
echo "4. Server logs (last 30 lines):"
echo "---"
tail -30 /tmp/rapidrtmp.log | grep -E "SPS|PPS|init|segment 0|Using keyframe|Created init"
echo "---"
echo ""

echo "5. Test playback:"
echo "   http://localhost:8888/test-player.html"
echo "   Stream key: demo"
echo ""

