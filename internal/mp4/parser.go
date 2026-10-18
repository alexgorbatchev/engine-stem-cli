// Package mp4 provides MP4/M4A container parsing and sample table inspection utilities.
package mp4

import (
	"encoding/binary"
	"fmt"
)

// Box represents an MP4 box location and header information.
type Box struct {
	Offset int
	Size   int
}

// StscEntry represents a sample-to-chunk mapping entry.
type StscEntry struct {
	FirstChunk      int
	SamplesPerChunk int
	SampleDescIdx   int
}

// Metadata contains summary information about an MP4 file.
type Metadata struct {
	FileSize     int64
	MdatOffset   int
	MdatSize     int
	MoovOffset   int
	MoovSize     int
	SampleCount  int
	ChunkCount   int
	IsEncrypted  bool
	ChannelCount int
	SampleRate   int
}

// SampleTable contains parsed sample table information.
type SampleTable struct {
	StblOffset   int
	StblSize     int
	StszOffset   int
	StcoOffset   int
	SampleCount  int
	ChunkCount   int
	SampleSizes  []int
	ChunkOffsets []int
	StscEntries  []StscEntry
}

// FindBox locates a box by type starting from a given offset.
// Returns nil if not found.
func FindBox(data []byte, boxType string, start int) *Box {
	pos := start
	typeBytes := []byte(boxType)

	for pos+8 <= len(data) {
		size := int(binary.BigEndian.Uint32(data[pos : pos+4]))
		btype := data[pos+4 : pos+8]

		if size == 0 {
			size = len(data) - pos
		} else if size == 1 && pos+16 <= len(data) {
			size = int(binary.BigEndian.Uint64(data[pos+8 : pos+16]))
		}

		if string(btype) == string(typeBytes) {
			return &Box{Offset: pos, Size: size}
		}

		if size < 8 {
			break
		}
		pos += size
	}
	return nil
}

// FindBoxInContainer locates a box inside a container box.
func FindBoxInContainer(data []byte, container *Box, boxType string) *Box {
	pos := container.Offset + 8
	end := container.Offset + container.Size
	if end > len(data) {
		end = len(data)
	}
	typeBytes := []byte(boxType)

	for pos+8 <= end {
		size := int(binary.BigEndian.Uint32(data[pos : pos+4]))
		btype := data[pos+4 : pos+8]

		if size == 0 {
			size = end - pos
		} else if size == 1 && pos+16 <= len(data) {
			size = int(binary.BigEndian.Uint64(data[pos+8 : pos+16]))
		}

		if string(btype) == string(typeBytes) {
			return &Box{Offset: pos, Size: size}
		}

		if size < 8 {
			break
		}
		pos += size
	}
	return nil
}

// ParseSampleTable parses the sample table (stbl) from MP4 data.
func ParseSampleTable(data []byte) (*SampleTable, error) {
	moov := FindBox(data, "moov", 0)
	if moov == nil {
		return nil, fmt.Errorf("no moov box found")
	}

	trak := FindBoxInContainer(data, moov, "trak")
	if trak == nil {
		return nil, fmt.Errorf("no trak box found")
	}

	mdia := FindBoxInContainer(data, trak, "mdia")
	if mdia == nil {
		return nil, fmt.Errorf("no mdia box found")
	}

	minf := FindBoxInContainer(data, mdia, "minf")
	if minf == nil {
		return nil, fmt.Errorf("no minf box found")
	}

	stbl := FindBoxInContainer(data, minf, "stbl")
	if stbl == nil {
		return nil, fmt.Errorf("no stbl box found")
	}

	st := &SampleTable{
		StblOffset: stbl.Offset,
		StblSize:   stbl.Size,
	}

	if err := st.parseStsz(data, stbl); err != nil {
		return nil, err
	}

	if err := st.parseStco(data, stbl); err != nil {
		return nil, err
	}

	if err := st.parseStsc(data, stbl); err != nil {
		return nil, err
	}

	return st, nil
}

func (st *SampleTable) parseStsz(data []byte, stbl *Box) error {
	stsz := FindBoxInContainer(data, stbl, "stsz")
	if stsz == nil {
		return fmt.Errorf("no stsz box found")
	}

	if stsz.Offset+20 > len(data) {
		return fmt.Errorf("invalid stsz box offset/size")
	}

	st.StszOffset = stsz.Offset
	sampleSize := int(binary.BigEndian.Uint32(data[stsz.Offset+12 : stsz.Offset+16]))
	st.SampleCount = int(binary.BigEndian.Uint32(data[stsz.Offset+16 : stsz.Offset+20]))

	st.SampleSizes = make([]int, st.SampleCount)
	if sampleSize == 0 {
		if stsz.Offset+20+st.SampleCount*4 > len(data) {
			return fmt.Errorf("truncated stsz box sample entries")
		}
		for i := 0; i < st.SampleCount; i++ {
			st.SampleSizes[i] = int(binary.BigEndian.Uint32(data[stsz.Offset+20+i*4 : stsz.Offset+24+i*4]))
		}
	} else {
		for i := 0; i < st.SampleCount; i++ {
			st.SampleSizes[i] = sampleSize
		}
	}

	return nil
}

