// Copyright 2024 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package s390xasm

import (
	"bytes"
	"debug/elf"
	"encoding/binary"
	"runtime"
	"fmt"
	"io"
	"log"
	"os"
	"os/exec"
	"strconv"
	"testing"
)

var objdumpPath = "objdump"

var objdumpCrossNames = [...]string{"s390x-linux-gnu-objdump", "s390x-linux-plan9-objdump"}

func testObjdumpS390x(t *testing.T, generate func(func([]byte))) {
	testObjdumpArch(t, generate)
}

func testObjdumpArch(t *testing.T, generate func(func([]byte))) {
	checkObjdumpS390x(t)
	testExtDis(t, "plan9", objdump, generate, allowedMismatchObjdump)
}

func checkObjdumpS390x(t *testing.T) {
        if testing.Short() {
                t.Skip("skipping objdump test in short mode")
        }
        if runtime.GOARCH != "s390x" {
                found := false
                for _, c := range objdumpCrossNames {
                        if _, err := exec.LookPath(c); err == nil {
                                objdumpPath = c
                                found = true
                                break
                        }
                }
                if !found {
                        t.Skip("skipping; test requires host tool objdump for ppc64 or ppc64le")
                }
        } else if _, err := exec.LookPath(objdumpPath); err != nil {
                t.Skip(err)
        }
}

func objdump(ext *ExtDis) error {
	// File already written with instructions; add ELF header.
	if err := writeELF64(ext.File, ext.Size); err != nil {
		return err
	}

	b, err := ext.Run(objdumpPath, "-d", "-z", ext.File.Name())
	if err != nil {
		return err
	}

	var (
		nmatch  int
		reading bool
		next    uint32 = start
		addr    uint32
		encbuf  [8]byte
		enc     []byte
		text    string
	)
	flush := func() {
		if addr == next {
			// PC-relative addresses are translated to absolute addresses based on PC by GNU objdump
			// Following logical rewrites the absolute addresses back to PC-relative ones for comparing
			// with our disassembler output which are PC-relative
			if text == "undefined" && len(enc) == 4 {
				text = "error: unknown instruction"
				enc = nil
			}
			if len(enc) == 4 {
				// prints as word but we want to record bytes
				enc[0], enc[3] = enc[3], enc[0]
				enc[1], enc[2] = enc[2], enc[1]
			}
			ext.Dec <- ExtInst{addr, encbuf, len(enc), text}
			encbuf = [8]byte{}
			next += uint32(len(enc))
			enc = nil
		}
	}
	var textangle = []byte("<.text>:")
	for {
		line, err := b.ReadSlice('\n')
		if err != nil {
			if err == io.EOF {
				break
			}
			return fmt.Errorf("reading objdump output: %v", err)
		}
		if bytes.Contains(line, textangle) {
			reading = true
			continue
		}
		if !reading {
			continue
		}
		if debug {
			os.Stdout.Write(line)
		}
		if enc1 := parseContinuation(line, encbuf[:len(enc)]); enc1 != nil {
			enc = enc1
			continue
		}
		flush()
		nmatch++
		addr, enc, text = parseLine(line, encbuf[:0])
		if addr > next {
			return fmt.Errorf("address out of sync expected <= %#x at %q in:\n%s", next, line, line)
		}
	}
	flush()
	if next != start+uint32(ext.Size) {
		return fmt.Errorf("not enough results found [%d %d]", next, start+ext.Size)
	}
	if err := ext.Wait(); err != nil {
		return fmt.Errorf("exec: %v", err)
	}

	return nil
}

//*****  CHECK THIS VAR VALUES ******* //
var (
	undefined     = []byte("undefined")
	unpredictable = []byte("unpredictable")
	slashslash    = []byte("//")
)

