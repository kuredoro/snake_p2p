package gather

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"sync"
	"time"

	"github.com/libp2p/go-libp2p-core/host"
	"github.com/libp2p/go-libp2p-core/network"
	"github.com/libp2p/go-libp2p-core/peer"
	pubsub "github.com/libp2p/go-libp2p-pubsub"
	"github.com/libp2p/go-libp2p/p2p/protocol/ping"

	"github.com/hashicorp/go-multierror"

	"github.com/kuredoro/snake_p2p/core"
	"github.com/kuredoro/snake_p2p/protocol/game"
	"github.com/kuredoro/snake_p2p/protocol/heartbeat"

	"github.com/rs/zerolog/log"
)

func HostAddrInfo(h host.Host) *peer.AddrInfo {
	return &peer.AddrInfo{
		ID:    h.ID(),
		Addrs: h.Addrs(),
	}
}

type GatherService struct {
	monitorDone, meshUpdateDone chan struct{}
	done                        bool

	h       host.Host
	streams map[peer.ID]network.Stream
	topic   *pubsub.Topic

	ttl          time.Duration
	desiredCount int

	mesh   peerMesh
	meshCh chan peerMeshMod

	ping             *ping.PingService
	game             *game.GameService
	conns            map[peer.ID]*heartbeat.HeartbeatService
	localConnUpdates chan heartbeat.PeerStatus

	// TODO: move beacon to snake.Node
	beacon *GatherPointBeacon

	gameCh chan<- game.GameEstablished
}

func NewGatherService(h host.Host, topic *pubsub.Topic, game *game.GameService, ping *ping.PingService, n int, TTL time.Duration, gameCh chan<- game.GameEstablished) (*GatherService, error) {
	gs := &GatherService{
		monitorDone:    make(chan struct{}),
		meshUpdateDone: make(chan struct{}),

		h:       h,
		streams: make(map[peer.ID]network.Stream),
		topic:   topic,

		ttl:          TTL,
		desiredCount: n,

		mesh:   make(peerMesh),
		meshCh: make(chan peerMeshMod),

		ping:             ping,
		game:             game,
		conns:            make(map[peer.ID]*heartbeat.HeartbeatService),
		localConnUpdates: make(chan heartbeat.PeerStatus),

		beacon: NewGatherPointBeacon(topic, *HostAddrInfo(h), n, TTL),

		gameCh: gameCh,
	}

	h.SetStreamHandler(ID, gs.GatherHandler)

	go gs.monitorLoop()
	go gs.meshUpdateLoop()

	return gs, nil
}

func (gs *GatherService) GatherHandler(stream network.Stream) {
	if gs.done {
		stream.Close()
		return
	}

	peer := stream.Conn().RemotePeer()
	log.Info().Str("id", peer.Pretty()).Msg("Seeker connected")

	hb, err := heartbeat.NewHeartbeat(gs.ping, stream.Conn().RemotePeer(), gs.localConnUpdates)
	if err != nil {
		panic(err)
	}

	gs.streams[peer] = stream
	gs.conns[peer] = hb

	// Proto start
	scanner := bufio.NewScanner(stream)
	readCh := make(chan bool)
	defer close(readCh)

	scan := func() {
		readCh <- scanner.Scan()
	}

	go scan()

	remotePeer := stream.Conn().RemotePeer()

	// TODO: writing to streams should probably be done from this function
	// for synchronization purposes, but maybe stream.Write is thread-safe...
	for ok := range readCh {
		if !ok {
			log.Info().Str("id", peer.Pretty()).Msg("Seeker withdrawn")
			return
		}

		var msg GatherMessage
		err := json.Unmarshal(scanner.Bytes(), &msg)
		if err != nil {
			log.Warn().Err(err).Str("seeker", remotePeer.Pretty()).Msg("Unmarshal JSON")
			go scan()
			continue
		}

		switch msg.Type {
		case Connected:
			if len(msg.Addrs) == 0 {
				log.Warn().
					Str("seeker", remotePeer.Pretty()).
					Msg("Seeker-seeker connection message lists no peers")
				break
			}

			log.Info().
				Str("from", remotePeer.Pretty()).
				Str("to", msg.Addrs[0].ID.Pretty()).
				Msg("New seeker-seeker connection")

			gs.meshCh <- addDoubleEdge(remotePeer, msg.Addrs[0].ID)
		case Disconnected:
			if len(msg.Addrs) == 0 {
				log.Warn().
					Str("seeker", remotePeer.Pretty()).
					Msg("Seeker-seeker connection reset message lists no peers")
				break
			}

			log.Info().
				Str("from", remotePeer.Pretty()).
				Str("to", msg.Addrs[0].ID.Pretty()).
				Msg("Seeker-seeker connection reset")

			gs.meshCh <- removeDoubleEdge(remotePeer, msg.Addrs[0].ID)
		default:
			log.Warn().
				Str("seeker", remotePeer.Pretty()).
				Int("type", int(msg.Type)).
				Msg("Gathering message of unknown type")
		}

		go scan()
	}
}

