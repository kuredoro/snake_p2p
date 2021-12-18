package heartbeat

import (
	"context"
	"errors"
	"time"

	"github.com/libp2p/go-libp2p-core/peer"
	"github.com/libp2p/go-libp2p/p2p/protocol/ping"
)

var HeartbeatEvery = 1 * time.Second

type status int

const (
	unknown status = iota
	alive
	dead
)

type PeerStatus struct {
	Peer  peer.ID
	Alive bool
}

type HeartbeatService struct {
	done chan struct{}

	ping *ping.PingService
	peer peer.ID

	peerStatus status

	reportCh chan PeerStatus
}

func NewHeartbeat(ping *ping.PingService, p peer.ID, outCh chan PeerStatus) (*HeartbeatService, error) {
	if ping == nil {
		return nil, errors.New("ping service is nil")
	}

	hb := &HeartbeatService{
		done: make(chan struct{}),

		ping: ping,
		peer: p,

		peerStatus: unknown,

		reportCh: outCh,
	}

	go hb.run()

	return hb, nil
}

func (h *HeartbeatService) run() {
	ctx, cancel := context.WithCancel(context.Background())

	for {
		select {
		case <-h.done:
			cancel()
			close(h.done)
			return
		case res := <-h.ping.Ping(ctx, h.peer):
			if res.Error != nil {
				if h.peerStatus != dead {
					h.reportCh <- PeerStatus{
						Peer:  h.peer,
						Alive: false,
					}

					h.peerStatus = dead
				}

				time.Sleep(HeartbeatEvery)
				continue
			}

			if h.peerStatus != alive {
				h.reportCh <- PeerStatus{
					Peer:  h.peer,
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
}

func (h *HeartbeatService) Close() {
	h.done <- struct{}{}
	<-h.done
}
