package keyvalues

import (
	"bufio"
	"encoding/binary"
	"errors"
	"io"
	"unicode/utf16"
	"unicode/utf8"
)

// errTruncatedUTF16 is returned by utf16Reader when input ends partway
// through a code unit. Reader.Read reports it as a *SyntaxError.
var errTruncatedUTF16 = errors.New("input ends partway through a UTF-16 character")

// utf16Reader converts UTF-16 read from r into UTF-8, so the scanner only
// ever sees UTF-8. Unpaired surrogates are converted to U+FFFD.
type utf16Reader struct {
	r       *bufio.Reader
	order   binary.ByteOrder
	partial []byte // encoded bytes of a rune that did not fit in the last Read
	err     error
}

func newUTF16Reader(r *bufio.Reader, order binary.ByteOrder) *utf16Reader {
	return &utf16Reader{r: r, order: order}
}

func (u *utf16Reader) Read(p []byte) (int, error) {
	n := copy(p, u.partial)
	u.partial = u.partial[n:]
	for n < len(p) && u.err == nil {
		// Return what has been converted rather than block for more input.
		if n > 0 && u.r.Buffered() < 2 {
			break
		}
		r, err := u.readRune()
		if err != nil {
			u.err = err
			break
		}
		var encoded [utf8.UTFMax]byte
		size := utf8.EncodeRune(encoded[:], r)
		copied := copy(p[n:], encoded[:size])
		n += copied
		if copied < size {
			u.partial = append([]byte(nil), encoded[copied:size]...)
		}
	}
	if n > 0 {
		return n, nil
	}
	return 0, u.err
}

func (u *utf16Reader) readRune() (rune, error) {
	var unit [2]byte
	if _, err := io.ReadFull(u.r, unit[:]); err != nil {
		if err == io.ErrUnexpectedEOF {
			return 0, errTruncatedUTF16
		}
		return 0, err
	}
	r := rune(u.order.Uint16(unit[:]))
	if !utf16.IsSurrogate(r) {
		return r, nil
	}
	if r >= 0xDC00 {
		return utf8.RuneError, nil // a low surrogate with no high surrogate
	}
	next, err := u.r.Peek(2)
	if err != nil && err != io.EOF {
		return 0, err
	}
	if len(next) < 2 {
		return utf8.RuneError, nil
	}
	low := rune(u.order.Uint16(next))
	if low < 0xDC00 || low > 0xDFFF {
		return utf8.RuneError, nil // a high surrogate with no low surrogate
	}
	_, _ = u.r.Discard(2)
	return utf16.DecodeRune(r, low), nil
}

// utf16Writer converts UTF-8 written to it into little-endian UTF-16. Invalid
// UTF-8 is converted to U+FFFD.
type utf16Writer struct {
	w       io.Writer
	partial []byte // the start of a rune split across writes
	buf     []byte
}

func (u *utf16Writer) Write(p []byte) (int, error) {
	n := len(p)
	if len(u.partial) > 0 {
		p = append(u.partial, p...)
		u.partial = nil
	}
	u.buf = u.buf[:0]
	for len(p) > 0 {
		if !utf8.FullRune(p) {
			u.partial = append([]byte(nil), p...)
			break
		}
		r, size := utf8.DecodeRune(p)
		p = p[size:]
		if r >= 0x10000 {
			high, low := utf16.EncodeRune(r)
			u.buf = appendUTF16LE(u.buf, high)
			u.buf = appendUTF16LE(u.buf, low)
		} else {
			u.buf = appendUTF16LE(u.buf, r)
		}
	}
	if _, err := u.w.Write(u.buf); err != nil {
		return 0, err
	}
	return n, nil
}

func appendUTF16LE(b []byte, r rune) []byte {
	return append(b, byte(r), byte(r>>8))
}
