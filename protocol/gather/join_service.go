package gather

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"

	"github.com/kuredoro/snake_p2p/protocol/heartbeat"
	"github.com/libp2p/go-libp2p-core/host"
	"github.com/libp2p/go-libp2p-core/network"
	"github.com/libp2p/go-libp2p-core/peer"
	"github.com/libp2p/go-libp2p/p2p/protocol/ping"
)

type JoinService struct {
	done chan struct{}

	h            host.Host
	ping         *ping.PingService
	stream       network.Stream
	conns        map[peer.ID]*heartbeat.HeartbeatService
	connHealthCh chan heartbeat.PeerStatus
}

func NewJoinService(ctx context.Context, h host.Host, ping *ping.PingService, pID peer.ID) (*JoinService, error) {
	stream, err := h.NewStream(ctx, pID, ID)
	if err != nil {
		return nil, fmt.Errorf("create gather protocol stream: %v", err)
	}

	service := &JoinService{
		done: make(chan struct{}),

		h:            h,
		ping:         ping,
		stream:       stream,
		conns:        make(map[peer.ID]*heartbeat.HeartbeatService),
		connHealthCh: make(chan heartbeat.PeerStatus),
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

	errCh := make(chan error)
	defer close(errCh)

	for {
		select {
		case <-js.done:
			err := js.stream.Close()
			if err != nil {
				fmt.Printf("ERR join service: close: %v\n", err)
			}

			<-readCh // When stream has closed, .Scan() should quit
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
				go func() {
					err := js.sendConnected(status.Peer)
					if err != nil {
						errCh <- fmt.Errorf("send connected: %v", err)
					}
				}()
			case false:
				go func() {
					err := js.sendDisconnected(status.Peer)
					if err != nil {
						errCh <- fmt.Errorf("send disconnected: %v", err)
					}
				}()
			}
		case ok := <-readCh:
			if !ok {
				err := js.stream.Close()
				if err != nil {
					fmt.Printf("ERR join service: close: %v\n", err)
				}

				// Do not scan() again
				continue
			}

			var msg GatherMessage
			err := json.Unmarshal(scanner.Bytes(), &msg)
			if err != nil {
				fmt.Printf("BAD MSG %q\n", scanner.Text())
				continue
			}

			switch msg.Type {
			case ConnectionRequest:
				fmt.Printf("CONN REQUEST to %v\n", msg.Addrs[0].ID)
				go func() {
					if len(msg.Addrs) == 0 {
						errCh <- fmt.Errorf("connection request does not list peers")
						return
					}
					// TODO: handle concurrent map access
					err := js.connect(msg.Addrs[0])
					if err != nil {
						errCh <- fmt.Errorf("connect to %v: %v", msg.Addrs[0].ID, err)
					}
				}()
			default:
				fmt.Printf("BAD TYPE %#v", msg)
			}

			go scan()
		case err := <-errCh:
			fmt.Printf("ERR JOIN %v\n", err)
		}
	}
}

func (js *JoinService) Close() {
	js.done <- struct{}{}
	<-js.done
}

func (js *JoinService) sendConnected(p peer.ID) error {
	fmt.Printf("CONNECTED %v\n", p)

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
	fmt.Printf("DISCONNECTED %v\n", p)

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
