package muxer

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"log"
)

// AVCDecoderConfigurationRecord represents the AVC configuration from FLV/RTMP
// This is sent as the first video packet when a stream starts
type AVCDecoderConfigurationRecord struct {
	ConfigurationVersion uint8
	AVCProfileIndication uint8
	ProfileCompatibility uint8
	AVCLevelIndication   uint8
	NALUnitLength        uint8
	SPS                  [][]byte // Sequence Parameter Sets
	PPS                  [][]byte // Picture Parameter Sets
}

// ParseAVCDecoderConfigurationRecord parses the AVCC structure from FLV video data
// This is called when we receive a video packet with AVCPacketType = 0 (sequence header)
func ParseAVCDecoderConfigurationRecord(data []byte) (*AVCDecoderConfigurationRecord, error) {
	if len(data) < 11 {
		return nil, fmt.Errorf("data too short for AVCDecoderConfigurationRecord: %d bytes", len(data))
	}

	record := &AVCDecoderConfigurationRecord{}
	r := bytes.NewReader(data)

	// Read configuration version (should be 1)
	if err := binary.Read(r, binary.BigEndian, &record.ConfigurationVersion); err != nil {
		return nil, err
	}

	// Read profile indication
	if err := binary.Read(r, binary.BigEndian, &record.AVCProfileIndication); err != nil {
		return nil, err
	}

	// Read profile compatibility
	if err := binary.Read(r, binary.BigEndian, &record.ProfileCompatibility); err != nil {
		return nil, err
	}

	// Read level indication
	if err := binary.Read(r, binary.BigEndian, &record.AVCLevelIndication); err != nil {
		return nil, err
	}

	// Read reserved (6 bits) + length size minus one (2 bits)
	var lengthSizeMinusOne uint8
	if err := binary.Read(r, binary.BigEndian, &lengthSizeMinusOne); err != nil {
		return nil, err
	}
	record.NALUnitLength = (lengthSizeMinusOne & 0x03) + 1 // Extract lower 2 bits and add 1

	// Read reserved (3 bits) + number of SPS (5 bits)
	var numOfSPS uint8
	if err := binary.Read(r, binary.BigEndian, &numOfSPS); err != nil {
		return nil, err
	}
	numOfSPS = numOfSPS & 0x1F // Extract lower 5 bits

	// Read SPS
	record.SPS = make([][]byte, numOfSPS)
	for i := 0; i < int(numOfSPS); i++ {
		var spsLength uint16
		if err := binary.Read(r, binary.BigEndian, &spsLength); err != nil {
			return nil, fmt.Errorf("failed to read SPS length: %w", err)
		}

		sps := make([]byte, spsLength)
		if n, err := r.Read(sps); err != nil || n != int(spsLength) {
			return nil, fmt.Errorf("failed to read SPS data: %w", err)
		}
		record.SPS[i] = sps
	}

	// Read number of PPS
	var numOfPPS uint8
	if err := binary.Read(r, binary.BigEndian, &numOfPPS); err != nil {
		return nil, err
	}

	// Read PPS
	record.PPS = make([][]byte, numOfPPS)
	for i := 0; i < int(numOfPPS); i++ {
		var ppsLength uint16
		if err := binary.Read(r, binary.BigEndian, &ppsLength); err != nil {
			return nil, fmt.Errorf("failed to read PPS length: %w", err)
		}

		pps := make([]byte, ppsLength)
		if n, err := r.Read(pps); err != nil || n != int(ppsLength) {
			return nil, fmt.Errorf("failed to read PPS data: %w", err)
		}
		record.PPS[i] = pps
	}

	log.Printf("Parsed AVCDecoderConfigurationRecord: Profile=%d, Level=%d, NALULength=%d, SPS count=%d, PPS count=%d",
		record.AVCProfileIndication, record.AVCLevelIndication, record.NALUnitLength, len(record.SPS), len(record.PPS))

	return record, nil
}

// ParseFLVVideoPacket extracts codec data and frame type from FLV video packet
// Returns: isSequenceHeader, isKeyFrame, avcData, error
func ParseFLVVideoPacket(data []byte) (isSequenceHeader bool, isKeyFrame bool, avcData []byte, err error) {
	if len(data) < 5 {
		return false, false, nil, fmt.Errorf("video packet too short: %d bytes", len(data))
	}

	// Byte 0: Frame type (4 bits) + Codec ID (4 bits)
	frameType := (data[0] >> 4) & 0x0F
	codecID := data[0] & 0x0F

	// Check if it's H.264/AVC (codec ID 7)
	if codecID != 7 {
		return false, false, nil, fmt.Errorf("not H.264/AVC codec: %d", codecID)
	}

	// Frame types:
	// 1 = keyframe (IDR)
	// 2 = inter frame
	// 3 = disposable inter frame
	isKeyFrame = frameType == 1

	// Byte 1: AVCPacketType
	// 0 = AVC sequence header (contains SPS/PPS)
	// 1 = AVC NALU (actual video data)
	// 2 = AVC end of sequence
	avcPacketType := data[1]
	isSequenceHeader = avcPacketType == 0

	// Bytes 2-4: Composition time (PTS offset)
	// compositionTime := int32(data[2])<<16 | int32(data[3])<<8 | int32(data[4])

	// The actual AVC data starts at byte 5
	avcData = data[5:]

	return isSequenceHeader, isKeyFrame, avcData, nil
}

// PrependSPSPPSAnnexB prepends SPS and PPS to frame data in Annex-B format
func PrependSPSPPSAnnexB(frameData []byte, sps, pps [][]byte) []byte {
	var buf bytes.Buffer
	startCode := []byte{0x00, 0x00, 0x00, 0x01}

	// Write all SPS
	for i, s := range sps {
		buf.Write(startCode)
		buf.Write(s)
		log.Printf("PrependSPSPPSAnnexB: Added SPS[%d] of %d bytes", i, len(s))
	}

	// Write all PPS
	for i, p := range pps {
		buf.Write(startCode)
		buf.Write(p)
		log.Printf("PrependSPSPPSAnnexB: Added PPS[%d] of %d bytes", i, len(p))
	}

	// Write frame data (should already be in Annex-B)
	buf.Write(frameData)
	log.Printf("PrependSPSPPSAnnexB: Total output size: %d bytes (SPS/PPS overhead + %d bytes frame data)", buf.Len(), len(frameData))

	return buf.Bytes()
}

// ConvertAVCCFrameToAnnexB converts an AVCC frame (with the codec configuration) to Annex-B
// This uses the NALUnitLength from the AVCC record to properly parse length-prefixed NALUs
func ConvertAVCCFrameToAnnexB(frameData []byte, naluLength int) ([]byte, error) {
	if naluLength != 4 {
		// For now, we only support 4-byte length prefixes (most common)
		// If needed, we can add support for 1, 2, or 3 byte lengths
		log.Printf("Warning: NALU length size is %d, using default AVCC conversion", naluLength)
		return ConvertAVCCToAnnexB(frameData)
	}

	return ConvertAVCCToAnnexB(frameData)
}