func parseLine(line []byte, encstart []byte) (addr uint32, enc []byte, text string) {
	ok := false
	oline := line
	i := index(line, ":\t")
	if i < 0 {
		log.Fatalf("cannot parse disassembly: %q", oline)
	}
	x, err := strconv.ParseUint(string(bytes.TrimSpace(line[:i])), 16, 32)
	if err != nil {
		log.Fatalf("cannot parse disassembly: %q", oline)
	}
	addr = uint32(x)
	line = line[i+2:]
	i = bytes.IndexByte(line, '\t')
	if i < 0 {
		log.Fatalf("cannot parse disassembly: %q", oline)
	}
	enc, ok = parseHex(line[:i], encstart)
	if !ok {
		log.Fatalf("cannot parse disassembly: %q", oline)
	}
	line = bytes.TrimSpace(line[i:])
	if bytes.Contains(line, undefined) {
		text = "undefined"
		return
	}
	if false && bytes.Contains(line, unpredictable) {
		text = "unpredictable"
		return
	}
	// Strip trailing comment starting with '#'
	if i := bytes.IndexByte(line, '#'); i >= 0 {
		line = bytes.TrimSpace(line[:i])
	}
	// Strip trailing comment starting with "//"
	if i := bytes.Index(line, slashslash); i >= 0 {
		line = bytes.TrimSpace(line[:i])
	}
	text = string(fixSpace(line))
	return
}

func parseContinuation(line []byte, enc []byte) []byte {
	i := index(line, ":\t")
	if i < 0 {
		return nil
	}
	line = line[i+1:]
	enc, _ = parseHex(line, enc)
	return enc
}

// writeELF64 writes an ELF64 header to the file, describing a text
// segment that starts at start and extends for size bytes.
func writeELF64(f *os.File, size int) error {
	f.Seek(0, io.SeekStart)
	var hdr elf.Header64
	var prog elf.Prog64
	var sect elf.Section64
	var buf bytes.Buffer
	binary.Write(&buf, binary.BigEndian, &hdr)
	off1 := buf.Len()
	binary.Write(&buf, binary.BigEndian, &prog)
	off2 := buf.Len()
	binary.Write(&buf, binary.BigEndian, &sect)
	off3 := buf.Len()
	buf.Reset()
	data := byte(elf.ELFDATA2MSB)
	hdr = elf.Header64{
		Ident:     [16]byte{0x7F, 'E', 'L', 'F', 2, data, 1},
		Type:      2,
		Machine:   uint16(elf.EM_S390),
		Version:   1,
		Entry:     start,
		Phoff:     uint64(off1),
		Shoff:     uint64(off2),
		Flags:     0x3,           //*****  CHECK THIS FLAG VALUE ******* //
		Ehsize:    uint16(off1),
		Phentsize: uint16(off2 - off1),
		Phnum:     1,
		Shentsize: uint16(off3 - off2),
		Shnum:     3,
		Shstrndx:  2,
	}
	binary.Write(&buf, binary.BigEndian, &hdr)
	prog = elf.Prog64{
		Type:   1,
		Off:    start,
		Vaddr:  start,
		Paddr:  start,
		Filesz: uint64(size),
		Memsz:  uint64(size),
		Flags:  5,
		Align:  start,
	}
	binary.Write(&buf, binary.BigEndian, &prog)
	binary.Write(&buf, binary.BigEndian, &sect) // NULL section
	sect = elf.Section64{
		Name:      1,
		Type:      uint32(elf.SHT_PROGBITS),
		Addr:      start,
		Off:       start,
		Size:      uint64(size),
		Flags:     uint64(elf.SHF_ALLOC | elf.SHF_EXECINSTR),
		Addralign: 4,
	}
	binary.Write(&buf, binary.BigEndian, &sect) // .text
	sect = elf.Section64{
		Name:      uint32(len("\x00.text\x00")),
		Type:      uint32(elf.SHT_STRTAB),
		Addr:      0,
		Off:       uint64(off2 + (off3-off2)*3),
		Size:      uint64(len("\x00.text\x00.shstrtab\x00")),
		Addralign: 1,
	}
	binary.Write(&buf, binary.BigEndian, &sect)
	buf.WriteString("\x00.text\x00.shstrtab\x00")
	f.Write(buf.Bytes())
	return nil
}
