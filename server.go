package mux

import (
	"errors"
	"fmt"

	"github.com/ab36245/go-websocket"
)

type Handler func(*Channel)

func Server(ws websocket.Socket, handler Handler) {
	input := make(chan []byte)
	output := make(chan []byte)

	go readSocket(ws, input)

	readers := make(map[uint]chan []byte)
loop:
	for {
		fmt.Printf("Server: reading\n")
		select {
		case bytes, ok := <-input:
			if !ok {
				fmt.Printf("Server: input channel is closed\n")
				break loop
			}
			fmt.Printf("Server: input %d bytes\n", len(bytes))
			id, bytes, err := readNumber(bytes)
			if err != nil {
				fmt.Printf("Server: invalid channel id: %s\n", err)
				break loop
			}
			fmt.Printf("Server: channel id %d\n", id)
			fmt.Printf("Server: bytes remaining %d\n", len(bytes))
			if id == 0 {
				fmt.Printf("Server: control channel\n")
				if len(bytes) == 0 {
					fmt.Printf("Server: empty control message\n")
					break loop
				}
				cmd := bytes[0]
				bytes = bytes[1:]
				switch cmd {
				case 0:
					fmt.Printf("Server: add channel request\n")
					chid, _, err := readNumber(bytes)
					if err != nil {
						fmt.Printf("Server: invalid add channel id: %s\n", err)
						break loop
					}
					fmt.Printf("Server: add channel id %d\n", chid)
					if chid == 0 {
						fmt.Printf("Server: channel id %d is reserved\n", chid)
						break loop
					}
					if _, ok := readers[chid]; ok {
						fmt.Printf("Server: channel id %d is already in use\n", chid)
						break loop
					}
					fmt.Printf("Server: adding channel %d\n", chid)
					reader := make(chan []byte)
					readers[chid] = reader
					go func() {
						ch := &Channel{
							id:     chid,
							reader: reader,
							writer: output,
						}
						fmt.Printf("Server: calling handler\n")
						handler(ch)
						fmt.Printf("Server: handler has completed\n")
						// TODO close reader channel?
						delete(readers, chid)
					}()
				default:
					fmt.Printf("Server: unknown control command\n")
					break loop
				}
			} else if reader, ok := readers[id]; ok {
				fmt.Printf("Server: reader channel\n")
				reader <- bytes
			} else {
				fmt.Printf("Server: invalid channel\n")
				break loop
			}
		case bytes, ok := <-output:
			if !ok {
				fmt.Printf("Server: output channel is closed\n")
				break loop
			}
			fmt.Printf("Server: output %d bytes\n", len(bytes))
			if err := writeSocket(ws, bytes); err != nil {
				fmt.Printf("Server: error writing to socket: %s\n", err)
				break loop
			}
		}
	}
	fmt.Printf("Server: shutting down connection\n")
	for id, reader := range readers {
		fmt.Printf("Server: closing channel %d reader\n", id)
		close(reader)
	}
	ws.Close()
}

func readSocket(ws websocket.Socket, input chan<- []byte) {
	fmt.Printf("readSocket: starting\n")
	for {
		msg, err := ws.Read()
		if errors.Is(err, websocket.ClosedError) {
			fmt.Printf("readSocket: client has closed socket\n")
			break
		}
		if err != nil {
			fmt.Printf("readSocket: got a read error: %s\n", err)
			break
		}
		if !msg.IsBinary() {
			fmt.Printf("readSocket: can't handle %s messages\n", msg.Kind)
			break
		}
		input <- msg.Data
	}
	fmt.Printf("readSocket: closing input channel\n")
	close(input)
}

func writeSocket(ws websocket.Socket, bytes []byte) error {
	fmt.Printf("writeSocket: writing %d bytes\n", len(bytes))
	return ws.WriteBinary(bytes)
}
