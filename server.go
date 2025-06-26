package mux

import (
	"errors"
	"fmt"

	"github.com/ab36245/go-websocket"
)

func NewServer(ws websocket.Socket, handler Handler) *Server {
	input := make(chan []byte)
	output := make(chan []byte)
	readers := make(map[uint]chan []byte)
	return &Server{
		handler: handler,
		input:   input,
		output:  output,
		readers: readers,
		ws:      ws,
	}
}

type Server struct {
	handler Handler
	input   chan []byte
	output  chan []byte
	readers map[uint]chan []byte
	ws      websocket.Socket
	// mutex?
}

func (s *Server) Run() {
	go s.doSocket()
	go s.doInput()
	s.doOutput()
}

func (s *Server) doSocket() {
	m := "Server.doSocket"
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
		s.input <- msg.Data
	}
	fmt.Printf("%s: closing input channel\n", m)
	close(s.input)
}

func (s *Server) doInput() {
	m := "Server.doInput"
	fmt.Printf("%s: starting\n", m)
	for {
		fmt.Printf("%s: reading\n", m)
		bytes, ok := <-s.input
		if !ok {
			fmt.Printf("%s: input channel is closed\n", m)
			break
		}
		fmt.Printf("%s: input %d bytes\n", m, len(bytes))
		id, bytes, err := readNumber(bytes)
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
	for id, reader := range s.readers {
		fmt.Printf("%s: closing channel %d reader\n", m, id)
		close(reader)
	}
	fmt.Printf("%s: closing websocket\n", m)
	s.ws.Close()
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
		s.doAddChannel(bytes)
	default:
		fmt.Printf("%s: unknown control command %d\n", m, cmd)
	}
}

func (s *Server) doAddChannel(bytes []byte) {
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
	}()
}

type Handler func(*Channel)

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
