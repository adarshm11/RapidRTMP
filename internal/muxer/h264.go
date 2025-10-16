package muxer

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"log"
)

// H.264 NAL unit types
const (
	NALUnitTypeSPS = 7
	NALUnitTypePPS = 8
	NALUnitTypeIDR = 5
)

// AnnexB start codes
var (
	// 4-byte start code (used for first NAL or after SPS/PPS)
	StartCode4 = []byte{0x00, 0x00, 0x00, 0x01}
	// 3-byte start code (used for most NALs)
	StartCode3 = []byte{0x00, 0x00, 0x01}
)

// ConvertAVCCToAnnexB converts H.264 from AVCC format (length-prefixed NAL units)
// to Annex-B format (start-code-prefixed NAL units).
//
// AVCC format (used by RTMP/FLV/MP4):
//
//	[4-byte length][NAL unit][4-byte length][NAL unit]...
//
// Annex-B format (used by raw H.264 streams, MPEG-TS):
//
//	[0x00 0x00 0x00 0x01][NAL unit][0x00 0x00 0x00 0x01][NAL unit]...
func ConvertAVCCToAnnexB(avccData []byte) ([]byte, error) {
	if len(avccData) == 0 {
		return nil, fmt.Errorf("empty AVCC data")
	}

	var annexB bytes.Buffer
	offset := 0
	nalCount := 0

	for offset < len(avccData) {
		// Need at least 4 bytes for length prefix
		if offset+4 > len(avccData) {
			break
		}

		// Read 4-byte length prefix (big-endian)
		nalSize := binary.BigEndian.Uint32(avccData[offset : offset+4])
		offset += 4

		// Validate NAL size
		if nalSize == 0 {
			log.Printf("Warning: Zero-length NAL unit at offset %d", offset-4)
			continue
		}

		if offset+int(nalSize) > len(avccData) {
			return nil, fmt.Errorf("invalid NAL size %d at offset %d (exceeds buffer)", nalSize, offset-4)
		}

		// Get NAL unit data
		nalUnit := avccData[offset : offset+int(nalSize)]
		offset += int(nalSize)

		// Get NAL unit type (lower 5 bits of first byte)
		nalType := nalUnit[0] & 0x1F

		// Write start code (use 4-byte for SPS/PPS/IDR, 3-byte for others)
		if nalType == NALUnitTypeSPS || nalType == NALUnitTypePPS || nalType == NALUnitTypeIDR {
			annexB.Write(StartCode4)
		} else {
			annexB.Write(StartCode3)
		}

		// Write NAL unit
		annexB.Write(nalUnit)
		nalCount++
	}

	if nalCount == 0 {
		return nil, fmt.Errorf("no NAL units found in AVCC data")
	}

	result := annexB.Bytes()
	log.Printf("Converted AVCC to Annex-B: %d bytes -> %d bytes (%d NAL units)",
		len(avccData), len(result), nalCount)

	return result, nil
}

// IsAVCCFormat detects if data is in AVCC format by checking for length prefix
func IsAVCCFormat(data []byte) bool {
	if len(data) < 5 {
		return false
	}

	// Check if first 4 bytes look like a reasonable NAL length
	nalSize := binary.BigEndian.Uint32(data[0:4])

	// NAL size should be reasonable (less than total data size)
	// and the 5th byte should be a valid NAL unit header
	if nalSize > 0 && nalSize < uint32(len(data)) {
		nalHeader := data[4]
		// Check if 5th byte looks like a NAL unit header
		// NAL header format: forbidden_zero_bit(1) + nal_ref_idc(2) + nal_unit_type(5)
		forbiddenBit := (nalHeader >> 7) & 0x01
		nalType := nalHeader & 0x1F

		// Forbidden bit must be 0, and NAL type should be valid (1-21 for H.264)
		return forbiddenBit == 0 && nalType >= 1 && nalType <= 21
	}

	return false
}

// IsAnnexBFormat detects if data is in Annex-B format by checking for start codes
func IsAnnexBFormat(data []byte) bool {
	if len(data) < 4 {
		return false
	}

	// Check for 4-byte start code
	if bytes.Equal(data[0:4], StartCode4) {
		return true
	}

	// Check for 3-byte start code
	if len(data) >= 3 && bytes.Equal(data[0:3], StartCode3) {
		return true
	}

	return false
}

// ExtractSPSandPPS extracts SPS and PPS NAL units from AVCC or Annex-B data
func ExtractSPSandPPS(data []byte) (sps, pps []byte, err error) {
	// Try to convert if in AVCC format
	annexBData := data
	if IsAVCCFormat(data) {
		annexBData, err = ConvertAVCCToAnnexB(data)
		if err != nil {
			return nil, nil, fmt.Errorf("failed to convert to Annex-B: %w", err)
		}
	}

	// Parse Annex-B data
	offset := 0
	for offset < len(annexBData) {
		// Find start code
		startCodeLen := 0
		if offset+4 <= len(annexBData) && bytes.Equal(annexBData[offset:offset+4], StartCode4) {
			startCodeLen = 4
		} else if offset+3 <= len(annexBData) && bytes.Equal(annexBData[offset:offset+3], StartCode3) {
			startCodeLen = 3
		} else {
			offset++
			continue
		}

		// Skip start code
		offset += startCodeLen

		if offset >= len(annexBData) {
			break
		}

		// Find next start code to get NAL length
		nextStart := offset + 1
		for nextStart < len(annexBData) {
			if (nextStart+4 <= len(annexBData) && bytes.Equal(annexBData[nextStart:nextStart+4], StartCode4)) ||
				(nextStart+3 <= len(annexBData) && bytes.Equal(annexBData[nextStart:nextStart+3], StartCode3)) {
				break
			}
			nextStart++
		}

		// Extract NAL unit
		nalUnit := annexBData[offset:nextStart]
		if len(nalUnit) > 0 {
			nalType := nalUnit[0] & 0x1F

			if nalType == NALUnitTypeSPS && sps == nil {
				sps = append(StartCode4, nalUnit...)
				log.Printf("Found SPS: %d bytes", len(nalUnit))
			} else if nalType == NALUnitTypePPS && pps == nil {
				pps = append(StartCode4, nalUnit...)
				log.Printf("Found PPS: %d bytes", len(nalUnit))
			}

			// If we found both, we're done
			if sps != nil && pps != nil {
				return sps, pps, nil
			}
		}

		offset = nextStart
	}

	if sps == nil && pps == nil {
		return nil, nil, fmt.Errorf("no SPS or PPS found in data")
	}

	return sps, pps, nil
}

// GetNALUnitType returns the type of the first NAL unit in the data
func GetNALUnitType(data []byte) (nalType uint8, err error) {
	if IsAVCCFormat(data) {
		if len(data) < 5 {
			return 0, fmt.Errorf("data too short for AVCC format")
		}
		return data[4] & 0x1F, nil
	}

	if IsAnnexBFormat(data) {
		startCodeLen := 4
		if bytes.Equal(data[0:3], StartCode3) {
			startCodeLen = 3
		}
		if len(data) <= startCodeLen {
			return 0, fmt.Errorf("data too short after start code")
		}
		return data[startCodeLen] & 0x1F, nil
	}

	return 0, fmt.Errorf("unknown format")
}
