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

	// fakeCh will never transmit anything, if anyone would want to receive
	// from it, they would block forever.
	// XXX: seems like a generic primitive and I've never seen it beeing used,
	// so it is probably an incorrect usage.
	fakeCh := make(chan ping.Result)
	defer close(fakeCh)

	resCh := h.ping.Ping(ctx, h.peer)

	timer := time.NewTimer(HeartbeatEvery)
	for {
		select {
		case <-h.done:
			cancel()
			close(h.done)
			return
		case res := <-resCh:
			if res.Error != nil {
				if h.peerStatus != dead {
					h.peerStatus = dead

					// XXX What the hell is this???
					// The problem is that when join service calls Close
					// the select chooses the done channel and quits the loop,
					// and then it waits us to send the status on channel that
					// it does not listen anymore...
					// Wait... why not close heartbeats before the done ch?
					h.reportCh <- PeerStatus{
						Peer:  h.peer,
						Alive: false,
					}
				}

				resCh = fakeCh
				continue
			}

			if h.peerStatus != alive {
				h.peerStatus = alive

				h.reportCh <- PeerStatus{
					Peer:  h.peer,
					Alive: true,
				}
			}

			resCh = fakeCh
		case <-timer.C:
			cancel()
			ctx, cancel = context.WithCancel(context.Background())

			resCh = h.ping.Ping(ctx, h.peer)
			timer.Reset(HeartbeatEvery)
		}
	}
}

func (h *HeartbeatService) Close() {
	h.done <- struct{}{}
	<-h.done
}
