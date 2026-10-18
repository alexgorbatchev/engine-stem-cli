// Package stems provides high-level encryption and decryption for Engine DJ .stems files.
package stems

import (
	"encoding/binary"
	"fmt"
	"os"

	"github.com/alexgorbatchev/engine-stem-cli/internal/crypto"
	"github.com/alexgorbatchev/engine-stem-cli/internal/mp4"
)

// Decrypt decrypts an Engine DJ .stems file to a standard M4A file.
func Decrypt(inputPath, outputPath string, verbose bool) error {
	data, err := os.ReadFile(inputPath)
	if err != nil {
		return fmt.Errorf("reading input %q: %w", inputPath, err)
	}

	if verbose {
		fmt.Printf("Reading input: %s (%d bytes)\n", inputPath, len(data))
	}

	mdat := mp4.FindBox(data, "mdat", 0)
	if mdat == nil {
		return fmt.Errorf("invalid file format: no mdat box found")
	}
	mdatDataStart := mdat.Offset + 8

	st, err := mp4.ParseSampleTable(data)
	if err != nil {
		return fmt.Errorf("parsing sample table: %w", err)
	}

	if verbose {
		fmt.Printf("Sample table: %d samples across %d chunks\n", st.SampleCount, st.ChunkCount)
	}

	sampleOffsets := st.CalculateSampleOffsets()
	cipher, err := crypto.NewCipher()
	if err != nil {
		return fmt.Errorf("initializing cipher: %w", err)
	}

	decryptedSamples := make([][]byte, st.SampleCount)
	newSizes := make([]int, st.SampleCount)
	totalPaddingRemoved := 0

	for i := 0; i < st.SampleCount; i++ {
		offset := sampleOffsets[i]
		size := st.SampleSizes[i]

		if offset+size > len(data) {
			return fmt.Errorf("sample %d offset out of bounds (%d + %d > %d)", i, offset, size, len(data))
		}

		sample := data[offset : offset+size]
		decrypted := cipher.DecryptSample(sample)
		decryptedSamples[i] = decrypted
		newSizes[i] = len(decrypted)
		totalPaddingRemoved += size - len(decrypted)
	}

	if verbose {
		fmt.Printf("Stripped %d bytes of PKCS#7 padding\n", totalPaddingRemoved)
	}

	newMdatContent := make([]byte, 0, len(data))
	for _, sample := range decryptedSamples {
		newMdatContent = append(newMdatContent, sample...)
	}
	newMdatSize := 8 + len(newMdatContent)

	newSampleOffsets := make([]int, st.SampleCount)
	offset := mdatDataStart
	for i := 0; i < st.SampleCount; i++ {
		newSampleOffsets[i] = offset
		offset += newSizes[i]
	}

	st.SampleSizes = newSizes
	newChunkOffsets := st.CalculateChunkOffsets(newSampleOffsets)

	output := buildOutput(data, mdat, newMdatSize, newMdatContent, st, newSizes, newChunkOffsets)

	if err := os.WriteFile(outputPath, output, 0644); err != nil {
		return fmt.Errorf("writing decrypted file %q: %w", outputPath, err)
	}

	if verbose {
		fmt.Printf("Wrote decrypted output: %s (%d bytes)\n", outputPath, len(output))
	}

	return nil
}

