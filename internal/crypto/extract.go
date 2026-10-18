package crypto

import (
	"bytes"
	"crypto/aes"
	"debug/macho"
	"encoding/binary"
	"fmt"
	"strconv"
	"strings"

	"golang.org/x/arch/arm64/arm64asm"
)

// extractKey reads code, never executes the installed application. Only the
// verified ARM64 initializer layout is supported; unknown layouts fail closed.
func extractKey(data []byte) ([]byte, error) {
	r := bytes.NewReader(data)
	if fat, err := macho.NewFatFile(r); err == nil {
		for _, arch := range fat.Arches {
			if arch.Cpu == macho.CpuArm64 {
				return extractMachOKey(arch.File)
			}
		}
		return nil, fmt.Errorf("Engine DJ binary has no ARM64 slice")
	}
	f, err := macho.NewFile(r)
	if err != nil {
		return nil, fmt.Errorf("parsing Engine DJ Mach-O binary: %w", err)
	}
	return extractMachOKey(f)
}

func aesInitStubs(f *macho.File) (map[uint64]bool, error) {
	if f.Symtab == nil || f.Dysymtab == nil {
		return nil, fmt.Errorf("Engine DJ binary is missing import tables")
	}
	stubs := make(map[uint64]bool)
	for _, load := range f.Loads {
		seg, ok := load.(*macho.Segment)
		if !ok || seg.Cmd != macho.LoadCmdSegment64 {
			continue
		}
		r := bytes.NewReader(seg.Raw())
		var header macho.Segment64
		if err := binary.Read(r, f.ByteOrder, &header); err != nil {
			return nil, fmt.Errorf("reading segment: %w", err)
		}
		for range header.Nsect {
			var s macho.Section64
			if err := binary.Read(r, f.ByteOrder, &s); err != nil {
				return nil, fmt.Errorf("reading section: %w", err)
			}
			if s.Flags&0xff != 8 || s.Reserve2 == 0 { // S_SYMBOL_STUBS
				continue
			}
			for i := uint64(0); i < s.Size/uint64(s.Reserve2); i++ {
				idx := uint64(s.Reserve1) + i
				if idx >= uint64(len(f.Dysymtab.IndirectSyms)) {
					return nil, fmt.Errorf("invalid indirect symbol index")
				}
				sym := f.Dysymtab.IndirectSyms[idx]
				if uint64(sym) < uint64(len(f.Symtab.Syms)) && f.Symtab.Syms[sym].Name == "_av_aes_init" {
					stubs[s.Addr+i*uint64(s.Reserve2)] = true
				}
			}
		}
	}
	if len(stubs) == 0 {
		return nil, fmt.Errorf("Engine DJ binary has no av_aes_init import")
	}
	return stubs, nil
}

func extractMachOKey(f *macho.File) ([]byte, error) {
	if f.Cpu != macho.CpuArm64 {
		return nil, fmt.Errorf("unsupported Engine DJ architecture; select a binary with an ARM64 slice")
	}
	stubs, err := aesInitStubs(f)
	if err != nil {
		return nil, err
	}
	section := f.Section("__text")
	if section == nil {
		return nil, fmt.Errorf("Engine DJ binary is missing __text")
	}
	text, err := section.Data()
	if err != nil {
		return nil, fmt.Errorf("reading Engine DJ instructions: %w", err)
	}
	// Locate the global key object passed to av_aes_init, then require the
	// initializer to reference that same object after building its stack bytes.
	addresses := make(map[uint64]bool)
	for off := 0; off+4 <= len(text); off += 4 {
		inst, err := arm64asm.Decode(text[off : off+4])
		if err != nil || inst.Op != arm64asm.BL {
			continue
		}
		rel, ok := inst.Args[0].(arm64asm.PCRel)
		if !ok || !stubs[uint64(int64(section.Addr)+int64(off)+int64(rel))] {
			continue
		}
		start := max(0, off-48)
		for pos := start; pos+8 <= off; pos += 4 {
			if addr, ok := globalAddress(text[pos:pos+8], section.Addr+uint64(pos)); ok {
				addresses[addr] = true
			}
		}
	}
	var key []byte
	for off := 0; off+8 <= len(text); off += 4 {
		inst, err := arm64asm.Decode(text[off : off+4])
		if err != nil || inst.Op != arm64asm.MOV {
			continue
		}
		if _, ok := immediate(inst.Args[1]); !ok {
			continue
		}
		candidate, end := initializerKey(text[off:min(off+1024, len(text))])
		if candidate == nil {
			continue
		}
		for pos := off + end; pos+8 <= min(off+end+256, len(text)); pos += 4 {
			addr, ok := globalAddress(text[pos:pos+8], section.Addr+uint64(pos))
			if ok && addresses[addr] {
				if key != nil && !bytes.Equal(key, candidate) {
					return nil, fmt.Errorf("ambiguous Engine DJ key initializers")
				}
				key = candidate
				break
			}
		}
	}
	if key == nil {
		return nil, fmt.Errorf("supported Engine DJ AES key initializer not found")
	}
	return key, nil
}

