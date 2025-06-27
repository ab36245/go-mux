package mux

import (
	"errors"

	"github.com/rs/zerolog/log"

	"github.com/ab36245/go-websocket"
)

type Handler func(*Channel)

func NewServer(socket websocket.Socket, handler Handler) *Server {
	channels := make(map[uint]*Channel)
	control := make(chan controlMessage)
	output := make(chan []byte)
	return &Server{
		channels: channels,
		closing:  false,
		control:  control,
		handler:  handler,
		output:   output,
		socket:   socket,
	}
}

type Server struct {
	channels map[uint]*Channel
	closing  bool
	control  chan controlMessage
	handler  Handler
	output   chan []byte
	socket   websocket.Socket
	// mutex?
}

func (s *Server) Run() {
	go s.doInput()
	go s.doOutput()
	s.doControl()
}

func (s *Server) doControl() {
loop:
	for {
		log.Trace().Msg("waiting for control message")
		msg, ok := <-s.control
		if !ok {
			log.Error().Msg("control channel has closed unexpectedly!")
			break
		}
		log.Trace().Stringer("kind", msg.kind).Msg("handling control message")
		switch msg.kind {
		case cmRemoteMessage:
			s.doRemoteMessage(msg.bytes)
		case cmOpenChannel:
			s.doOpenChannel(msg.id)
		case cmCloseChannel:
			s.doCloseChannel(msg.id)
		case cmFinishChannel:
			if s.doFinishChannel(msg.id) {
				break loop
			}
		default:
			log.Error().Stringer("kind", msg.kind).Msg("not handled")
		}
	}
}

func (s *Server) doInput() {
	for {
		log.Trace().Msg("waiting for socket")
		msg, err := s.socket.Read()
		if errors.Is(err, websocket.ClosedError) {
			log.Debug().Msg("client has closed socket")
			break
		}
		if err != nil {
			log.Error().Err(err).Msg("socket read returned an error")
			break
		}
		log.Trace().Stringer("kind", msg.Kind).Msg("read message")
		if !msg.IsBinary() {
			log.Error().Stringer("kind", msg.Kind).Msg("can't handle message")
			break
		}
		log.Trace().Msg("reading channel id")
		id, bytes, err := readNumber(msg.Data)
		if err != nil {
			log.Error().Err(err).Msg("invalid channel id")
			break
		}
		log.Trace().Uint("id", id).Int("bytes", len(bytes)).Msg("got valid id")
		if id == controlChannelId {
			log.Trace().Msg("control channel")
			s.control <- controlMessage{
				kind:  cmRemoteMessage,
				bytes: bytes,
			}
		} else if ch, ok := s.channels[id]; ok {
			log.Trace().Msg("valid channel")
			ch.input <- bytes
		} else {
			log.Error().Msg("invalid channel")
			break
		}
	}
	log.Trace().Msg("stopping")
	s.closing = true
	for id := range s.channels {
		s.control <- controlMessage{
			kind: cmCloseChannel,
			id:   id,
		}
	}
}

func (s *Server) doOutput() {
	for {
		log.Trace().Msg("waiting for output")
		bytes, ok := <-s.output
		if !ok {
			log.Debug().Msg("output channel is closed")
			break
		}
		log.Trace().Int("bytes", len(bytes)).Msg("read bytes")
		err := s.socket.WriteBinary(bytes)
		if err != nil {
			log.Error().Err(err).Msg("writing to socket failed")
			break
		}
	}
}

func (s *Server) doRemoteMessage(bytes []byte) {
	log.Trace().Int("bytes", len(bytes)).Msg("got control message from remote")
	if len(bytes) == 0 {
		log.Error().Msg("empty control message")
		log.Warn().Msg("this should shutdown the entire connection")
		return
	}
	cmd := controlMessageKind(bytes[0])
	log.Trace().Stringer("cmd", cmd).Msg("got command")
	bytes = bytes[1:]
	switch cmd {
	case cmOpenChannel:
		id, _, err := readNumber(bytes)
		if err != nil {
			log.Error().Err(err).Msg("invalid channel")
			return
		}
		s.doOpenChannel(id)
	default:
		log.Error().Stringer("cmd", cmd).Msg("unknown command")
		log.Warn().Msg("this should shutdown the entire connection")
		return
	}
}

func (s *Server) doOpenChannel(id uint) {
	log.Trace().Uint("id", id).Msg("opening channel")
	if id == controlChannelId {
		log.Error().Uint("id", id).Msg("channel id is reserved")
		log.Warn().Msg("this should shutdown the entire connection")
		return
	}
	if _, ok := s.channels[id]; ok {
		log.Error().Uint("id", id).Msg("channel is already in use")
		log.Warn().Msg("this should shutdown the entire connection")
		return
	}
	log.Trace().Uint("id", id).Msg("creating channel")
	ch := &Channel{
		control: s.control,
		id:      id,
		input:   make(chan []byte),
		output:  s.output,
		state:   csOpen,
	}

	// lock?
	s.channels[id] = ch
	// unlock?

	log.Trace().Uint("id", id).Msg("spawning go routine")
	go func() {
		log.Trace().Msg("calling handler")
		s.handler(ch)
		log.Trace().Uint("id", ch.id).Msg("handler has completed")
		s.control <- controlMessage{
			kind: cmFinishChannel,
			id:   ch.id,
		}
	}()

	log.Trace().Uint("id", id).Msg("finished opening channel")
}

func (s *Server) doCloseChannel(id uint) {
	log.Trace().Uint("id", id).Msg("closing channel")
	ch, ok := s.channels[id]
	if !ok {
		log.Trace().Uint("id", id).Msg("not found")
		log.Warn().Msg("this should shutdown the entire connection")
		return
	}
	log.Trace().Uint("id", id).Msg("closing input")
	close(ch.input)
}

func (s *Server) doFinishChannel(id uint) bool {
	log.Trace().Uint("id", id).Msg("finishing channel")

	// lock?
	delete(s.channels, id)
	// unlock?

	count := len(s.channels)
	log.Trace().Int("count", count).Msg("channels still open")

	shutdown := count == 0 && s.closing
	if shutdown {
		log.Debug().Msg("shutting down")
	}
	return shutdown
}
