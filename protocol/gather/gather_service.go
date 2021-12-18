package gather

import (
	"bufio"
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

	"github.com/kuredoro/snake_p2p/protocol/heartbeat"
)

type PeerError struct {
	Peer peer.ID
	Err  error
}

func (e *PeerError) Error() string {
	return fmt.Sprintf("peer %s: %v", e.Peer.Pretty(), e.Err)
}

func (e *PeerError) Unwrap() error {
	return e.Err
}

func HostAddrInfo(h host.Host) *peer.AddrInfo {
	return &peer.AddrInfo{
		ID:    h.ID(),
		Addrs: h.Addrs(),
	}
}

type GatherService struct {
	monitorDone, meshUpdateDone chan struct{}

	h       host.Host
	streams map[peer.ID]network.Stream
	topic   *pubsub.Topic
	ttl     time.Duration

	mesh   peerMesh
	meshCh chan peerMeshMod

	ping             *ping.PingService
	conns            map[peer.ID]*heartbeat.HeartbeatService
	localConnUpdates chan heartbeat.PeerStatus

	beacon *GatherPointBeacon
}

func NewGatherService(h host.Host, topic *pubsub.Topic, ping *ping.PingService, TTL time.Duration) (*GatherService, error) {
	gs := &GatherService{
		monitorDone:    make(chan struct{}),
		meshUpdateDone: make(chan struct{}),

		h:       h,
		streams: make(map[peer.ID]network.Stream),
		topic:   topic,
		ttl:     TTL,

		mesh:   make(peerMesh),
		meshCh: make(chan peerMeshMod),

		ping:             ping,
		conns:            make(map[peer.ID]*heartbeat.HeartbeatService),
		localConnUpdates: make(chan heartbeat.PeerStatus),

		beacon: NewGatherPointBeacon(topic, *HostAddrInfo(h), TTL),
	}

	h.SetStreamHandler(ID, gs.GatherHandler)

	go gs.monitorLoop()
	go gs.meshUpdateLoop()

	return gs, nil
}

func (gs *GatherService) GatherHandler(stream network.Stream) {
	fmt.Printf("PEER JOINED %v\n", stream.Conn().RemotePeer())

	hb, err := heartbeat.NewHeartbeat(gs.ping, stream.Conn().RemotePeer(), gs.localConnUpdates)
	if err != nil {
		panic(err)
	}

	peer := stream.Conn().RemotePeer()
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
	for {
		select {
		case ok := <-readCh:
			if !ok {
				fmt.Printf("Not ok\n")
				return
			}
			fmt.Printf("READ\n")

			var msg GatherMessage
			err := json.Unmarshal(scanner.Bytes(), &msg)
			if err != nil {
				fmt.Printf("ERR FROM %s: %v\n", remotePeer, err)
				go scan()
				continue
			}

			switch msg.Type {
			case Connected:
				if len(msg.Addrs) == 0 {
					fmt.Printf("WAT empty msg addrs from %s for CONN\n", remotePeer)
					break
				}
				fmt.Printf("CONN %s <-> %s\n", remotePeer, msg.Addrs[0].ID)
				gs.meshCh <- addDoubleEdge(remotePeer, msg.Addrs[0].ID)
			case Disconnected:
				if len(msg.Addrs) == 0 {
					fmt.Printf("WAT empty msg addrs from %s for DISC\n", remotePeer)
					break
				}
				fmt.Printf("DISC %s <-> %s\n", remotePeer, msg.Addrs[0].ID)
				gs.meshCh <- removeDoubleEdge(remotePeer, msg.Addrs[0].ID)
			default:
				fmt.Printf("WAT\n")
			}

			go scan()
		}
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

			fmt.Printf("mesh update:\n%v\n", gs.mesh)

			if !rescan {
				continue
			}

			clique := gs.mesh.FindClique(4, gs.h.ID())
			if clique == nil {
				fmt.Printf("NO CLIQUES FOUND\n")
				continue
			}

			fmt.Printf("CLIQUE FOUND %v\n", clique)
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
				// TODO: rename ID to Peer
				gs.meshCh <- addDoubleEdge(gs.h.ID(), peerStatus.Peer)
				fmt.Printf("PEER ALIVE %v\n", peerStatus.Peer)

				err := gs.askEverybodyToConnectTo(peerStatus.Peer)
				if err != nil {
					merr := err.(*multierror.Error)
					for _, peerErr := range merr.Errors {
						gs.peerDisconnected(peerErr.(*PeerError).Peer)
						fmt.Printf("ASK ERR %v\n", peerErr)
					}
				}
			case false:
				gs.meshCh <- removeDoubleEdge(gs.h.ID(), peerStatus.Peer)

				gs.peerDisconnected(peerStatus.Peer)
				/*
				   The main stream may still be alive

				*/

				fmt.Printf("PEER DEAD %v\n", peerStatus.Peer)
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
		fmt.Printf("Asking %v\n", srcID)

		stream, ok := gs.streams[srcID]
		if !ok {
			fmt.Printf("WAT peer is present in the mesh, but has no stream... ID %v\n", srcID)
			continue
		}

		go func() {
			errCh <- gs.askPeerToConnectTo(stream, gs.h.Peerstore().PeerInfo(peer))
		}()

		msgCount++
	}

	// Sanity check
	if msgCount != len(gs.mesh)-2 {
		fmt.Printf("WAT going to notify %d peers, but expected %d\n", msgCount, len(gs.streams)-1)
	}

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
		return &PeerError{
			Peer: pi.ID,
			Err:  fmt.Errorf("marshal: %v", err),
		}
	}

	raw = append(raw, '\n')

	_, err = stream.Write(raw)
	if err != nil {
		return &PeerError{
			Peer: pi.ID,
			Err:  fmt.Errorf("send: %v", err),
		}
	}

	return nil
}

func (gs *GatherService) Close() {
	gs.monitorDone <- struct{}{}
	<-gs.monitorDone

	gs.meshUpdateDone <- struct{}{}
	<-gs.meshUpdateDone

	gs.beacon.Close()

	var wg sync.WaitGroup
	wg.Add(len(gs.conns))

	for _, hb := range gs.conns {
		go func(hb *heartbeat.HeartbeatService) {
			hb.Close()
			wg.Done()
		}(hb)
	}

	wg.Wait()

	close(gs.localConnUpdates)

	for _, s := range gs.streams {
		s.Close()
	}
}
