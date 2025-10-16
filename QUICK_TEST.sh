#!/bin/bash

echo "🎬 RapidRTMP - Quick Test Script with SPS/PPS Fix"
echo "================================================="
echo ""

# Colors
GREEN='\033[0;32m'
BLUE='\033[0;34m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

# Generate token
echo -e "${BLUE}1. Generating publish token...${NC}"
TOKEN_RESPONSE=$(curl -s http://localhost:8080/api/v1/publish \
  -H "Content-Type: application/json" \
  -d '{"streamKey":"test"}')

TOKEN=$(echo $TOKEN_RESPONSE | python3 -c "import sys, json; print(json.load(sys.stdin)['token'])" 2>/dev/null)

if [ -z "$TOKEN" ]; then
  echo -e "${YELLOW}❌ Failed to generate token. Is server running?${NC}"
  echo "   Start server with: ./rapidrtmp"
  exit 1
fi

echo -e "${GREEN}✅ Token generated!${NC}"
echo ""

# Display FFmpeg command
echo -e "${BLUE}2. Start streaming with this command:${NC}"
echo ""
echo "ffmpeg -re -f lavfi -i testsrc=size=1280x720:rate=30 \\"
echo "  -f lavfi -i sine=frequency=1000 \\"
echo "  -c:v libx264 -preset veryfast -b:v 2500k \\"
echo "  -c:a aac -b:a 128k \\"
echo "  -f flv \"rtmp://localhost:1935/live/test?token=$TOKEN\""
echo ""

# Display monitoring commands
echo -e "${BLUE}3. Monitor SPS/PPS extraction (in another terminal):${NC}"
echo ""
echo "tail -f /tmp/rapidrtmp.log | grep -E \"SPS|PPS|sequence|Prepended\""
echo ""

# Display playback URL
echo -e "${BLUE}4. Watch the stream:${NC}"
echo ""
echo "  Browser: http://localhost:8888/test-player.html"
echo "  Stream Key: test"
echo ""

# Display what to look for
echo -e "${YELLOW}📋 What to Look For:${NC}"
echo ""
echo "Server logs should show:"
echo "  ✅ \"Received AVC sequence header for stream test\""
echo "  ✅ \"Parsed AVCDecoderConfigurationRecord: ... SPS count=1, PPS count=1\""
echo "  ✅ \"Stored SPS/PPS for stream test\""
echo "  ✅ \"Prepended SPS/PPS to keyframe\" (every 2-3 seconds)"
echo "  ✅ \"Created segment X for stream test\""
echo ""
echo "Browser should show:"
echo "  ✅ Status: Playing"
echo "  ✅ Video appears (colorful test pattern)"
echo "  ✅ No fragParsingError"
echo "  ✅ No bufferAppendError"
echo ""

echo -e "${GREEN}🚀 Ready to test!${NC}"
echo ""
echo "Press Ctrl+C after starting FFmpeg to stop."
echo ""

# Ask if user wants to auto-start FFmpeg
read -p "Start FFmpeg streaming now? (y/n) " -n 1 -r
echo ""
if [[ $REPLY =~ ^[Yy]$ ]]; then
  echo -e "${BLUE}Starting FFmpeg...${NC}"
  echo ""
  ffmpeg -re -f lavfi -i testsrc=size=1280x720:rate=30 \
    -f lavfi -i sine=frequency=1000 \
    -c:v libx264 -preset veryfast -b:v 2500k \
    -c:a aac -b:a 128k \
    -f flv "rtmp://localhost:1935/live/test?token=$TOKEN"
fi

