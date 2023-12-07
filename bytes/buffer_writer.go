package bytes

type BufferWriter interface {
	WriteTo(b *Buffer)
}

type BufferWriteFunc func(b *Buffer)

func (fn BufferWriteFunc) WriteTo(b *Buffer) { fn(b) }
