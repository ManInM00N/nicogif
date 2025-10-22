package gifencoder

import "bytes"

// ByteArray implements a growing byte buffer similar to the JavaScript version
type ByteArray struct {
	pages    [][]byte
	page     int
	cursor   int
	pageSize int
}

const defaultPageSize = 4096

// NewByteArray creates a new ByteArray with default page size
func NewByteArray() *ByteArray {
	ba := &ByteArray{
		page:     -1,
		pageSize: defaultPageSize,
		pages:    make([][]byte, 0),
	}
	ba.newPage()
	return ba
}

func (ba *ByteArray) newPage() {
	ba.page++
	ba.pages = append(ba.pages, make([]byte, ba.pageSize))
	ba.cursor = 0
}

// WriteByte writes a single byte to the buffer
func (ba *ByteArray) WriteByte(val byte) {
	if ba.cursor >= ba.pageSize {
		ba.newPage()
	}
	ba.pages[ba.page][ba.cursor] = val
	ba.cursor++
}

// WriteBytes writes a byte slice to the buffer
func (ba *ByteArray) WriteBytes(data []byte) {
	for _, b := range data {
		ba.WriteByte(b)
	}
}

// WriteUTFBytes writes a string as UTF-8 bytes
func (ba *ByteArray) WriteUTFBytes(s string) {
	for i := 0; i < len(s); i++ {
		ba.WriteByte(s[i])
	}
}

// GetData returns all written data as a single byte slice
func (ba *ByteArray) GetData() []byte {
	var buf bytes.Buffer
	for i, page := range ba.pages {
		if i < len(ba.pages)-1 {
			buf.Write(page)
		} else {
			buf.Write(page[:ba.cursor])
		}
	}
	return buf.Bytes()
}

// GetPages returns the internal pages for direct access
func (ba *ByteArray) GetPages() [][]byte {
	return ba.pages
}

// GetCursor returns the current cursor position
func (ba *ByteArray) GetCursor() int {
	return ba.cursor
}

// GetPageSize returns the page size
func (ba *ByteArray) GetPageSize() int {
	return ba.pageSize
}
