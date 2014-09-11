package wsconn

import (
	"io"

	"github.com/gorilla/websocket"
)

// Currently the connection is only used for sending text message i.e. (utf-8
// encoded messages)
type Conn struct {
	*websocket.Conn
	reader io.Reader // cached reader, which is reset whenever the messages is EOF
}

func NewConnection(w *websocket.Conn) *Conn {
	return &Conn{Conn: w}
}

func (c *Conn) Write(b []byte) (int, error) {
	w, err := c.Conn.NextWriter(websocket.BinaryMessage)
	if err != nil {
		return 0, err
	}
	n, err := w.Write(b)
	if err != nil {
		return n, err
	}
	err = w.Close()
	if err != nil {
		return n, err
	}
	return n, nil
}

func (c *Conn) Read(b []byte) (int, error) {
	if c.reader == nil {
		t, r, err := c.NextReader()
		if err != nil {
			return 0, err
		}
		if t != websocket.BinaryMessage {
			return 0, nil
		}
		c.reader = r
	}
	n, err := c.reader.Read(b)
	if err != nil {
		// if we finished reading the current frame, reset the reader
		// and treats EOF as no-op because we might have more frames
		if err == io.EOF {
			c.reader = nil
			err = nil
		}
		return n, err
	}
	return n, nil
}

func (c *Conn) Close() error {
	c.WriteMessage(websocket.CloseMessage, []byte{})
	return c.Conn.Close()
}
