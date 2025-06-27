package mux

import "fmt"

const controlChannelId uint = 0

type controlMessage struct {
	kind  controlMessageKind
	id    uint
	bytes []byte
}

type controlMessageKind int

const (
	cmRemoteMessage = 0

	cmOpenChannel  = 1 // Must agree with client
	cmCloseChannel = 2 // Must agree with client

	cmFinishChannel = cmCloseChannel + 1 + iota
)

func (k controlMessageKind) String() string {
	switch k {
	case cmRemoteMessage:
		return "cmRemoteMessage"
	case cmOpenChannel:
		return "cmOpenChannel"
	case cmCloseChannel:
		return "cmCloseChannel"
	case cmFinishChannel:
		return "cmFinishChannel"
	default:
		return fmt.Sprintf("unknown (%d)", int(k))
	}
}
