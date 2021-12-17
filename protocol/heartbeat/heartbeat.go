package heartbeat

import (
	"errors"
	"time"

	"github.com/libp2p/go-libp2p-core/peer"
	"github.com/libp2p/go-libp2p/p2p/protocol/ping"
	"golang.org/x/net/context"
)

var HeartbeatEvery = 1 * time.Second

type status int

const (
	unknown status = iota
	alive
	dead
)

type PeerStatus struct {
	ID    peer.ID
	Alive bool
}

type HeartbeatService struct {
	ctx   context.Context
	Close func()

	ping *ping.PingService
	peer peer.ID

	peerStatus status

	reportCh chan PeerStatus
}

func NewHeartbeat(ctx context.Context, ping *ping.PingService, p peer.ID, outCh chan PeerStatus) (*HeartbeatService, error) {
	if ping == nil {
		return nil, errors.New("ping service is nil")
	}

	localCtx, cancel := context.WithCancel(ctx)

	hb := &HeartbeatService{
		ctx:   localCtx,
		Close: cancel,
		ping:  ping,
		peer:  p,

		peerStatus: unknown,

		reportCh: outCh,
	}

	go hb.run()

	return hb, nil
}

func (h *HeartbeatService) run() {
	for {
		res := <-h.ping.Ping(h.ctx, h.peer)

		if h.ctx.Err() != nil {
			return
		}

		if res.Error != nil && h.peerStatus != dead {
			h.reportCh <- PeerStatus{
				ID:    h.peer,
				Alive: false,
			}

			h.peerStatus = dead
			continue
		}

		if h.peerStatus != alive {
			h.reportCh <- PeerStatus{
				ID:    h.peer,
				Alive: true,
			}

			h.peerStatus = alive
		}

		// Note: sleeping here for HeartbeatEvery seconds makes heartbeats
		// fire a bit less frequently than desired. That's because we do
		// not take away the amount of time Ping has taken from the
		// HeartbeatEvery, so the heartbeat is fired every HeartbeatEvery +
		// 'how much Ping has taken'. Since Ping has a timeout, the heartbeat
		// will be fired eventually, but making it fire every HeartbeatEvery
		// exactly will clutter this function, which doesn't seem worth it.
		time.Sleep(HeartbeatEvery)
	}
}