func (gs *GatherService) meshUpdateLoop() {
	scanResults := make(chan []peer.ID)
	defer close(scanResults)

	for {
		select {
		case <-gs.meshUpdateDone:
			close(gs.meshUpdateDone)
			return
		case mod := <-gs.meshCh:
			rescan := mod(gs.mesh)

			log.Debug().Msgf("Mesh updated:\n%v", gs.mesh)

			if !rescan {
				continue
			}

			clique := gs.mesh.FindClique(gs.desiredCount, gs.h.ID())
			if clique == nil {
				log.Debug().Msg("No cliques found")
				continue
			}

			log.Info().Msgf("Clique found %v", clique)

			gs.done = true

			addrs := make([]peer.AddrInfo, len(clique))
			for i, id := range clique {
				addrs[i].ID = id
			}

			msg := GatherMessage{
				Type:  GatheringFinished,
				Addrs: addrs,
			}

			raw, err := json.Marshal(&msg)
			if err != nil {
				panic(err)
			}

			raw = append(raw, '\n')

			var wg sync.WaitGroup
			wg.Add(len(gs.streams))
			for id, stream := range gs.streams {
				go func(id peer.ID, s network.Stream) {
					_, err := s.Write(raw)
					if err != nil {
						log.Err(err).Str("seeker", id.Pretty()).Msg("Send gathering finished message")
					}
					wg.Done()
				}(id, stream)
			}

			wg.Wait()

			gs.closeHeartbeats()

			for id := range gs.streams {
				// JoinService will close the stream itself
				// TODO: delete loop or create new? Does it even matter,
				// this service should be garbage collected...
				delete(gs.streams, id)
			}

			gs.gameCh <- game.GameEstablished{
				Facilitator: gs.h.ID(),
				Game:        gs.game.GetInstance(),
			}
		}
	}
}

func (gs *GatherService) monitorLoop() {
	for {
		select {
		case <-gs.monitorDone:
			close(gs.monitorDone)
			return
		case peerStatus := <-gs.localConnUpdates:
			switch peerStatus.Alive {
			case true:
				err := gs.game.Connect(context.Background(), peerStatus.Peer)
				if err != nil {
					log.Err(err).
						Str("peer", peerStatus.Peer.Pretty()).
						Msg("Create new game connection")
					continue
				}

				gs.meshCh <- addDoubleEdge(gs.h.ID(), peerStatus.Peer)

				log.Info().
					Str("seeker", peerStatus.Peer.Pretty()).
					Msg("Facilitator-seeker connection established")

				err = gs.askEverybodyToConnectTo(peerStatus.Peer)
				if err != nil {
					merr := err.(*multierror.Error)
					for _, err := range merr.Errors {
						peerErr := err.(*core.PeerError)
						gs.peerDisconnected(peerErr.Peer)
						log.Err(peerErr.Err).
							Str("seeker", peerErr.Peer.Pretty()).
							Msg("Request seeker-seeker connection")
					}
				}
			case false:
				gs.game.Disconnect(peerStatus.Peer)

				gs.meshCh <- removeDoubleEdge(gs.h.ID(), peerStatus.Peer)

				gs.peerDisconnected(peerStatus.Peer)
				/*
									   The main stream may still be alive

					                   UPD: decided that if ping dies, the whole thing dies...
				*/

				log.Info().
					Str("seeker", peerStatus.Peer.Pretty()).
					Msg("Facilitator-seeker connection reset")
			}
		}
	}
}

