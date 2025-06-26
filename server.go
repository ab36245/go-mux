package mux

import (
	"errors"
	"fmt"

	"github.com/ab36245/go-websocket"
)

type Handler func(*Channel)

func NewServer(ws websocket.Socket, handler Handler) *Server {
	output := make(chan []byte)
	readers := make(map[uint]chan []byte)
	return &Server{
		handler: handler,
		output:  output,
		readers: readers,
		ws:      ws,
	}
}

type Server struct {
	closing bool
	handler Handler
	output  chan []byte
	readers map[uint]chan []byte
	ws      websocket.Socket
	// mutex?
}

func (s *Server) Run() {
	go s.doInput()
	s.doOutput()
}

func (s *Server) doInput() {
	m := "Server.doInput"
	fmt.Printf("%s: starting\n", m)
	for {
		msg, err := s.ws.Read()
		if errors.Is(err, websocket.ClosedError) {
			fmt.Printf("%s: client has closed socket\n", m)
			break
		}
		if err != nil {
			fmt.Printf("%s: got a read error: %s\n", m, err)
			break
		}
		if !msg.IsBinary() {
			fmt.Printf("%s: can't handle %s messages\n", m, msg.Kind)
			break
		}
		fmt.Printf("%s: read message %s\n", m, msg)
		fmt.Printf("%s: reading channel id\n", m)
		id, bytes, err := readNumber(msg.Data)
		if err != nil {
			fmt.Printf("%s: invalid channel id: %s\n", m, err)
			break
		}
		fmt.Printf("%s: channel id %d\n", m, id)
		fmt.Printf("%s: bytes remaining %d\n", m, len(bytes))
		if id == 0 {
			fmt.Printf("%s: control channel\n", m)
			s.doControl(bytes)
		} else if reader, ok := s.readers[id]; ok {
			fmt.Printf("%s: reader channel\n", m)
			reader <- bytes
		} else {
			fmt.Printf("%s: invalid channel\n", m)
			break
		}
	}
	fmt.Printf("%s: shutting down connection\n", m)
	s.closing = true
	for id, reader := range s.readers {
		fmt.Printf("%s: closing channel %d reader\n", m, id)
		close(reader)
	}
}

func (s *Server) doControl(bytes []byte) {
	m := "Server.doControl"
	if len(bytes) == 0 {
		fmt.Printf("%s: empty control message\n", m)
		return
	}
	cmd := bytes[0]
	bytes = bytes[1:]
	switch cmd {
	case 0:
		fmt.Printf("%s: add channel command\n", m)
		s.doControlAddChannel(bytes)
	default:
		fmt.Printf("%s: unknown control command %d\n", m, cmd)
	}
}

func (s *Server) doControlAddChannel(bytes []byte) {
	m := "Server.doAddChannel"
	chid, _, err := readNumber(bytes)
	if err != nil {
		fmt.Printf("%s: invalid add channel id: %s\n", m, err)
		return
	}
	fmt.Printf("%s: add channel id %d\n", m, chid)
	if chid == 0 {
		fmt.Printf("%s: channel id %d is reserved\n", m, chid)
		return
	}
	if _, ok := s.readers[chid]; ok {
		fmt.Printf("%s: channel id %d is already in use\n", m, chid)
		return
	}
	fmt.Printf("%s: adding channel %d\n", m, chid)
	reader := make(chan []byte)
	// lock
	s.readers[chid] = reader
	// unlock
	go func() {
		ch := &Channel{
			id:     chid,
			reader: reader,
			writer: s.output,
		}
		fmt.Printf("%s: calling handler\n", m)
		s.handler(ch)
		fmt.Printf("%s: handler has completed\n", m)
		// TODO close reader channel?
		// lock
		delete(s.readers, chid)
		// unlock
		fmt.Printf("%s: no. of channels remaining is %d\n", m, len(s.readers))
		if len(s.readers) == 0 {
			fmt.Printf("%s: no remaining channels\n", m)
			fmt.Printf("%s: closing flag is %v\n", m, s.closing)
			if s.closing {
				fmt.Printf("%s: closing output channel\n", m)
				close(s.output)
			}
		}
	}()
}

func (s *Server) doOutput() {
	m := "Server.doOutput"
	fmt.Printf("%s: starting\n", m)
	for {
		bytes, ok := <-s.output
		if !ok {
			fmt.Printf("%s: output channel is closed\n", m)
			break
		}
		fmt.Printf("%s: output %d bytes\n", m, len(bytes))
		err := s.ws.WriteBinary(bytes)
		if err != nil {
			fmt.Printf("%s: error writing to socket: %s\n", m, err)
			break
		}
	}
}
