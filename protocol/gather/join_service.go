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
	ctx    context.Context
	cancel func()

	stream network.Stream
}

func NewJoinService(ctx context.Context, h host.Host, pID peer.ID) (*JoinService, error) {
	stream, err := h.NewStream(ctx, pID, ID)
	if err != nil {
		return nil, fmt.Errorf("create gather protocol stream: %v", err)
	}

	localCtx, cancel := context.WithCancel(ctx)

	service := &JoinService{
		ctx:    localCtx,
		cancel: cancel,

		stream: stream,
	}

	stream.Write(nil)

	go service.Run()

	return service, nil
}

func (js *JoinService) Run() {
	scanner := bufio.NewScanner(js.stream)

	for scanner.Scan() {
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
	}
}
