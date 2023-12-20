package bytes

import (
	"bytes"
	"fmt"
	"io"
	"strconv"
	"time"
	"unicode/utf8"
)

type Buffer struct {
	data []byte

	// ifStack is a stack of conditions
	//
	// calling If will push a condition value into it
	//
	// calling ElseIf, Else will modify the last value
	//
	// calling EndIf will pop the last value
	ifStack []bool
	// forStack is a stack to control for loop
	//
	// calling For will push a true before starting for loop, and pop it after finishing
	//
	// calling Break will modify the last value to false, so that the loop could break early
	forStack []bool
}

func NewBuffer(b []byte) *Buffer {
	return &Buffer{
		data:     b,
		ifStack:  make([]bool, 0, 8),
		forStack: make([]bool, 0, 8),
	}
}

// Reader creates an io.Reader for Read.
func (b Buffer) Reader() io.Reader { return bytes.NewReader(b.BytesCopy()) }

// ReadCloser creates an io.ReadCloser, but Close does nothing.
func (b Buffer) ReadCloser() io.ReadCloser { return io.NopCloser(bytes.NewReader(b.BytesCopy())) }

// Write implements io.Writer, and it will write data to buffer directly, without considering any conditions.
// If you need chaining call, use WriteBytes instead.
func (b *Buffer) Write(data []byte) (int, error) {
	b.data = append(b.data, data...)
	return len(data), nil
}

// String implements Stringer.
func (b Buffer) String() string { return string(b.data) }

func (b Buffer) Len() int { return len(b.data) }

func (b Buffer) Cap() int { return cap(b.data) }

func (b *Buffer) Reset() {
	b.data = b.data[:0]
	b.ifStack = b.ifStack[:0]
}

// Bytes returns b.data. It is NOT a copy, so the result will be changed after b modified.
func (b Buffer) Bytes() []byte { return b.data }

func (b Buffer) BytesCopy() []byte {
	bs := make([]byte, b.Len())
	copy(bs, b.data)
	return bs
}

func (b *Buffer) WriteInt(i int) *Buffer {
	return b.WriteInt64(int64(i))
}

func (b *Buffer) WriteInt8(i int8) *Buffer {
	return b.WriteInt64(int64(i))
}

func (b *Buffer) WriteInt16(i int16) *Buffer {
	return b.WriteInt64(int64(i))
}

func (b *Buffer) WriteInt32(i int32) *Buffer {
	return b.WriteInt64(int64(i))
}

func (b *Buffer) WriteInt64(i int64) *Buffer {
	if !b.shouldWrite() {
		return b
	}
	b.data = strconv.AppendInt(b.data, i, 10)
	return b
}

func (b *Buffer) WriteUint(i uint) *Buffer {
	return b.WriteUint64(uint64(i))
}

func (b *Buffer) WriteUint8(i uint8) *Buffer {
	return b.WriteUint64(uint64(i))
}

func (b *Buffer) WriteUint16(i uint16) *Buffer {
	return b.WriteUint64(uint64(i))
}

func (b *Buffer) WriteUint32(i uint32) *Buffer {
	return b.WriteUint64(uint64(i))
}

func (b *Buffer) WriteUint64(i uint64) *Buffer {
	if !b.shouldWrite() {
		return b
	}
	b.data = strconv.AppendUint(b.data, i, 10)
	return b
}

func (b *Buffer) WriteBytes(s []byte) *Buffer {
	if !b.shouldWrite() {
		return b
	}
	b.data = append(b.data, s...)
	return b
}

func (b *Buffer) WriteByte(c byte) *Buffer {
	if !b.shouldWrite() {
		return b
	}
	b.data = append(b.data, c)
	return b
}

func (b *Buffer) WriteRune(r rune) *Buffer {
	if !b.shouldWrite() {
		return b
	}
	var br [4]byte
	n := utf8.EncodeRune(br[:], r)
	b.data = append(b.data, br[:n]...)
	return b
}

func (b *Buffer) WriteString(s string) *Buffer {
	if !b.shouldWrite() {
		return b
	}
	b.data = append(b.data, s...)
	return b
}

func isInsecureCharacter(c byte) bool {
	return c < 32 || c == '\\' || c == '"'
}

var hex = "0123456789abcdef"

func (b *Buffer) WriteJsonSafeBytes(s []byte) *Buffer {
	return b.WriteJsonSafeString(string(s))
}

func (b *Buffer) WriteJsonSafeString(s string) *Buffer {
	if !b.shouldWrite() {
		return b
	}
	return b.writeJsonSafeString(s)
}

// WriteJsonSafeString converts strings with json syntax to safe strings (by adding a '\\' prefix)
func (b *Buffer) writeJsonSafeString(s string) *Buffer {
	i := 0
	for j := 0; j < len(s); {
		if c := s[j]; c < utf8.RuneSelf {
			if !isInsecureCharacter(c) {
				j++
				continue
			}
			if i < j {
				b.WriteString(s[i:j])
			}
			b.WriteByte('\\')
			switch c {
			case '\\', '"':
				b.WriteByte(c)
			case '\n':
				b.WriteByte('n')
			case '\r':
				b.WriteByte('r')
			case '\t':
				b.WriteByte('t')
			default:
				b.WriteString("u00").WriteByte(hex[c>>4]).WriteByte(hex[c&0xF])
			}
			j++
			i = j
			continue
		}
		c, size := utf8.DecodeRuneInString(s[j:])
		if c == utf8.RuneError && size == 1 {
			if i < j {
				b.WriteString(s[i:j])
			}
			b.WriteString(`\ufffd`)
			j += size
			i = j
			continue
		}
		if c == '\u2028' || c == '\u2029' {
			if i < j {
				b.WriteString(s[i:j])
			}
			b.WriteString(`\u202`).WriteByte(hex[c&0xF])
			j += size
			i = j
			continue
		}
		j += size
	}
	if i < len(s) {
		b.WriteString(s[i:])
	}
	return b
}

