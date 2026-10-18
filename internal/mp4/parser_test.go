package mp4

import (
	"encoding/binary"
	"strings"
	"testing"
)

func createSyntheticMP4() []byte {
	buf := make([]byte, 0, 1024)

	// ftyp box
	ftyp := make([]byte, 28)
	binary.BigEndian.PutUint32(ftyp[0:4], 28)
	copy(ftyp[4:8], "ftyp")
	copy(ftyp[8:12], "isom")
	buf = append(buf, ftyp...)

	// mdat box header (offset 28)
	mdatHeader := make([]byte, 8)
	binary.BigEndian.PutUint32(mdatHeader[0:4], 8+64) // 64 bytes media data
	copy(mdatHeader[4:8], "mdat")
	buf = append(buf, mdatHeader...)

	// 64 bytes mdat payload (2 samples of 32 bytes)
	mdatPayload := make([]byte, 64)
	buf = append(buf, mdatPayload...)

	// moov container
	moovStart := len(buf)

	// stbl box content
	stsz := make([]byte, 28)
	binary.BigEndian.PutUint32(stsz[0:4], 28)
	copy(stsz[4:8], "stsz")
	binary.BigEndian.PutUint32(stsz[12:16], 0) // variable size
	binary.BigEndian.PutUint32(stsz[16:20], 2) // 2 samples
	binary.BigEndian.PutUint32(stsz[20:24], 32)
	binary.BigEndian.PutUint32(stsz[24:28], 32)

	stco := make([]byte, 20)
	binary.BigEndian.PutUint32(stco[0:4], 20)
	copy(stco[4:8], "stco")
	binary.BigEndian.PutUint32(stco[12:16], 1)  // 1 chunk
	binary.BigEndian.PutUint32(stco[16:20], 36) // chunk offset at mdat payload start

	stsc := make([]byte, 28)
	binary.BigEndian.PutUint32(stsc[0:4], 28)
	copy(stsc[4:8], "stsc")
	binary.BigEndian.PutUint32(stsc[12:16], 1) // 1 entry
	binary.BigEndian.PutUint32(stsc[16:20], 1) // first chunk
	binary.BigEndian.PutUint32(stsc[20:24], 2) // 2 samples per chunk
	binary.BigEndian.PutUint32(stsc[24:28], 1) // sample desc idx

	stblContent := append(stsz, stco...)
	stblContent = append(stblContent, stsc...)

	stbl := make([]byte, 8+len(stblContent))
	binary.BigEndian.PutUint32(stbl[0:4], uint32(len(stbl)))
	copy(stbl[4:8], "stbl")
	copy(stbl[8:], stblContent)

	minfContent := stbl
	minf := make([]byte, 8+len(minfContent))
	binary.BigEndian.PutUint32(minf[0:4], uint32(len(minf)))
	copy(minf[4:8], "minf")
	copy(minf[8:], minfContent)

	mdiaContent := minf
	mdia := make([]byte, 8+len(mdiaContent))
	binary.BigEndian.PutUint32(mdia[0:4], uint32(len(mdia)))
	copy(mdia[4:8], "mdia")
	copy(mdia[8:], mdiaContent)

	trakContent := mdia
	trak := make([]byte, 8+len(trakContent))
	binary.BigEndian.PutUint32(trak[0:4], uint32(len(trak)))
	copy(trak[4:8], "trak")
	copy(trak[8:], trakContent)

	moovContent := trak
	moov := make([]byte, 8+len(moovContent))
	binary.BigEndian.PutUint32(moov[0:4], uint32(len(moov)))
	copy(moov[4:8], "moov")
	copy(moov[8:], moovContent)

	buf = append(buf, moov...)
	_ = moovStart

	return buf
}

func TestFindBoxAndParseSampleTable(t *testing.T) {
	data := createSyntheticMP4()

	moov := FindBox(data, "moov", 0)
	if moov == nil {
		t.Fatalf("FindBox('moov') returned nil")
	}

	mdat := FindBox(data, "mdat", 0)
	if mdat == nil {
		t.Fatalf("FindBox('mdat') returned nil")
	}

	st, err := ParseSampleTable(data)
	if err != nil {
		t.Fatalf("ParseSampleTable error = %v", err)
	}

	if st.SampleCount != 2 {
		t.Errorf("SampleCount = %d, want 2", st.SampleCount)
	}

	if st.ChunkCount != 1 {
		t.Errorf("ChunkCount = %d, want 1", st.ChunkCount)
	}

	offsets := st.CalculateSampleOffsets()
	if len(offsets) != 2 {
		t.Fatalf("CalculateSampleOffsets returned %d offsets, want 2", len(offsets))
	}

	if offsets[0] != 36 {
		t.Errorf("offsets[0] = %d, want 36", offsets[0])
	}
	if offsets[1] != 68 {
		t.Errorf("offsets[1] = %d, want 68", offsets[1])
	}

	chunkOffsets := st.CalculateChunkOffsets(offsets)
	if len(chunkOffsets) != 1 || chunkOffsets[0] != 36 {
		t.Errorf("CalculateChunkOffsets = %v, want [36]", chunkOffsets)
	}
}

