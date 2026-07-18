package telnet

import (
	"fmt"
	"io"
	"net"
	"time"
)

const (
	iac  = 255
	will = 251
	wont = 252
	do   = 253
	dont = 254
	sb   = 250
	se   = 240
)

type Connector interface {
	ReadBytes(delim byte) ([]byte, error)
	Write(cmd string) error
	Close() error
}

var _ Connector = (*Conn)(nil)

type Conn struct {
	conn net.Conn
}

func NewConn(host, port string) (*Conn, error) {
	addr := fmt.Sprintf("%s:%s", host, port)
	conn, err := net.DialTimeout("tcp", addr, 10*time.Second)
	if err != nil {
		return nil, err
	}
	return &Conn{conn: conn}, nil
}

func (c *Conn) ReadBytes(delim byte) ([]byte, error) {
	return filterTelnetWithReply(c.conn, delim)
}

func (c *Conn) Write(cmd string) error {
	_, err := fmt.Fprintf(c.conn, "%s\r\n", cmd)
	return err
}

func (c *Conn) Close() error {
	return c.conn.Close()
}

func filterTelnetWithReply(rw io.ReadWriter, delim byte) ([]byte, error) {
	var buf []byte
	b := make([]byte, 1)
	for {
		_, err := rw.Read(b)
		if err != nil {
			if len(buf) > 0 {
				return buf, err
			}
			return nil, err
		}
		if b[0] == iac {
			if _, err = rw.Read(b); err != nil {
				return buf, err
			}
			switch b[0] {
			case iac:
				buf = append(buf, 0xFF)
			case will:
				rw.Read(b)
				rw.Write([]byte{iac, dont, b[0]})
			case do:
				rw.Read(b)
				rw.Write([]byte{iac, wont, b[0]})
			case wont, dont:
				rw.Read(b)
			case sb:
				for {
					rw.Read(b)
					if b[0] == iac {
						rw.Read(b)
						if b[0] == se {
							break
						}
					}
				}
			default:
				rw.Read(b)
			}
		} else if b[0] == delim {
			return buf, nil
		} else {
			buf = append(buf, b[0])
		}
	}
}

func filterTelnet(r io.Reader, delim byte) ([]byte, error) {
	var buf []byte
	b := make([]byte, 1)
	for {
		_, err := r.Read(b)
		if err != nil {
			if len(buf) > 0 {
				return buf, err
			}
			return nil, err
		}
		if b[0] == iac {
			if _, err = r.Read(b); err != nil {
				return buf, err
			}
			switch b[0] {
			case iac:
				buf = append(buf, 0xFF)
			case will, wont, do, dont:
				r.Read(b)
			case sb:
				for {
					r.Read(b)
					if b[0] == iac {
						r.Read(b)
						if b[0] == se {
							break
						}
					}
				}
			default:
				r.Read(b)
			}
		} else if b[0] == delim {
			return buf, nil
		} else {
			buf = append(buf, b[0])
		}
	}
}
