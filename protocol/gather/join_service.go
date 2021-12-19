package gather

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"

	"github.com/kuredoro/snake_p2p/core"
	"github.com/kuredoro/snake_p2p/protocol/game"
	"github.com/kuredoro/snake_p2p/protocol/heartbeat"
	"github.com/libp2p/go-libp2p-core/host"
	"github.com/libp2p/go-libp2p-core/network"
	"github.com/libp2p/go-libp2p-core/peer"
	"github.com/libp2p/go-libp2p/p2p/protocol/ping"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
)

type JoinService struct {
	done chan struct{}

	h            host.Host
	ping         *ping.PingService
	game         *game.GameService
	stream       network.Stream
	conns        map[peer.ID]*heartbeat.HeartbeatService
	connHealthCh chan heartbeat.PeerStatus

	log zerolog.Logger

	gameCh chan<- core.GameEstablished
}

func NewJoinService(ctx context.Context, h host.Host, game *game.GameService, ping *ping.PingService, pID peer.ID, gameCh chan<- core.GameEstablished) (*JoinService, error) {
	stream, err := h.NewStream(ctx, pID, ID)
	if err != nil {
		return nil, fmt.Errorf("create gather protocol stream: %v", err)
	}

	logger := log.Logger.With().Str("facilitator", pID.Pretty()).Logger()

	service := &JoinService{
		done: make(chan struct{}),

		h:            h,
		ping:         ping,
		game:         game,
		stream:       stream,
		conns:        make(map[peer.ID]*heartbeat.HeartbeatService),
		connHealthCh: make(chan heartbeat.PeerStatus),

		log: logger,

		gameCh: gameCh,
	}

	// Makes sure the facilitator's callback is called.
	stream.Write(nil)

	go service.run()

	return service, nil
}

func (js *JoinService) run() {
	scanner := bufio.NewScanner(js.stream)
	readCh := make(chan bool)
	defer close(readCh)

	scan := func() {
		readCh <- scanner.Scan()
	}

	go scan()

	// XXX: I'm so hungry...
	// What is the proper way to handle this interdependency between
	// reading and exiting. So much to learn....
	reading := true

	for {
		select {
		case <-js.done:
			err := js.stream.Close()
			if err != nil {
				fmt.Printf("ERR join service: close: %v\n", err)
			}

			js.closeHeartbeats()

			if reading {
				<-readCh // When stream has closed, .Scan() should quit
			}
			close(js.done)
			return
		case status := <-js.connHealthCh:
			switch status.Alive {
			case true:
				// TODO: research best practices for handling sending and
				// receiving messages concurrently. How API should look?
				// If sendXXX has a return type of error, then it is
				// impossible to forget to send an error, but then we
				// have to write a boilerplate wrapper around the funcs.
				// What will befnefit us in a long run, I wonder...
				// UPD: I've deleted errCh, why do we even use it?
				// And in GatherService we use _no_ goroutines...
				// UPD2: I've removed goroutines...
				err := js.game.Connect(context.Background(), status.Peer)
				if err != nil {
					js.log.Err(err).
						Str("seeker", status.Peer.Pretty()).
						Msg("New game connection with peer seeker")
					continue
				}

				js.log.Info().
					Str("seeker", status.Peer.Pretty()).
					Msg("New game connection with peer seeker")

				err = js.sendConnected(status.Peer)
				if err != nil {
					js.log.Err(err).
						Str("seeker", status.Peer.Pretty()).
						Msg("Notify about new seeker-seeker connection")
				}
			case false:
				js.log.Info().
					Str("seeker", status.Peer.Pretty()).
					Msg("Peer seeker game connection reset")

				js.game.Disconnect(status.Peer)

				err := js.sendDisconnected(status.Peer)
				if err != nil {
					js.log.Err(err).
						Str("seeker", status.Peer.Pretty()).
						Msg("Notify about seeker-seeker connection reset")
				}
			}
		case ok := <-readCh:
			if !ok {
				reading = false
				err := js.stream.Close()
				if err != nil {
					js.log.Err(err).Msg("Close stream")
				}

				// Do not scan() again
				continue
			}

			var msg GatherMessage
			err := json.Unmarshal(scanner.Bytes(), &msg)
			if err != nil {
				js.log.Err(err).
					Str("text", fmt.Sprintf("%q", scanner.Text())).
					Msg("Received junk from facilitator")
				go scan()
				continue
			}

			switch msg.Type {
			case ConnectionRequest:
				go func() {
					if len(msg.Addrs) == 0 {
						js.log.Warn().
							Msg("Received connection request does not list peers")
						return
					}

					js.log.Info().
						Str("to", msg.Addrs[0].ID.Pretty()).
						Msg("Seeker-seeker connection request")

					// TODO: handle concurrent map access
					err := js.connect(msg.Addrs[0])
					if err != nil {
						js.log.Err(err).
							Str("to", msg.Addrs[0].ID.Pretty()).
							Msg("Connect to peer seeker")
					}
				}()
			case GatheringFinished:
				err := js.stream.Close()
				if err != nil {
					js.log.Err(err).
						Msg("Reset stream")
				}

				foundMyself := false
				for _, pi := range msg.Addrs {
					if pi.ID == js.h.ID() {
						foundMyself = true
						break
					}
				}

				js.closeHeartbeats()

				// God, this (reading flag) is so... error prone...
				reading = false

				if !foundMyself {
					continue
				}

				js.log.Info().
					Msg("Chosen for a game")

				js.gameCh <- core.GameEstablished{
					Facilitator: js.stream.Conn().RemotePeer(),
					Game:        js.game.GetInstance(),
				}
				continue
			default:
				js.log.Warn().
					Int("type", int(msg.Type)).
					Msg("Received message of unknown type")
			}

			go scan()
		}
	}
}

