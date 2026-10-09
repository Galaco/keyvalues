package keyvalues

import (
	"encoding/binary"
	"errors"
	"strings"
	"testing"
)

// FuzzRead checks that any input either parses into a well formed tree that
// survives being written as UTF-8 and UTF-16 and read back unchanged, or
// fails with a *SyntaxError.
func FuzzRead(f *testing.F) {
	for _, tc := range readCases {
		f.Add(tc.input)
		f.Add(string(encodeUTF16(tc.input, binary.LittleEndian)))
		f.Add(string(encodeUTF16(tc.input, binary.BigEndian)))
	}
	for _, tc := range readErrorCases {
		f.Add(tc.input)
		f.Add(string(encodeUTF16(tc.input, binary.LittleEndian)))
	}
	for _, data := range fixtures(f) {
		f.Add(data)
	}

	f.Fuzz(func(t *testing.T, input string) {
		for _, escapes := range []bool{false, true} {
			r := NewReader(strings.NewReader(input))
			if escapes {
				r.UseEscapeSequences()
			}
			doc, err := r.Read()
			if err != nil {
				var syntaxErr *SyntaxError
				if !errors.As(err, &syntaxErr) {
					t.Fatalf("escapes %v: got %T %v, want a *SyntaxError", escapes, err, err)
				}
				continue
			}
			checkParents(t, doc)
			if err := roundTrip(input, escapes); err != nil {
				t.Fatalf("escapes %v: %v", escapes, err)
			}
		}
	})
}
