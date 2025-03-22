// Copyright 2024 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package s390xasm

import (
	"encoding/binary"
	"strings"
	"testing"
)

func TestObjdumpS390xTestDecodeGNUSyntaxdata(t *testing.T) {
	testObjdumpS390x(t, testdataCases(t))
}

func TestObjdumpS390xTestDecodeGoSyntaxdata(t *testing.T) {
	testObjdumpS390x(t, testdataCases(t))
}

// objdumpManualTests holds test cases that will be run by TestObjdumpPowerManual.
// If you are debugging a few cases that turned up in a longer run, it can be useful
// to list them here and then use -run=Manual, particularly with tracing enabled.
// Note that these are byte sequences, so they must be reversed from the usual
// word presentation.
var objdumpManualTests = `
6d746162
4c040000
88000017
`

// allowedMismatchObjdump reports whether the mismatch between text and dec
// should be allowed by the test.
func allowedMismatchObjdump(text string, size int, inst *Inst, dec ExtInst) bool {
        // we support more instructions than binutils
        if strings.Contains(dec.text, ".long") {
                return true
        }

        switch inst.Op {
        default:
                return true
        }

        if len(dec.enc) >= 4 {
                _ = binary.BigEndian.Uint32(dec.enc[:4])
        }

        return false
}
