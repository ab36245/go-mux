package mux

import (
	"iter"

	"github.com/rs/zerolog/log"
)

type Channel struct {
	control chan<- controlMessage
	id      uint
	input   chan []byte
	output  chan<- []byte
	state   channelState
}

func (c Channel) Id() uint {
	return c.id
}

func (c *Channel) Close() {
	if c.state == csOpen {
		c.control <- controlMessage{
			kind: cmCloseChannel,
			id:   c.id,
		}
	}
}

func (c *Channel) Read() iter.Seq[[]byte] {
	return func(yield func([]byte) bool) {
		log.Trace().Uint("id", c.id).Msg("iterator starting")
		for {
			log.Trace().Uint("id", c.id).Msg("reading input")
			bytes, ok := <-c.input
			if !ok {
				log.Debug().Uint("id", c.id).Msg("input channel has closed")
				break
			}
			if !yield(bytes) {
				log.Debug().Uint("id", c.id).Msg("iterator consumer is done")
				break
			}
		}
		log.Trace().Uint("id", c.id).Msg("iterator is done")
	}
}

func (c *Channel) Write(bytes []byte) error {
	log.Trace().Uint("id", c.id).Int("bytes", len(bytes)).Msg("writing")
	buf := writeNumber(c.id)
	buf = append(buf, bytes...)
	c.output <- buf
	return nil
}

type channelState int

const (
	csOpen channelState = iota
	csClosing
)
