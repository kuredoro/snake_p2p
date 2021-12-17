package gather

import (
	"time"

	"github.com/libp2p/go-libp2p-core/peer"
)

type GatherMessageType int

const (
	ConnectionRequest = iota + 1
	Connected
	Disconnected
	GatheringFinished
)

type GatherPointMessage struct {
	ConnectTo          peer.AddrInfo
	TTL                time.Duration
	DesiredPlayerCount uint
	CurrentPlayerCount uint
}

// GatherMessage represents a set of all different messages
// possible to be sent in the gather protocol. This kind of all-in-one
// message is only possible because of the simplicity of the protocol
// bus is potentially inefficient. If any of the message types were to
// have additional fields, the messages of all other types would have
// to have these additional fields, even though they don't need them.
// (TODO: consider protobuf).
type GatherMessage struct {
	Type  GatherMessageType
	Addrs []peer.AddrInfo
}
