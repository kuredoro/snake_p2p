package gather

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"

	"github.com/libp2p/go-libp2p-core/host"
	"github.com/libp2p/go-libp2p-core/network"
	"github.com/libp2p/go-libp2p-core/peer"
)

type JoinService struct {
	done chan struct{}

	stream network.Stream
}

func NewJoinService(ctx context.Context, h host.Host, pID peer.ID) (*JoinService, error) {
	stream, err := h.NewStream(ctx, pID, ID)
	if err != nil {
		return nil, fmt.Errorf("create gather protocol stream: %v", err)
	}

	service := &JoinService{
		done: make(chan struct{}),

		stream: stream,
	}

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

	scan()

	for {
		select {
		case <-js.done:
			err := js.stream.Close()
			if err != nil {
				fmt.Printf("ERR join service: close: %v\n", err)
			}
			close(js.done)
			return
		case ok := <-readCh:
			if !ok {
				err := js.stream.Close()
				if err != nil {
					fmt.Printf("ERR join service: close: %v\n", err)
				}
				close(js.done)
				return
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
			default:
				fmt.Printf("BAD TYPE %#v", msg)
			}

			scan()
		}
	}
}

func (js *JoinService) Close() {
	js.done <- struct{}{}
	<-js.done
}