func (js *JoinService) closeHeartbeats() {
	for id, hb := range js.conns {
		hb.Close()
		delete(js.conns, id)
	}
}

func (js *JoinService) Close() {
	js.done <- struct{}{}
	<-js.done
}

func (js *JoinService) sendConnected(p peer.ID) error {
	log.Info().
		Str("to", p.Pretty()).
		Str("facilitator", js.stream.Conn().RemotePeer().Pretty()).
		Msg("Send seeker connected message")

	msg := GatherMessage{
		Type:  Connected,
		Addrs: []peer.AddrInfo{{ID: p}},
	}

	raw, err := json.Marshal(&msg)
	if err != nil {
		return fmt.Errorf("marshal: %v", err)
	}

	raw = append(raw, '\n')

	_, err = js.stream.Write(raw)
	if err != nil {
		return fmt.Errorf("write: %v", err)
	}

	return nil
}

func (js *JoinService) sendDisconnected(p peer.ID) error {
	log.Info().
		Str("from", p.Pretty()).
		Str("facilitator", js.stream.Conn().RemotePeer().Pretty()).
		Msg("Send seeker disconnected message")

	msg := GatherMessage{
		Type:  Disconnected,
		Addrs: []peer.AddrInfo{{ID: p}},
	}

	raw, err := json.Marshal(&msg)
	if err != nil {
		return fmt.Errorf("marshal: %v", err)
	}

	raw = append(raw, '\n')

	_, err = js.stream.Write(raw)
	if err != nil {
		return fmt.Errorf("write: %v", err)
	}

	return nil
}

func (js *JoinService) connect(pi peer.AddrInfo) error {
	if _, exists := js.conns[pi.ID]; exists {
		return nil
	}

	err := js.h.Connect(context.Background(), pi)
	if err != nil {
		return fmt.Errorf("raw connect: %v", err)
	}

	hb, err := heartbeat.NewHeartbeat(js.ping, pi.ID, js.connHealthCh)
	if err != nil {
		return fmt.Errorf("create heartbeat: %v", err)
	}

	js.conns[pi.ID] = hb
	return nil
}
