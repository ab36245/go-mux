package mux

import (
	"fmt"
	"iter"
)

type Channel struct {
	id     uint
	reader <-chan []byte
	writer chan<- []byte
}

func (c Channel) Id() uint {
	return c.id
}

func (c *Channel) Read() iter.Seq[[]byte] {
	return func(yield func([]byte) bool) {
		for {
			bytes, ok := <-c.reader
			if !ok {
				fmt.Printf("Channel.Read: reader has closed\n")
				break
			}
			if !yield(bytes) {
				fmt.Printf("Channel.Read: consumer is done\n")
				break
			}
		}
	}
}

func (c *Channel) Write(bytes []byte) error {
	buf := writeNumber(c.id)
	buf = append(buf, bytes...)
	c.writer <- buf
	return nil
}
