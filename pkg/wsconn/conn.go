package wsconn

import (
	"io"
	"log"

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

func (c Conn) Write(b []byte) (int, error) {
	if err := c.WriteMessage(websocket.BinaryMessage, b); err != nil {
		return 0, err
	}
	return len(b), nil
}

func (c Conn) Read(b []byte) (int, error) {
	if c.reader == nil {
		t, r, err := c.NextReader()
		if err != nil {
			// if the connection is closed, we returns EOF instead of
			// ErrUnexpectedEOF so io.Copy behaves sanely
			if err == io.ErrUnexpectedEOF {
				return 0, io.EOF
			}
			return 0, err
		}
		if t != websocket.BinaryMessage {
			log.Printf("ws: ignored non-binary message in stream")
			return 0, nil
		}
		c.reader = r
	}
	n, err := c.reader.Read(b)
	if err != nil {
		if err == io.EOF {
			c.reader = nil
		}
		return n, err
	}
	return n, nil
}

func (c Conn) Close() error {
	return c.Conn.Close()
}
