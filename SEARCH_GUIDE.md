# 🔍 Reference Codebase Search Guide

## What We Need to Fix

We need to extract **SPS/PPS (Sequence Parameter Set / Picture Parameter Set)** from RTMP and prepend them to H.264 keyframes before muxing to MP4.

---

## 🎯 What to Search For

### GitHub Search Queries

**Best Starting Points:**

1. **"RTMP to HLS Go"**
   ```
   language:Go "rtmp" "hls" "sps" "pps"
   ```
   Look for: How they extract codec data from RTMP

2. **"go-rtmp SPS PPS"**
   ```
   language:Go "yutopp/go-rtmp" "sps" OR "pps" OR "avc"
   ```
   Look for: How they handle the metadata/codec data callback

3. **"RTMP FLV H264 metadata"**
   ```
   language:Go "flv" "h264" "metadata" "avc"
   ```
   Look for: FLV tag parsing for codec data

4. **"H.264 access unit Go"**
   ```
   language:Go "access unit" "h264" "sps" "pps"
   ```
   Look for: How to build complete access units

---

## 📚 Specific Repositories to Check

### 1. **livego** (Most Similar!)
- **URL:** `https://github.com/gwuhaolin/livego`
- **Look For:**
  - `protocol/rtmp/` directory
  - How they handle `OnMetaData` 
  - Search files for: `"avcDecoderConfigurationRecord"` or `"AVCDecoderConfigurationRecord"`
  - Check `protocol/hls/` for how they create segments

### 2. **monibuca** (Modern, Active)
- **URL:** `https://github.com/Monibuca/engine`
- **Look For:**
  - Plugin architecture for RTMP/HLS
  - `codec` package or module
  - How they parse RTMP metadata

### 3. **go-oryx** (SRS Go Port)
- **URL:** `https://github.com/ossrs/go-oryx`
- **Look For:**
  - RTMP protocol handling
  - FLV demuxing
  - HLS muxing logic

### 4. **joy4** (Multimedia Library)
- **URL:** `https://github.com/nareix/joy4`
- **Look For:**
  - `format/flv/` for FLV parsing
  - `av/avutil/` for codec utilities
  - `format/ts/` or `format/mp4/` for muxing

---

## 🔑 Key Code Patterns to Find

### Pattern 1: Extract SPS/PPS from RTMP Metadata

Look for code like:
```go
// In OnMetaData or OnPublish handler
func (h *Handler) OnSetDataFrame(timestamp uint32, data io.Reader) error {
    // Read the whole metadata
    var metadata rtmp.MetaData
    
    // Look for videocodecid, width, height
    // And most importantly: AVCDecoderConfigurationRecord
}
```

### Pattern 2: Parse AVC Decoder Configuration

Look for:
```go
// Parsing AVCDecoderConfigurationRecord
// This contains SPS and PPS
type AVCDecoderConfigurationRecord struct {
    ConfigurationVersion uint8
    AVCProfileIndication uint8
    ProfileCompatibility uint8
    AVCLevelIndication   uint8
    NALUnitLength        uint8
    SPS                  [][]byte // This is what we need!
    PPS                  [][]byte // This too!
}
```

### Pattern 3: Prepend SPS/PPS to Keyframes

Look for:
```go
// When creating segments
if frame.IsKeyFrame {
    // Prepend SPS
    output.Write([]byte{0x00, 0x00, 0x00, 0x01}) // Start code
    output.Write(sps)
    
    // Prepend PPS
    output.Write([]byte{0x00, 0x00, 0x00, 0x01}) // Start code
    output.Write(pps)
}

// Then write the frame
output.Write(frameData)
```

---

## 📖 Documentation to Search

### FFmpeg Wiki
- **URL:** `https://trac.ffmpeg.org/wiki/`
- Search for: "RTMP", "FLV", "H.264 bitstream"

### RTMP Specification
- Look for: AMF0/AMF3 metadata format
- Focus on: Video codec ID 7 (AVC/H.264)

### H.264 Spec (ISO 14496-15)
- Look for: "AVCDecoderConfigurationRecord"
- Section on: NAL unit types (7=SPS, 8=PPS, 5=IDR)

---

## 🎓 What We'll Learn

From these codebases, we need to understand:

1. **Where RTMP sends SPS/PPS**
   - Usually in the first video packet after connect
   - Or in metadata (`@setDataFrame` message)
   - Packet type: Video with frame type 5 (keyframe) and codec 7 (AVC)

2. **How to Parse It**
   - Read the FLV video tag
   - Extract AVCPacketType (0 = AVC sequence header)
   - Parse the configuration record

3. **How to Use It**
   - Store SPS/PPS globally for the stream
   - Prepend to each segment's first keyframe
   - Convert to Annex-B format (add start codes)

---

## 🛠️ What to Bring Back

When you find relevant code, look for:

### File 1: RTMP Metadata Handler
- How do they receive the AVC configuration?
- Where do they store SPS/PPS?

### File 2: H.264 Parser
- How do they parse AVCDecoderConfigurationRecord?
- How do they convert AVCC to Annex-B?

### File 3: Segment Creator
- How do they start each segment?
- Do they prepend SPS/PPS?
- How do they handle keyframes vs. non-keyframes?

---

## 🚀 Quick Wins to Look For

### Option A: Direct AVCDecoderConfigurationRecord Parser
If you find a ready-made parser, we can use it directly!

### Option B: Complete FLV Demuxer
Some libs have full FLV handling we could import.

### Option C: Example RTMP→HLS Pipeline
Best case: Find a working example we can adapt!

---

## 📝 What to Copy/Note

When you find relevant code:

1. **File path** in the repo
2. **Function/struct names**
3. **Key logic** (screenshot or copy the core algorithm)
4. **Dependencies** they use
5. **License** (make sure it's permissive: MIT, Apache, BSD)

---

## 💡 Specific Files to Check in Repos

Common file names:
- `rtmp_handler.go` or `conn_handler.go`
- `flv.go` or `flv_demuxer.go`
- `h264.go` or `avc.go`
- `hls_muxer.go` or `segment.go`
- `codec.go` or `codec_parser.go`

---

## 🎯 Success Criteria

You've found what we need when you see code that:

✅ Extracts SPS/PPS from RTMP packets  
✅ Stores codec configuration per stream  
✅ Prepends SPS/PPS to keyframes  
✅ Converts AVCC → Annex-B properly  
✅ Creates valid MP4/fMP4 segments  

---

## 📞 Report Back With

Once you find something useful:

1. **Repo name and URL**
2. **Specific file(s)** (e.g., `protocol/rtmp/handler.go`)
3. **Key function** (e.g., `parseAVCDecoderConfig()`)
4. **Brief description** of what it does
5. **License** (so we know if we can use it)

Then we can adapt it to our codebase!

---

**TIP:** Start with `livego` - it's the most similar to what we're building!