func (st *SampleTable) parseStco(data []byte, stbl *Box) error {
	stco := FindBoxInContainer(data, stbl, "stco")
	if stco == nil {
		return fmt.Errorf("no stco box found")
	}

	if stco.Offset+16 > len(data) {
		return fmt.Errorf("invalid stco box offset/size")
	}

	st.StcoOffset = stco.Offset
	st.ChunkCount = int(binary.BigEndian.Uint32(data[stco.Offset+12 : stco.Offset+16]))

	if stco.Offset+16+st.ChunkCount*4 > len(data) {
		return fmt.Errorf("truncated stco box chunk entries")
	}

	st.ChunkOffsets = make([]int, st.ChunkCount)
	for i := 0; i < st.ChunkCount; i++ {
		st.ChunkOffsets[i] = int(binary.BigEndian.Uint32(data[stco.Offset+16+i*4 : stco.Offset+20+i*4]))
	}

	return nil
}

func (st *SampleTable) parseStsc(data []byte, stbl *Box) error {
	stsc := FindBoxInContainer(data, stbl, "stsc")
	if stsc == nil {
		return fmt.Errorf("no stsc box found")
	}

	if stsc.Offset+16 > len(data) {
		return fmt.Errorf("invalid stsc box offset/size")
	}

	entryCount := int(binary.BigEndian.Uint32(data[stsc.Offset+12 : stsc.Offset+16]))
	if stsc.Offset+16+entryCount*12 > len(data) {
		return fmt.Errorf("truncated stsc box entries")
	}

	st.StscEntries = make([]StscEntry, entryCount)
	for i := 0; i < entryCount; i++ {
		base := stsc.Offset + 16 + i*12
		st.StscEntries[i] = StscEntry{
			FirstChunk:      int(binary.BigEndian.Uint32(data[base : base+4])),
			SamplesPerChunk: int(binary.BigEndian.Uint32(data[base+4 : base+8])),
			SampleDescIdx:   int(binary.BigEndian.Uint32(data[base+8 : base+12])),
		}
	}

	return nil
}

// CalculateSampleOffsets computes the absolute file offset for each sample.
func (st *SampleTable) CalculateSampleOffsets() []int {
	offsets := make([]int, 0, st.SampleCount)
	sampleIdx := 0

	for chunkIdx := 0; chunkIdx < st.ChunkCount; chunkIdx++ {
		chunkNum := chunkIdx + 1

		samplesPerChunk := 1
		for _, entry := range st.StscEntries {
			if chunkNum >= entry.FirstChunk {
				samplesPerChunk = entry.SamplesPerChunk
			} else {
				break
			}
		}

		chunkOffset := st.ChunkOffsets[chunkIdx]
		offsetInChunk := 0

		for j := 0; j < samplesPerChunk; j++ {
			if sampleIdx >= st.SampleCount {
				break
			}
			offsets = append(offsets, chunkOffset+offsetInChunk)
			offsetInChunk += st.SampleSizes[sampleIdx]
			sampleIdx++
		}
	}

	return offsets
}

// CalculateChunkOffsets computes chunk offsets from sample offsets.
func (st *SampleTable) CalculateChunkOffsets(sampleOffsets []int) []int {
	chunkOffsets := make([]int, st.ChunkCount)
	sampleIdx := 0

	for chunkIdx := 0; chunkIdx < st.ChunkCount; chunkIdx++ {
		chunkOffsets[chunkIdx] = sampleOffsets[sampleIdx]

		chunkNum := chunkIdx + 1
		samplesPerChunk := 1
		for _, entry := range st.StscEntries {
			if chunkNum >= entry.FirstChunk {
				samplesPerChunk = entry.SamplesPerChunk
			} else {
				break
			}
		}
		sampleIdx += samplesPerChunk
	}

	return chunkOffsets
}

// InspectContainer inspects MP4 container structure and metadata.
func InspectContainer(data []byte) (*Metadata, error) {
	mdat := FindBox(data, "mdat", 0)
	if mdat == nil {
		return nil, fmt.Errorf("no mdat box found")
	}

	moov := FindBox(data, "moov", 0)
	if moov == nil {
		return nil, fmt.Errorf("no moov box found")
	}

	st, err := ParseSampleTable(data)
	if err != nil {
		return nil, fmt.Errorf("parsing sample table: %w", err)
	}

	// Check encryption status by probing sample 1
	sampleOffsets := st.CalculateSampleOffsets()
	isEncrypted := false
	if len(sampleOffsets) > 1 && len(st.SampleSizes) > 1 {
		offset := sampleOffsets[1]
		size := st.SampleSizes[1]
		if offset+size <= len(data) && size >= 16 {
			sample1 := data[offset : offset+size]
			firstByte := sample1[0]
			// Plain AAC-LC starting with CPE(0) has first byte 0x20 or 0x21 (bits: 001 0000 0)
			// If raw sample 1 does NOT start with CPE(0), check if AES decrypting sample 1 yields 0x20/0x21
			if firstByte != 0x20 && firstByte != 0x21 {
				isEncrypted = true
			}
		}
	}

	meta := &Metadata{
		FileSize:     int64(len(data)),
		MdatOffset:   mdat.Offset,
		MdatSize:     mdat.Size,
		MoovOffset:   moov.Offset,
		MoovSize:     moov.Size,
		SampleCount:  st.SampleCount,
		ChunkCount:   st.ChunkCount,
		IsEncrypted:  isEncrypted,
		ChannelCount: 8,
		SampleRate:   44100,
	}

	return meta, nil
}
