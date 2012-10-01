package tfe

import (
	"io"
	"errors"
	"io/ioutil"
)

var (
	errReaderClosed = errors.New("reader closed")
	errNoData       = errors.New("no data")
)

/*
* Implements a Reader that can be repeated read.
 */
type CachedReader struct {
	Bytes  []byte
	Offset int
	Closed bool
}

func (c *CachedReader) Read(p []byte) (n int, err error) {
	err = nil
	if c.Closed {
		err = errReaderClosed
		return
	}
	if c.Bytes == nil {
		err = errNoData
		return
	}

	offset := c.Offset
	remaining := len(c.Bytes) - offset
	space := len(p)

	if remaining == 0 {
		err = io.EOF
		return
	}

	if remaining <= space {
		copy(p, c.Bytes[offset:])
		c.Offset += remaining
		n = remaining
	} else {
		copy(p, c.Bytes[offset:offset+space])
		c.Offset += space
		n = space
	}
	return
}

func (c *CachedReader) Close() error {
	c.Closed = true
	return nil
}

func (c *CachedReader) Reset() {
	c.Offset = 0
	c.Closed = false
}

func NewCachedReader(b io.Reader) (cr *CachedReader, err error) {
	bytes, err := ioutil.ReadAll(b)
	if err != nil {
		return
	}
	cr = &CachedReader{bytes, 0, false}
	return
}
