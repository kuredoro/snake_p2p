package gather

import (
	"context"
	"fmt"

	"github.com/libp2p/go-libp2p-core/host"
	"github.com/libp2p/go-libp2p-core/network"
	"github.com/libp2p/go-libp2p-core/peer"
)

type SyncService struct {
	ctx    context.Context
	cancel func()

	stream network.Stream
}

func NewSyncService(ctx context.Context, h host.Host, pID peer.ID) (*SyncService, error) {
	stream, err := h.NewStream(ctx, pID, ID)
	if err != nil {
		return nil, fmt.Errorf("create gather protocol stream: %v", err)
	}

	localCtx, cancel := context.WithCancel(ctx)

	service := &SyncService{
		ctx:    localCtx,
		cancel: cancel,

		stream: stream,
	}

	go service.Run()

	return service, nil
}

func (s *SyncService) Run() {
}
