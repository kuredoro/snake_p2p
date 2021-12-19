package game

import (
	"bufio"
	"fmt"
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

type GameInstance struct {
	done    chan struct{}
	streams map[peer.ID]network.Stream

	recv chan core.PlayerMove

	mu sync.Mutex
}

func NewGameInstance() *GameInstance {
	return &GameInstance{
		done:    make(chan struct{}),
		streams: make(map[peer.ID]network.Stream),

		recv: make(chan core.PlayerMove),
	}
}

func (gi *GameInstance) IncommingMoves() <-chan core.PlayerMove {
	return gi.recv
}

func (gi *GameInstance) AddPeer(s network.Stream) {
	p := s.Conn().RemotePeer()

	gi.mu.Lock()
	gi.streams[p] = s
	gi.mu.Unlock()
}

func (gi *GameInstance) RemovePeer(p peer.ID) error {
	gi.mu.Lock()
	defer gi.mu.Unlock()

	s, exists := gi.streams[p]
	if !exists {
		return nil
	}

	err := s.Close()
	if err != nil {
		return fmt.Errorf("close game stream: %v", err)
	}

	delete(gi.streams, p)

	return nil
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

	gi.done <- struct{}{}
	<-gi.done
}

func (gi *GameInstance) PeerCount() int {
	gi.mu.Lock()
	n := len(gi.streams)
	gi.mu.Unlock()

	return n
}

func (gi *GameInstance) Run() {
	gi.mu.Lock()
	defer gi.mu.Unlock()

	for _, s := range gi.streams {
		go gi.readLoop(s)
	}
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

			close(gi.done)
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

				log.Error().Msg("Dead")

				// Do not scan() again
				continue
			}

			n, err := strconv.Atoi(scanner.Text())
			if err != nil {
				log.Err(err).
					Str("player", remotePeer.Pretty()).
					Msg("Parse player move")
				continue
			}

			dir := core.Direction(n)

			gi.recv <- core.PlayerMove{
				Moves: map[int]core.Direction{
					0: dir,
				},
			}
		}
	}
}
