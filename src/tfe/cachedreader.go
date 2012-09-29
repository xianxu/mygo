package tfe

import (
	"io"
	"errors"
	"io/ioutil"
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
	n = 0
	if c.Closed {
		err = errors.New("reader closed")
		return
	}
	if c.Bytes == nil {
		err = errors.New("no data")
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
	} else {
		copy(p, c.Bytes[offset:offset+space])
		c.Offset += space
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

func NewCachedReader(b io.ReadCloser) (cr *CachedReader, err error) {
	bytes, err := ioutil.ReadAll(b)
	if err != nil {
		return
	}
	cr = &CachedReader{bytes, 0, false}
	return
}
