package gather

import (
	"time"

	"github.com/libp2p/go-libp2p-core/peer"
)

type GatherPointMessage struct {
	ConnectTo          peer.AddrInfo
	TTL                time.Duration
	DesiredPlayerCount uint
	CurrentPlayerCount uint
}