func immediate(arg arm64asm.Arg) (uint64, bool) {
	switch v := arg.(type) {
	case arm64asm.Imm:
		return uint64(v.Imm), true
	case arm64asm.Imm64:
		return v.Imm, true
	case arm64asm.ImmShift:
		// Shifted immediates are deliberately outside the supported layout.
		n, err := strconv.ParseUint(strings.TrimPrefix(v.String(), "#"), 0, 64)
		return n, err == nil
	}
	return 0, false
}

func globalAddress(code []byte, pc uint64) (uint64, bool) {
	a, err := arm64asm.Decode(code[:4])
	if err != nil || a.Op != arm64asm.ADRP {
		return 0, false
	}
	b, err := arm64asm.Decode(code[4:8])
	if err != nil || b.Op != arm64asm.ADD || b.Args[0].String() != a.Args[0].String() || b.Args[1].String() != a.Args[0].String() {
		return 0, false
	}
	rel, ok := a.Args[1].(arm64asm.PCRel)
	n, valid := immediate(b.Args[2])
	return uint64(int64(pc&^0xfff)+int64(rel)) + n, ok && valid
}

func stackOffset(arg arm64asm.Arg) (int, bool) {
	m, ok := arg.(arm64asm.MemImmediate)
	if !ok || m.Base != arm64asm.RegSP(arm64asm.SP) || m.Mode != arm64asm.AddrOffset {
		return 0, false
	}
	// x/arch exposes the address displacement through its String method.
	s := m.String()
	if s == "[SP]" {
		return 0, true
	}
	n, err := strconv.Atoi(strings.TrimSuffix(strings.TrimPrefix(s, "[SP,#"), "]"))
	return n, err == nil && n >= 0 && n < 256
}

func initializerKey(code []byte) ([]byte, int) {
	regs := make(map[string]uint64)
	stack := make(map[int]uint64)
	writes := make(map[int]int)
	value := func(arg arm64asm.Arg) (uint64, bool) {
		if n, ok := immediate(arg); ok {
			return n, true
		}
		if arg == arm64asm.WZR || arg == arm64asm.XZR {
			return 0, true
		}
		if arg == nil {
			return 0, false
		}
		n, ok := regs[arg.String()]
		return n, ok
	}
	for off := 0; off+4 <= len(code); off += 4 {
		in, err := arm64asm.Decode(code[off : off+4])
		if err != nil {
			return nil, off
		}
		if in.Op == arm64asm.BL {
			for start := range writes {
				key := make([]byte, aes.BlockSize)
				valid := true
				for i := range key {
					if writes[start+i] < 2 {
						valid = false
					}
					key[i] = byte(stack[start+i])
				}
				if valid && writes[start+aes.BlockSize] > 0 && stack[start+aes.BlockSize] == 0 {
					return key, off
				}
			}
			return nil, off
		}
		a, b := in.Args[0], in.Args[1]
		if a == nil {
			return nil, off
		}
		switch in.Op {
		case arm64asm.MOV:
			delete(regs, a.String())
			if n, ok := value(b); ok {
				regs[a.String()] = n
			}
		case arm64asm.ADD, arm64asm.EOR:
			x, okX := value(b)
			y, okY := value(in.Args[2])
			delete(regs, a.String())
			if okX && okY {
				if in.Op == arm64asm.ADD {
					regs[a.String()] = x + y
				} else {
					regs[a.String()] = x ^ y
				}
			}
		case arm64asm.LDR, arm64asm.LDRB:
			delete(regs, a.String())
			if offset, ok := stackOffset(b); ok {
				if n, exists := stack[offset]; exists {
					regs[a.String()] = n
				}
			}
		case arm64asm.STR, arm64asm.STRB:
			if offset, ok := stackOffset(b); ok {
				n, known := value(a)
				if !known {
					return nil, off
				}
				if in.Op == arm64asm.STRB {
					n &= 0xff
					writes[offset]++
				}
				stack[offset] = n
			}
		case arm64asm.B, arm64asm.RET:
			return nil, off
		case arm64asm.ORR:
			delete(regs, a.String())
		default:
			return nil, off
		}
	}
	return nil, len(code)
}