func TestInspectContainer(t *testing.T) {
	data := createSyntheticMP4()
	meta, err := InspectContainer(data)
	if err != nil {
		t.Fatalf("InspectContainer error = %v", err)
	}

	if meta.SampleCount != 2 {
		t.Errorf("SampleCount = %d, want 2", meta.SampleCount)
	}
	if meta.ChunkCount != 1 {
		t.Errorf("ChunkCount = %d, want 1", meta.ChunkCount)
	}
	if meta.ChannelCount != 8 {
		t.Errorf("ChannelCount = %d, want 8", meta.ChannelCount)
	}
}

func TestParseSampleTableErrorPaths(t *testing.T) {
	tests := []struct {
		name string
		data []byte
	}{
		{"empty data", []byte{}},
		{"no moov", []byte{0, 0, 0, 8, 'f', 't', 'y', 'p'}},
		{"no mdat in inspect", []byte{0, 0, 0, 8, 'm', 'o', 'o', 'v'}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := ParseSampleTable(tt.data)
			if err == nil {
				t.Errorf("ParseSampleTable(%s) expected error, got nil", tt.name)
			}

			_, err = InspectContainer(tt.data)
			if err == nil {
				t.Errorf("InspectContainer(%s) expected error, got nil", tt.name)
			}
		})
	}
}

func TestFindBoxExtendedSize(t *testing.T) {
	buf := make([]byte, 32)
	binary.BigEndian.PutUint32(buf[0:4], 1) // extended size flag
	copy(buf[4:8], "test")
	binary.BigEndian.PutUint64(buf[8:16], 32) // size 32

	box := FindBox(buf, "test", 0)
	if box == nil {
		t.Fatalf("FindBox with extended size returned nil")
	}
	if box.Size != 32 {
		t.Errorf("box.Size = %d, want 32", box.Size)
	}
}

func TestParseSampleTableNestedErrors(t *testing.T) {
	// moov container with empty trak (size 8)
	trakHeader := make([]byte, 8)
	binary.BigEndian.PutUint32(trakHeader[0:4], 8)
	copy(trakHeader[4:8], "trak")

	moov := make([]byte, 16)
	binary.BigEndian.PutUint32(moov[0:4], 16)
	copy(moov[4:8], "moov")
	copy(moov[8:], trakHeader)

	_, err := ParseSampleTable(moov)
	if err == nil || !strings.Contains(err.Error(), "no mdia box found") {
		t.Errorf("expected no mdia box error, got %v", err)
	}

	// mdia inside trak (trak size 16)
	mdiaHeader := make([]byte, 8)
	binary.BigEndian.PutUint32(mdiaHeader[0:4], 8)
	copy(mdiaHeader[4:8], "mdia")

	trak := make([]byte, 16)
	binary.BigEndian.PutUint32(trak[0:4], 16)
	copy(trak[4:8], "trak")
	copy(trak[8:], mdiaHeader)

	moov = make([]byte, 24)
	binary.BigEndian.PutUint32(moov[0:4], 24)
	copy(moov[4:8], "moov")
	copy(moov[8:], trak)

	_, err = ParseSampleTable(moov)
	if err == nil || !strings.Contains(err.Error(), "no minf box found") {
		t.Errorf("expected no minf box error, got %v", err)
	}

	// minf inside mdia (mdia size 16) inside trak (trak size 24)
	minfHeader := make([]byte, 8)
	binary.BigEndian.PutUint32(minfHeader[0:4], 8)
	copy(minfHeader[4:8], "minf")

	mdia := make([]byte, 16)
	binary.BigEndian.PutUint32(mdia[0:4], 16)
	copy(mdia[4:8], "mdia")
	copy(mdia[8:], minfHeader)

	trak = make([]byte, 24)
	binary.BigEndian.PutUint32(trak[0:4], 24)
	copy(trak[4:8], "trak")
	copy(trak[8:], mdia)

	moov = make([]byte, 32)
	binary.BigEndian.PutUint32(moov[0:4], 32)
	copy(moov[4:8], "moov")
	copy(moov[8:], trak)

	_, err = ParseSampleTable(moov)
	if err == nil || !strings.Contains(err.Error(), "no stbl box found") {
		t.Errorf("expected no stbl box error, got %v", err)
	}
}