func (gs *GatherService) peerDisconnected(p peer.ID) {
	if _, connected := gs.streams[p]; !connected {
		return
	}

	gs.streams[p].Close()
	delete(gs.streams, p)

	gs.conns[p].Close()
	delete(gs.conns, p)

	gs.meshCh <- removePeer(p)
}

func (gs *GatherService) askEverybodyToConnectTo(peer peer.ID) (merr error) {
	errCh := make(chan error)

	msgCount := 0
	for srcID := range gs.mesh {
		if srcID == peer || srcID == gs.h.ID() {
			continue
		}
		log.Debug().
			Str("from", srcID.Pretty()).
			Str("to", peer.Pretty()).
			Msg("Requesting seeker-seeker connection")

		stream, ok := gs.streams[srcID]
		if !ok {
			log.Error().
				Str("seeker", srcID.Pretty()).
				Msg("Mesh lists a connected for which there's no stream")
			continue
		}

		go func() {
			errCh <- gs.askPeerToConnectTo(stream, gs.h.Peerstore().PeerInfo(peer))
		}()

		msgCount++
	}

	// Sanity check
	/* Doesn't work right now (expected is wrong)
	    // TODO: Think about the logic and concurrency here...
		expectedCount := len(gs.mesh) - 1
		if msgCount != expectedCount {
			log.Warn().
				Int("count", msgCount).
				Int("expected", expectedCount).
				Msg("Expected to ask to connected a different number of peers")
		}
	*/

	for i := 0; i < msgCount; i++ {
		err := <-errCh
		if err != nil {
			merr = multierror.Append(merr, fmt.Errorf("send connection request: %v", err))
		}
	}

	close(errCh)

	return
}

func (gs *GatherService) askPeerToConnectTo(stream network.Stream, pi peer.AddrInfo) error {
	msg := GatherMessage{
		Type:  ConnectionRequest,
		Addrs: []peer.AddrInfo{pi},
	}

	raw, err := json.Marshal(&msg)
	if err != nil {
		return &core.PeerError{
			Peer: pi.ID,
			Err:  fmt.Errorf("marshal: %v", err),
		}
	}

	raw = append(raw, '\n')

	_, err = stream.Write(raw)
	if err != nil {
		return &core.PeerError{
			Peer: pi.ID,
			Err:  fmt.Errorf("send: %v", err),
		}
	}

	return nil
}

func (gs *GatherService) closeHeartbeats() {
	var wg sync.WaitGroup
	wg.Add(len(gs.conns))

	for _, hb := range gs.conns {
		go func(hb *heartbeat.HeartbeatService) {
			hb.Close()
			wg.Done()
		}(hb)
	}

	wg.Wait()

	for id := range gs.conns {
		delete(gs.conns, id)
	}
}

func (gs *GatherService) Close() {
	gs.monitorDone <- struct{}{}
	<-gs.monitorDone

	gs.meshUpdateDone <- struct{}{}
	<-gs.meshUpdateDone

	gs.beacon.Close()

	gs.closeHeartbeats()

	close(gs.localConnUpdates)

	for _, s := range gs.streams {
		s.Close()
	}
}
