package testfixture

import (
	"bytes"
	"encoding/binary"
	"os"
	"path/filepath"
	"testing"
)

// EngineBinary contains synthetic ARM64 instructions and an AES import stub.
// The values vary per call; no installed application bytes are copied here.
func EngineBinary(t testing.TB, key []byte) string {
	t.Helper()
	if len(key) != 16 {
		t.Fatal("synthetic AES key must have 16 bytes")
	}
	words := []uint32{0x90000008, 0x91000108, 0x94000000} // ADRP, ADD, BL (patched below).
	words[2] = 0x94000000 | uint32((0x800-8)/4)
	words = append(words, 0x528002a8, 0xb90003e8) // MOV W8,#21; STR W8,[SP].
	for i, b := range key {
		words = append(words,
			0xb94003e8,                 // LDR W8,[SP]
			0x11000108|uint32(i)<<10,   // ADD W8,W8,#i
			0x52800009|uint32(b)<<5,    // MOV W9,#synthetic-byte
			0x4a090108,                 // EOR W8,W8,W9
			0x390003e8|uint32(i+4)<<10, // STRB W8,[SP,#i+4]
		)
	}
	for i := range key {
		words = append(words,
			0x394003e8|uint32(i+4)<<10, // LDRB W8,[SP,#i+4]
			0xb94003e9,                 // LDR W9,[SP]
			0x11000129|uint32(i)<<10,   // ADD W9,W9,#i
			0x4a080128,                 // EOR W8,W9,W8
			0x390003e8|uint32(i+4)<<10, // STRB W8,[SP,#i+4]
		)
	}
	words = append(words, 0x390003ff|uint32(20)<<10, 0x94000000, 0x90000008, 0x91000108, 0xd65f03c0)
	text := new(bytes.Buffer)
	for _, w := range words {
		if err := binary.Write(text, binary.LittleEndian, w); err != nil {
			t.Fatal(err)
		}
	}
	// A minimal 64-bit Mach-O: __text and __stubs, symbol + indirect tables.
	data := make([]byte, 0x900)
	put := func(offset int, values ...uint32) {
		for i, v := range values {
			binary.LittleEndian.PutUint32(data[offset+4*i:], v)
		}
	}
	put(0, 0xfeedfacf, 0x100000c, 0, 2, 3, 72+160+24+80, 0, 0)
	put(32, 0x19, 232)
	copy(data[40:], "__TEXT")
	binary.LittleEndian.PutUint64(data[56:], 0x100000000)
	binary.LittleEndian.PutUint64(data[64:], 0x1000)
	binary.LittleEndian.PutUint64(data[80:], uint64(len(data)))
	put(88, 7, 5, 2, 0)
	section := func(offset int, name string, addr uint64, size uint64, fileoff, flags, stride uint32) {
		copy(data[offset:], name)
		copy(data[offset+16:], "__TEXT")
		binary.LittleEndian.PutUint64(data[offset+32:], addr)
		binary.LittleEndian.PutUint64(data[offset+40:], size)
		put(offset+48, fileoff, 2, 0, 0, flags, 0, stride, 0)
	}
	section(104, "__text", 0x100000400, uint64(text.Len()), 0x400, 0x80000400, 0)
	section(184, "__stubs", 0x100000c00, 12, 0x800, 8, 12)
	put(264, 2, 24, 0x820, 1, 0x840, 32)
	put(288, 0xb, 80)
	put(288+56, 0x830, 1)
	put(0x820, 1)
	data[0x824] = 1 // external undefined symbol
	copy(data[0x841:], "_av_aes_init\x00")
	copy(data[0x400:], text.Bytes())
	path := filepath.Join(t.TempDir(), "synthetic-analyzer")
	if err := os.WriteFile(path, data, 0600); err != nil {
		t.Fatal(err)
	}
	return path
}