// Encrypt encrypts a standard M4A file to Engine DJ .stems format.
func Encrypt(inputPath, outputPath string, verbose bool) error {
	data, err := os.ReadFile(inputPath)
	if err != nil {
		return fmt.Errorf("reading input %q: %w", inputPath, err)
	}

	if verbose {
		fmt.Printf("Reading input: %s (%d bytes)\n", inputPath, len(data))
	}

	mdat := mp4.FindBox(data, "mdat", 0)
	if mdat == nil {
		return fmt.Errorf("invalid file format: no mdat box found")
	}
	mdatDataStart := mdat.Offset + 8

	st, err := mp4.ParseSampleTable(data)
	if err != nil {
		return fmt.Errorf("parsing sample table: %w", err)
	}

	if verbose {
		fmt.Printf("Sample table: %d samples across %d chunks\n", st.SampleCount, st.ChunkCount)
	}

	sampleOffsets := st.CalculateSampleOffsets()
	samples := make([][]byte, st.SampleCount)
	for i := 0; i < st.SampleCount; i++ {
		offset := sampleOffsets[i]
		size := st.SampleSizes[i]
		if offset+size > len(data) {
			return fmt.Errorf("sample %d offset out of bounds (%d + %d > %d)", i, offset, size, len(data))
		}
		samples[i] = make([]byte, size)
		copy(samples[i], data[offset:offset+size])
	}

	cipher, err := crypto.NewCipher()
	if err != nil {
		return fmt.Errorf("initializing cipher: %w", err)
	}

	encryptedSamples := make([][]byte, st.SampleCount)
	newSizes := make([]int, st.SampleCount)
	totalPadding := 0

	for i := 0; i < st.SampleCount; i++ {
		encrypted := cipher.EncryptSample(samples[i])
		encryptedSamples[i] = encrypted
		newSizes[i] = len(encrypted)
		totalPadding += len(encrypted) - len(samples[i])
	}

	if verbose {
		fmt.Printf("Added %d bytes of PKCS#7 padding\n", totalPadding)
	}

	newMdatContent := make([]byte, 0, len(data))
	for _, sample := range encryptedSamples {
		newMdatContent = append(newMdatContent, sample...)
	}
	newMdatSize := 8 + len(newMdatContent)

	newSampleOffsets := make([]int, st.SampleCount)
	offset := mdatDataStart
	for i := 0; i < st.SampleCount; i++ {
		newSampleOffsets[i] = offset
		offset += newSizes[i]
	}

	st.SampleSizes = newSizes
	newChunkOffsets := st.CalculateChunkOffsets(newSampleOffsets)

	output := buildOutput(data, mdat, newMdatSize, newMdatContent, st, newSizes, newChunkOffsets)

	if err := os.WriteFile(outputPath, output, 0644); err != nil {
		return fmt.Errorf("writing encrypted file %q: %w", outputPath, err)
	}

	if verbose {
		fmt.Printf("Wrote encrypted output: %s (%d bytes)\n", outputPath, len(output))
	}

	return nil
}

func buildOutput(data []byte, mdat *mp4.Box, newMdatSize int, newMdatContent []byte,
	st *mp4.SampleTable, newSizes []int, newChunkOffsets []int) []byte {

	output := make([]byte, 0, len(data))

	output = append(output, data[:mdat.Offset]...)

	mdatHeader := make([]byte, 8)
	binary.BigEndian.PutUint32(mdatHeader[0:4], uint32(newMdatSize))
	copy(mdatHeader[4:8], []byte("mdat"))
	output = append(output, mdatHeader...)

	output = append(output, newMdatContent...)

	moovStart := mdat.Offset + mdat.Size
	moovData := make([]byte, len(data)-moovStart)
	copy(moovData, data[moovStart:])

	stcoRelOffset := st.StcoOffset - moovStart
	stszRelOffset := st.StszOffset - moovStart

	for i := 0; i < st.ChunkCount; i++ {
		pos := stcoRelOffset + 16 + i*4
		if pos+4 <= len(moovData) {
			binary.BigEndian.PutUint32(moovData[pos:pos+4], uint32(newChunkOffsets[i]))
		}
	}

	for i := 0; i < st.SampleCount; i++ {
		pos := stszRelOffset + 20 + i*4
		if pos+4 <= len(moovData) {
			binary.BigEndian.PutUint32(moovData[pos:pos+4], uint32(newSizes[i]))
		}
	}

	output = append(output, moovData...)

	return output
}