func TestInspectEncryptedContainer(t *testing.T) {
	data := createSyntheticMP4()
	// Set payload byte of sample 1 to encrypted signature (0xe4)
	data[68] = 0xe4

	meta, err := InspectContainer(data)
	if err != nil {
		t.Fatalf("InspectContainer error = %v", err)
	}

	if !meta.IsEncrypted {
		t.Errorf("expected IsEncrypted = true, got false")
	}
}

func TestParseSampleTableStszFixedSize(t *testing.T) {
	data := createSyntheticMP4()
	// Locate stsz box and change sampleSize (offset + 12) to 32
	stszPos := strings.Index(string(data), "stsz")
	if stszPos > 4 {
		binary.BigEndian.PutUint32(data[stszPos+8:stszPos+12], 32)
	}

	st, err := ParseSampleTable(data)
	if err != nil {
		t.Fatalf("ParseSampleTable with fixed sample size failed: %v", err)
	}

	if st.SampleCount != 2 || st.SampleSizes[0] != 32 {
		t.Errorf("SampleSizes[0] = %d, want 32", st.SampleSizes[0])
	}
}

func TestParseSampleTableTruncatedBoxes(t *testing.T) {
	// stsz truncated offset
	stbl := make([]byte, 16)
	binary.BigEndian.PutUint32(stbl[0:4], 16)
	copy(stbl[4:8], "stbl")
	binary.BigEndian.PutUint32(stbl[8:12], 8)
	copy(stbl[12:16], "stsz") // stsz size 8, but parser expects >= 20

	minf := make([]byte, 24)
	binary.BigEndian.PutUint32(minf[0:4], 24)
	copy(minf[4:8], "minf")
	copy(minf[8:], stbl)

	mdia := make([]byte, 32)
	binary.BigEndian.PutUint32(mdia[0:4], 32)
	copy(mdia[4:8], "mdia")
	copy(mdia[8:], minf)

	trak := make([]byte, 40)
	binary.BigEndian.PutUint32(trak[0:4], 40)
	copy(trak[4:8], "trak")
	copy(trak[8:], mdia)

	moov := make([]byte, 48)
	binary.BigEndian.PutUint32(moov[0:4], 48)
	copy(moov[4:8], "moov")
	copy(moov[8:], trak)

	_, err := ParseSampleTable(moov)
	if err == nil || !strings.Contains(err.Error(), "invalid stsz box offset/size") {
		t.Errorf("expected invalid stsz error, got %v", err)
	}
}

func TestParseSampleTableTruncatedCoAndSc(t *testing.T) {
	// stsz valid (20 bytes header + 8 bytes entries = 28)
	stsz := make([]byte, 28)
	binary.BigEndian.PutUint32(stsz[0:4], 28)
	copy(stsz[4:8], "stsz")
	binary.BigEndian.PutUint32(stsz[12:16], 0)
	binary.BigEndian.PutUint32(stsz[16:20], 2)

	// stco truncated offset (< 16)
	stco := make([]byte, 8)
	binary.BigEndian.PutUint32(stco[0:4], 8)
	copy(stco[4:8], "stco")

	stblContent := append(stsz, stco...)
	stbl := make([]byte, 8+len(stblContent))
	binary.BigEndian.PutUint32(stbl[0:4], uint32(len(stbl)))
	copy(stbl[4:8], "stbl")
	copy(stbl[8:], stblContent)

	minf := make([]byte, 8+len(stbl))
	binary.BigEndian.PutUint32(minf[0:4], uint32(len(minf)))
	copy(minf[4:8], "minf")
	copy(minf[8:], stbl)

	mdia := make([]byte, 8+len(minf))
	binary.BigEndian.PutUint32(mdia[0:4], uint32(len(mdia)))
	copy(mdia[4:8], "mdia")
	copy(mdia[8:], minf)

	trak := make([]byte, 8+len(mdia))
	binary.BigEndian.PutUint32(trak[0:4], uint32(len(trak)))
	copy(trak[4:8], "trak")
	copy(trak[8:], mdia)

	moov := make([]byte, 8+len(trak))
	binary.BigEndian.PutUint32(moov[0:4], uint32(len(moov)))
	copy(moov[4:8], "moov")
	copy(moov[8:], trak)

	_, err := ParseSampleTable(moov)
	if err == nil || !strings.Contains(err.Error(), "invalid stco box offset/size") {
		t.Errorf("expected invalid stco error, got %v", err)
	}
}
