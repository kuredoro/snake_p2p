package game

import (
	"bufio"
	"math/rand"
	"sort"
	"strconv"
	"sync"

	"github.com/hashicorp/go-multierror"
	"github.com/kuredoro/snake_p2p/core"
	"github.com/libp2p/go-libp2p-core/network"
	"github.com/libp2p/go-libp2p-core/peer"
	"github.com/rs/zerolog/log"
)

type GameEstablished struct {
	Facilitator peer.ID
	Game        *GameInstance
}

type playerMove struct {
	ID  peer.ID
	Dir core.Direction
}

type GameInstanceEvent interface{}

type GameInstance struct {
	done       chan struct{}
	finishedCh chan struct{}
	streams    map[peer.ID]network.Stream
	selfID     peer.ID
	Seed       int64

	recv  chan interface{}
	moves chan playerMove

	mu sync.Mutex
}

func NewGameInstance() *GameInstance {
	return &GameInstance{
		done:       make(chan struct{}),
		finishedCh: make(chan struct{}),
		streams:    make(map[peer.ID]network.Stream),

		recv: make(chan interface{}),

		// FIXME: if SendEvent and we form the move, then we need the user to
		// receive the PlayerMoves event
		// but if it waits send event to finishi... it just deadlocks.
		moves: make(chan playerMove, 32),
	}
}

func (gi *GameInstance) PlayersIDs() []peer.ID {
	gi.mu.Lock()
	defer gi.mu.Unlock()
	var playerIDs []peer.ID
	for id := range gi.streams {
		playerIDs = append(playerIDs, id)
	}
	playerIDs = append(playerIDs, gi.selfID)
	sort.Slice(playerIDs, func(i, j int) bool {
		return playerIDs[i] < playerIDs[j]
	})
	return playerIDs
}

func (gi *GameInstance) SelfID() peer.ID {
	gi.mu.Lock()
	defer gi.mu.Unlock()
	return gi.selfID
}

func (gi *GameInstance) IncommingMoves() <-chan interface{} {
	return gi.recv
}

func (gi *GameInstance) AddPeer(s network.Stream) {
	p := s.Conn().RemotePeer()

	gi.mu.Lock()
	gi.streams[p] = s
	gi.mu.Unlock()
}

func (gi *GameInstance) RemovePeer(p peer.ID) {
	gi.mu.Lock()
	defer gi.mu.Unlock()

	s, exists := gi.streams[p]
	if !exists {
		return
	}

	err := s.Close()
	if err != nil {
		log.Err(err).Msg("Remove peer and close connection")
	}

	delete(gi.streams, p)
}

func (gi *GameInstance) Close() {
	gi.mu.Lock()
	defer gi.mu.Unlock()

	for p, s := range gi.streams {
		err := s.Close()
		if err != nil {
			log.Err(err).
				Str("peer", p.Pretty()).
				Msg("Close game stream")
		}
	}

	close(gi.done)
	<-gi.finishedCh
	for range gi.streams {
		<-gi.finishedCh
	}

	gi.done = make(chan struct{})
}

func (gi *GameInstance) PeerCount() int {
	gi.mu.Lock()
	n := len(gi.streams)
	gi.mu.Unlock()

	return n
}

func (gi *GameInstance) Run() int64 {
	gi.Seed = int64(gi.negotiateSeed())

	gi.mu.Lock()
	defer gi.mu.Unlock()

	for _, s := range gi.streams {
		if gi.selfID == "" {
			gi.selfID = s.Conn().LocalPeer()
		}

		go gi.readLoop(s)
	}

	go gi.syncLoop()

	return gi.Seed
}

func ToBytesUint32(n uint32) []byte {
	b := make([]byte, 4)
	for i := 0; n != 0 && i < 4; i++ {
		b[i] = byte(n & 0xFF)
		n >>= 8
	}

	return b
}

func ToUint32Bytes(b []byte) (n uint32) {
	for k := 0; k < 4; k++ {
		// TODO: precedance?
		n |= uint32(b[k]) << (8 * k)
	}

	return
}

