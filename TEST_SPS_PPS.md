# 🎯 SPS/PPS Implementation - Testing Guide

## What We Fixed

We implemented **proper H.264 SPS/PPS extraction and handling** based on the `livego` codebase reference!

### Key Changes:

1. **Created `internal/muxer/avc_parser.go`**
   - `ParseAVCDecoderConfigurationRecord()` - Extracts SPS/PPS from RTMP metadata
   - `ParseFLVVideoPacket()` - Parses FLV video packets to detect sequence headers
   - `PrependSPSPPSAnnexB()` - Prepends SPS/PPS to keyframes in Annex-B format

2. **Updated `internal/rtmp/server.go`**
   - Added `sps`, `pps`, `naluLength` fields to `ConnHandler`
   - Modified `OnVideo()` to:
     - Detect and parse AVC sequence headers (first packet with SPS/PPS)
     - Store SPS/PPS for the stream
     - Prepend SPS/PPS to every keyframe
     - Convert AVCC frames to Annex-B format

3. **Simplified `internal/muxer/ffmpeg.go`**
   - Removed redundant AVCC conversion (now done upstream)
   - Frames arrive already in correct Annex-B format with SPS/PPS

---

## How It Works

### 1. Stream Start
```
FFmpeg connects → RTMP handshake → OnPublish()
```

### 2. First Video Packet (Sequence Header)
```
RTMP sends AVC sequence header
↓
ParseFLVVideoPacket() detects it's a sequence header
↓
ParseAVCDecoderConfigurationRecord() extracts SPS/PPS
↓
Store SPS/PPS in ConnHandler
↓
Log: "Stored SPS/PPS for stream test: 1 SPS, 1 PPS, NALU length=4"
```

### 3. Subsequent Video Frames
```
For each frame:
  ↓
  ParseFLVVideoPacket() → isKeyFrame, avcData
  ↓
  ConvertAVCCToAnnexB(avcData) → Convert length-prefixed to start codes
  ↓
  If keyframe: PrependSPSPPSAnnexB() → Add SPS + PPS + frame data
  ↓
  Publish to segmenter
  ↓
  FFmpeg muxer → fMP4 segment
```

---

## Testing Steps

### Step 1: Start Streaming

Run this command (generated above):
```bash
ffmpeg -re -f lavfi -i testsrc=size=1280x720:rate=30 \
  -f lavfi -i sine=frequency=1000 \
  -c:v libx264 -preset veryfast -b:v 2500k \
  -c:a aac -b:a 128k \
  -f flv "rtmp://localhost:1935/live/test?token=YOUR_TOKEN"
```

### Step 2: Watch Server Logs

In a new terminal:
```bash
tail -f /tmp/rapidrtmp.log | grep -E "SPS|PPS|sequence|Prepended"
```

**Expected output:**
```
✅ Received AVC sequence header for stream test (XX bytes)
✅ Parsed AVCDecoderConfigurationRecord: Profile=66, Level=30, NALULength=4, SPS count=1, PPS count=1
✅ Stored SPS/PPS for stream test: 1 SPS, 1 PPS, NALU length=4
✅ Prepended SPS/PPS to keyframe for stream test (total size: XXXX bytes)
✅ Prepended SPS/PPS to keyframe for stream test (total size: XXXX bytes)
```

### Step 3: Check Segment Creation

```bash
tail -f /tmp/rapidrtmp.log | grep "Created segment"
```

**Expected output:**
```
✅ 2025/10/15 18:XX:XX Created segment 0 for stream test (XXX frames, XXX.XX KB)
✅ 2025/10/15 18:XX:XX Created segment 1 for stream test (XXX frames, XXX.XX KB)
```

### Step 4: Test Playback

1. Open browser: `http://localhost:8888/test-player.html`
2. Enter stream key: `test`
3. Click "Load Stream"
4. **Watch for video!** 🎥

---

## What to Look For

### ✅ Success Indicators:

1. **Logs show SPS/PPS extraction:**
   ```
   Received AVC sequence header
   Parsed AVCDecoderConfigurationRecord
   Stored SPS/PPS
   ```

2. **Keyframes have SPS/PPS prepended:**
   ```
   Prepended SPS/PPS to keyframe (total size: XXXX bytes)
   ```

3. **No FFmpeg errors:**
   - No more "missing picture in access unit" errors
   - Segments created successfully

4. **Browser playback works:**
   - No `fragParsingError`
   - No `bufferAppendError`
   - Video plays smoothly!

### ❌ Potential Issues:

1. **"Warning: Keyframe received but no SPS/PPS stored"**
   - Means sequence header wasn't received first
   - Solution: Restart stream

2. **"Failed to parse AVCDecoderConfigurationRecord"**
   - Data corruption or wrong format
   - Check FFmpeg encoder settings

3. **Still getting FFmpeg errors:**
   - Check if SPS/PPS are actually being prepended
   - Verify Annex-B conversion is working

---

## Debugging

### Check if SPS/PPS are in segments:

```bash
# Get first segment
curl -s http://localhost:8080/live/test/segment_0.m4s > /tmp/test_segment.m4s

# Look for H.264 start codes (0x00 0x00 0x00 0x01)
xxd /tmp/test_segment.m4s | grep "0000 0001" | head -5
```

**Should see multiple start codes!**

### Check segment file type:

```bash
file /tmp/test_segment.m4s
```

**Should say:** `ISO Media, MP4 Base Media v1` (not just "data")

---

## Technical Details

### FLV Video Packet Format:
```
Byte 0: [Frame Type (4 bits)][Codec ID (4 bits)]
Byte 1: AVCPacketType (0=seq header, 1=NALU, 2=end)
Bytes 2-4: Composition time offset
Bytes 5+: AVC data
```

### AVCDecoderConfigurationRecord Format:
```
configurationVersion: uint8
AVCProfileIndication: uint8
profile_compatibility: uint8
AVCLevelIndication: uint8
lengthSizeMinusOne: uint8 (6 bits reserved + 2 bits)
numOfSequenceParameterSets: uint8 (3 bits reserved + 5 bits)
  for each SPS:
    sequenceParameterSetLength: uint16
    sequenceParameterSetNALUnit: bytes
numOfPictureParameterSets: uint8
  for each PPS:
    pictureParameterSetLength: uint16
    pictureParameterSetNALUnit: bytes
```

### Annex-B Format (What FFmpeg Expects):
```
[0x00 0x00 0x00 0x01][SPS]
[0x00 0x00 0x00 0x01][PPS]
[0x00 0x00 0x00 0x01][IDR frame]
[0x00 0x00 0x00 0x01][P-frame]
...
```

---

## Reference

Based on:
- **livego** (`github.com/gwuhaolin/livego`)
  - `protocol/rtmp/rtmp.go` - VirReader with demuxer
  - `container/flv/demuxer.go` - FLV packet parsing
  - `protocol/rtmp/core/conn_server.go` - RTMP message handling

Key insight from livego:
> The demuxer (`flv.Demuxer`) automatically parses the first video packet 
> to extract codec configuration (SPS/PPS) and stores it for the stream.
> Subsequent packets use this configuration to build complete access units.

---

**Date:** October 15, 2025  
**Status:** SPS/PPS implementation complete, ready for testing!  
**Expected Result:** Browser video playback should work! 🎉