func (b *Buffer) WriteFloat32(f float32, format byte, precision int) *Buffer {
	if !b.shouldWrite() {
		return b
	}
	b.data = strconv.AppendFloat(b.data, float64(f), format, precision, 32)
	return b
}

func (b *Buffer) WriteFloat64(f float64, format byte, precision int) *Buffer {
	if !b.shouldWrite() {
		return b
	}
	b.data = strconv.AppendFloat(b.data, f, format, precision, 64)
	return b

}

// WritePointer appends the hexadecimal address of p with a "0x" prefix.
// uintptr is like uint, its bitsize depends on the operating system.
func (b *Buffer) WritePointer(p uintptr) *Buffer {
	if !b.shouldWrite() {
		return b
	}
	return b.WriteString("0x").WriteUint(uint(p))
}

func (b *Buffer) WriteBool(v bool) *Buffer {
	if !b.shouldWrite() {
		return b
	}
	b.data = strconv.AppendBool(b.data, v)
	return b
}

func (b *Buffer) WriteTime(t time.Time, format string) *Buffer {
	if !b.shouldWrite() {
		return b
	}
	b.data = t.AppendFormat(b.data, format)
	return b
}

func (b *Buffer) WriteDuration(d time.Duration) *Buffer {
	if !b.shouldWrite() {
		return b
	}
	return b.WriteString(d.String())
}

func (b *Buffer) WriteFromBufferWriter(w BufferWriter) *Buffer {
	if !b.shouldWrite() {
		return b
	}
	w.WriteTo(b)
	return b

}

// WriteAny writes any value to the buffer.
// DO NOT write byte or rune value with this if you want its ASCII representation.
// Use WriteByte or WriteRune instead.
func (b *Buffer) WriteAny(v any) *Buffer {
	switch vv := v.(type) {
	case nil:
		return b.WriteString("<nil>")
	case string:
		return b.WriteString(vv)
	case BufferWriter:
		vv.WriteTo(b)
		return b
	case bool:
		return b.WriteBool(vv)
	case int:
		return b.WriteInt(vv)
	case int8:
		return b.WriteInt8(vv)
	case int16:
		return b.WriteInt16(vv)
	case int32:
		return b.WriteInt32(vv)
	case int64:
		return b.WriteInt64(vv)
	case uint:
		return b.WriteUint(vv)
	case uint8:
		return b.WriteUint8(vv)
	case uint16:
		return b.WriteUint16(vv)
	case uint32:
		return b.WriteUint32(vv)
	case uint64:
		return b.WriteUint64(vv)
	case float32:
		return b.WriteFloat32(vv, 'f', -1)
	case float64:
		return b.WriteFloat64(vv, 'f', -1)
	case []byte:
		return b.WriteBytes(vv)
	case time.Time:
		return b.WriteTime(vv, "2006-01-02 15:04:05")
	case time.Duration:
		return b.WriteDuration(vv)
	case fmt.Stringer:
		return b.WriteString(vv.String())
	case error:
		return b.WriteString(vv.Error())
	}
	b.WriteString(fmt.Sprintf("%+v", v))
	return b
}

func (b *Buffer) If(cond bool) *Buffer {
	b.ifStack = append(b.ifStack, cond)
	return b
}

func (b *Buffer) ElseIf(cond bool) *Buffer {
	if len(b.ifStack) == 0 {
		b.ifStack = append(b.ifStack, false)
	}
	b.ifStack[len(b.ifStack)-1] = !b.ifStack[len(b.ifStack)-1] && cond
	return b
}

func (b *Buffer) Else() *Buffer {
	if len(b.ifStack) == 0 {
		b.ifStack = append(b.ifStack, false)
		return b
	}
	b.ifStack[len(b.ifStack)-1] = !b.ifStack[len(b.ifStack)-1]
	return b
}

func (b *Buffer) EndIf() *Buffer {
	if len(b.ifStack) > 0 {
		b.ifStack = b.ifStack[:len(b.ifStack)-1]
	}
	return b
}

// shouldWrite returns true if no condition is set or the condition is true.
func (b *Buffer) shouldWrite() bool {
	return len(b.ifStack) == 0 || b.ifStack[len(b.ifStack)-1]
}

// For starts a for loop with the given start, end and step.
// Function do will be called for each step.
func (b *Buffer) For(start, end, step int, do func(b *Buffer, i int)) *Buffer {
	if !b.shouldWrite() {
		return b
	}
	b.forStack = append(b.forStack, true)
	for j := start; j < end && b.forStack[len(b.forStack)-1]; j += step {
		do(b, j)
	}
	b.forStack = b.forStack[:len(b.forStack)-1]
	return b
}

// ForNTimes is a shortcut of For(0, n, 1, do);
// i would be 0, 1, 2, ..., n-1.
func (b *Buffer) ForNTimes(n int, do func(b *Buffer, i int)) *Buffer {
	return b.For(0, n, 1, do)
}

// Break breaks the for loop;
// DO NOT use this outside of For, or it may cause panic.
func (b *Buffer) Break() *Buffer {
	if len(b.forStack) > 0 {
		b.forStack[len(b.forStack)-1] = false
	}
	return b
}