func (gi *GameInstance) negotiateSeed() uint32 {
	gi.mu.Lock()
	defer gi.mu.Unlock()

	// TODO: TextMarshaller interface
	my := rand.Uint32()
	myBytes := ToBytesUint32(my)

	for peer, s := range gi.streams {
		_, err := s.Write(myBytes)
		if err != nil {
			log.Err(err).
				Str("peer", peer.Pretty()).
				Msg("Send our random piece")
		}
	}

	otherBytes := make([]byte, 4)
	for peer, s := range gi.streams {
		_, err := s.Read(otherBytes)
		if err != nil {
			log.Err(err).
				Str("peer", peer.Pretty()).
				Msg("Receive other random piece")
			continue
		}

		log.Debug().
			Str("peer", peer.Pretty()).
			Uint32("piece", ToUint32Bytes(otherBytes)).
			Msg("Receive other random piece")

		for i, other := range otherBytes {
			myBytes[i] ^= other
		}
	}

	seed := ToUint32Bytes(myBytes)
	log.Info().Uint32("seed", seed).Msg("Negotiated random seed")

	return seed
}

func (gi *GameInstance) SendMove(move core.Direction) (err error) {
	gi.mu.Lock()
	defer gi.mu.Unlock()

	// TODO: have a sane protocol, not just numbers flying around...
	msg := strconv.Itoa(int(move)) + "\n"

	for p, s := range gi.streams {
		_, err := s.Write([]byte(msg))
		if err != nil {
			err = multierror.Append(err, &core.PeerError{
				Peer: p,
				Err:  err,
			})
		}
	}

	gi.moves <- playerMove{
		ID:  gi.selfID,
		Dir: move,
	}

	return
}

func (gi *GameInstance) readLoop(stream network.Stream) {
	remotePeer := stream.Conn().RemotePeer()

	scanner := bufio.NewScanner(stream)
	readCh := make(chan bool)
	defer close(readCh)

	scan := func() {
		readCh <- scanner.Scan()
	}

	go scan()

	// XXX: refer to JoinService
	reading := true

	for {
		select {
		case <-gi.done:
			err := stream.Close()
			if err != nil {
				log.Err(err).
					Str("player", remotePeer.Pretty()).
					Msg("Close game stream due to Close()")
			}

			if reading {
				<-readCh // When stream has closed, .Scan() should quit
			}

			gi.finishedCh <- struct{}{}
			return
		case ok := <-readCh:
			if !ok {
				reading = false
				err := stream.Close()
				if err != nil {
					log.Err(err).
						Str("player", remotePeer.Pretty()).
						Msg("Close game stream")
				}

                gi.RemovePeer(remotePeer)

				// XXX: hax number 1000
				gi.moves <- playerMove{}

				log.Error().Msg("Dead")

				gi.recv <- remotePeer

				// Do not scan() again
				continue
			}

			n, err := strconv.Atoi(scanner.Text())
			if err != nil {
				log.Err(err).
					Str("player", remotePeer.Pretty()).
					Msg("Parse player move")

				go scan()
				continue
			}

			dir := core.Direction(n)

			gi.moves <- playerMove{
				ID:  remotePeer,
				Dir: dir,
			}

			go scan()
		}
	}
}

func (gi *GameInstance) syncLoop() {
	msg := core.PlayerMoves{
		Moves: make(map[peer.ID]core.Direction),
	}

	for peerMove := range gi.moves {
		if peerMove.ID != "" {
			log.Debug().Str("peer", peerMove.ID.Pretty()).Int("dir", int(peerMove.Dir)).Msg("Received move")
			msg.Moves[peerMove.ID] = peerMove.Dir
		} else {
			log.Debug().Msg("stub player move received to recheck peer count condition")
		}

		if len(msg.Moves) == gi.PeerCount()+1 {
			gi.recv <- msg

			msg = core.PlayerMoves{
				Moves: make(map[peer.ID]core.Direction),
			}
		}
	}
}
