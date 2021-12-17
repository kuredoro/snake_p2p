package gather

import (
	"context"
	"encoding/json"
	"fmt"
	"sort"
	"strconv"
	"strings"
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

type peerMesh map[peer.ID]map[peer.ID]struct{}

type peerMeshMod func(peerMesh) bool

func addEdge(from, to peer.ID) peerMeshMod {
    return func(mesh peerMesh) bool {
        if _, exists := mesh[from]; !exists {
            mesh[from] = make(map[peer.ID]struct{})
        }

        mesh[from][to] = struct{}{}
        return true
    }
}

func removeEdge(from, to peer.ID) peerMeshMod {
    return func(mesh peerMesh) bool {
        delete(mesh[from], to)
        return false
    }
}

func addDoubleEdge(from, to peer.ID) peerMeshMod {
    return func(mesh peerMesh) bool {
        addEdge(from, to)(mesh)
        addEdge(to, from)(mesh)
        return true
    }
}

func removeDoubleEdge(from, to peer.ID) peerMeshMod {
    return func(mesh peerMesh) bool {
        removeEdge(from, to)(mesh)
        removeEdge(to, from)(mesh)
        return false
    }
}

func (m peerMesh) String() string {
    var str strings.Builder

    index2peer := make([]peer.ID, 0, len(m))
    for id := range m {
        index2peer = append(index2peer, id)
    }

    sort.Slice(index2peer, func(i, j int) bool {
        return index2peer[i].String() < index2peer[j].String()
    })

    peer2index := make(map[peer.ID]int)
    for i, id := range index2peer {
        peer2index[id] = i
    }

    neightbours := make([]int, 0, len(m))
    for i, srcID := range index2peer {
        idStr := srcID.String()
        str.WriteString(strconv.Itoa(i))
        str.WriteRune(' ')
        str.WriteString(idStr[len(idStr)-6:])
        str.WriteString(": ")

        neightbours = neightbours[:0]
        for destID := range m[srcID] {
            neightbours = append(neightbours, peer2index[destID])
        }

        sort.Ints(neightbours)

        for _, index := range neightbours {
            str.WriteString(strconv.Itoa(index))
            str.WriteRune(' ')
        }
        str.WriteRune('\n')
    }

    return str.String()
}

type GatherService struct {
	ctx                      context.Context
	cancel                   func()
	publishDone, monitorDone chan struct{}

	h       host.Host
	streams map[peer.ID]network.Stream
	topic   *pubsub.Topic
	ttl     time.Duration

	mesh peerMesh

	ping             *ping.PingService
	conns            map[peer.ID]*heartbeat.HeartbeatService
	localConnUpdates chan heartbeat.PeerStatus
}

func NewGatherService(ctx context.Context, h host.Host, topic *pubsub.Topic, ping *ping.PingService, TTL time.Duration) (*GatherService, error) {
	localCtx, cancel := context.WithCancel(ctx)

	gs := &GatherService{
		ctx:         localCtx,
		cancel:      cancel,
		publishDone: make(chan struct{}),
		monitorDone: make(chan struct{}),

		h:       h,
		streams: make(map[peer.ID]network.Stream),
		topic:   topic,
		ttl:     TTL,

		mesh: make(peerMesh),

		ping:             ping,
		conns:            make(map[peer.ID]*heartbeat.HeartbeatService),
		localConnUpdates: make(chan heartbeat.PeerStatus),
	}

	h.SetStreamHandler(ID, gs.GatherHandler)

	go gs.publishLoop()
	go gs.monitorLoop()

	return gs, nil
}

func (gs *GatherService) GatherHandler(stream network.Stream) {
	fmt.Printf("PEER JOINED %v\n", stream.Conn().RemotePeer())

	hb, err := heartbeat.NewHeartbeat(gs.ctx, gs.ping, stream.Conn().RemotePeer(), gs.localConnUpdates)
	if err != nil {
		panic(err)
	}

	peer := stream.Conn().RemotePeer()
	gs.streams[peer] = stream
	gs.conns[peer] = hb
}

func (gs *GatherService) publishLoop() {
	timer := time.NewTimer(gs.ttl)

	for {
		select {
		case <-timer.C:
			err := gs.publish()
			if err != nil {
				fmt.Printf("gather service: announce: %v\n", err)
			}
			timer.Reset(gs.ttl)
		case <-gs.ctx.Done():
			close(gs.publishDone)
			return
		}
	}
}

func (gs *GatherService) publish() error {
	selfInfo := peer.AddrInfo{
		ID:    gs.h.ID(),
		Addrs: gs.h.Addrs(),
	}

	msg := GatherPointMessage{
		ConnectTo:          selfInfo,
		TTL:                time.Minute,
		DesiredPlayerCount: 3,
		CurrentPlayerCount: 0,
	}

	msgBytes, err := json.Marshal(msg)
	if err != nil {
		return fmt.Errorf("marshal gather point messasge: %v", err)
	}

	err = gs.topic.Publish(gs.ctx, msgBytes)
	if err != nil {
		return fmt.Errorf("publish gather point message: %v", err)
	}

	return nil
}

func (gs *GatherService) monitorLoop() {
	for {
		select {
		case <-gs.ctx.Done():
			close(gs.monitorDone)
			return
		case peerStatus := <-gs.localConnUpdates:
			switch peerStatus.Alive {
			case true:
				// TODO: rename ID to Peer
				if _, exists := gs.mesh[gs.h.ID()]; !exists {
					gs.mesh[gs.h.ID()] = make(map[peer.ID]struct{})
				}

				if _, exists := gs.mesh[peerStatus.ID]; !exists {
					gs.mesh[peerStatus.ID] = make(map[peer.ID]struct{})
				}

				gs.mesh[gs.h.ID()][peerStatus.ID] = struct{}{}
				gs.mesh[peerStatus.ID][gs.h.ID()] = struct{}{}
				fmt.Printf("PEER ALIVE %v\n", peerStatus.ID)

				err := gs.askEverybodyToConnectTo(peerStatus.ID)
				if err != nil {
					fmt.Printf("EVTC %v\n", err)
				}
			case false:
				delete(gs.mesh[gs.h.ID()], peerStatus.ID)
				delete(gs.mesh[peerStatus.ID], gs.h.ID())

				gs.streams[peerStatus.ID].Close()
				delete(gs.streams, peerStatus.ID)

				gs.conns[peerStatus.ID].Close()
				delete(gs.conns, peerStatus.ID)

				fmt.Printf("PEER DEAD %v\n", peerStatus.ID)
			}
		}
	}
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
		return fmt.Errorf("marshal: %v", err)
	}

	raw = append(raw, '\n')

	_, err = stream.Write(raw)
	if err != nil {
		return fmt.Errorf("send: %v", err)
	}

	return nil
}

func (gs *GatherService) Close() {
	var wg sync.WaitGroup
	wg.Add(len(gs.conns))

	for _, hb := range gs.conns {
		go func(hb *heartbeat.HeartbeatService) {
			hb.Close()
			wg.Done()
		}(hb)
	}

	wg.Wait()

	gs.cancel()
	<-gs.publishDone
	<-gs.monitorDone

	close(gs.localConnUpdates)

	for _, s := range gs.streams {
		s.Close()
	}
}
