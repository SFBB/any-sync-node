package peermanager

import (
	"context"
	"sync"
	"time"

	"github.com/anyproto/any-sync/app"
	"github.com/anyproto/any-sync/app/logger"
	"github.com/anyproto/any-sync/commonspace/peermanager"
	"github.com/anyproto/any-sync/net"
	"github.com/anyproto/any-sync/net/peer"
	"github.com/anyproto/any-sync/net/pool"
	"github.com/anyproto/any-sync/net/streampool"
	"go.uber.org/atomic"
	"go.uber.org/zap"
	"storj.io/drpc"
)

const reconnectTimeout = time.Minute

type responsiblePeer struct {
	peerId   string
	lastFail atomic.Time
}

type nodePeerManager struct {
	spaceId                 string
	responsiblePeers        []responsiblePeer
	responsiblePeersUpdated atomic.Time
	responsiblePeersMu      sync.Mutex
	p                       *provider
	streamPool              streampool.StreamPool
}

func (n *nodePeerManager) Init(a *app.App) (err error) {
	n.streamPool = a.MustComponent(streampool.CName).(streampool.StreamPool)
	return nil
}

func (n *nodePeerManager) Name() (name string) {
	return peermanager.CName
}

func (n *nodePeerManager) SendResponsible(ctx context.Context, msg drpc.Message, streamPool streampool.StreamPool) (err error) {
	ctx = logger.CtxWithFields(context.Background(), logger.CtxGetFields(ctx)...)
	return streamPool.Send(ctx, msg, func(ctx context.Context) (peers []peer.Peer, err error) {
		return n.getResponsiblePeers(ctx, n.p.pool)
	})
}

func (n *nodePeerManager) SendMessage(ctx context.Context, peerId string, msg drpc.Message) error {
	ctx = logger.CtxWithFields(context.Background(), logger.CtxGetFields(ctx)...)
	if n.isResponsible(peerId) {
		return n.streamPool.Send(ctx, msg, func(ctx context.Context) ([]peer.Peer, error) {
			log.InfoCtx(ctx, "sendPeer send", zap.String("peerId", peerId))
			p, e := n.p.pool.Get(ctx, peerId)
			if e != nil {
				return nil, e
			}
			return []peer.Peer{p}, nil
		})
	}
	log.InfoCtx(ctx, "sendPeer sendById", zap.String("peerId", peerId))
	return n.streamPool.SendById(ctx, msg, peerId)
}

func (n *nodePeerManager) BroadcastMessage(ctx context.Context, msg drpc.Message) (err error) {
	ctx = logger.CtxWithFields(context.Background(), logger.CtxGetFields(ctx)...)
	if e := n.SendResponsible(ctx, msg, n.streamPool); e != nil {
		log.InfoCtx(ctx, "broadcast sendResponsible error", zap.Error(e))
	}
	log.InfoCtx(ctx, "broadcast", zap.String("spaceId", n.spaceId))
	return n.streamPool.Broadcast(ctx, msg, n.spaceId)
}

func (n *nodePeerManager) GetResponsiblePeers(ctx context.Context) (peers []peer.Peer, err error) {
	return n.getResponsiblePeers(ctx, n.p.pool)
}

func (n *nodePeerManager) GetNodePeers(ctx context.Context) (peers []peer.Peer, err error) {
	return n.GetResponsiblePeers(ctx)
}

func (n *nodePeerManager) KeepAlive(ctx context.Context) {}

func (n *nodePeerManager) getResponsiblePeers(ctx context.Context, netPool pool.Pool) (peers []peer.Peer, err error) {
	for _, rp := range n.getResponsiblePeersObjects() {
		if time.Since(rp.lastFail.Load()) > reconnectTimeout {
			p, e := netPool.Get(ctx, rp.peerId)
			if e != nil {
				log.InfoCtx(ctx, "can't connect to peer", zap.Error(err), zap.String("peerId", rp.peerId))
				rp.lastFail.Store(time.Now())
				continue
			}
			peers = append(peers, p)
		}
	}
	if len(peers) == 0 {
		return nil, net.ErrUnableToConnect
	}
	return
}

func (n *nodePeerManager) isResponsible(peerId string) bool {
	for _, rp := range n.getResponsiblePeersObjects() {
		if rp.peerId == peerId {
			return true
		}
	}
	return false
}

func (n *nodePeerManager) getResponsiblePeersObjects() []responsiblePeer {
	if len(n.responsiblePeers) != 0 && time.Since(n.responsiblePeersUpdated.Load()) < time.Minute {
		return n.responsiblePeers
	}

	n.responsiblePeersMu.Lock()
	defer n.responsiblePeersMu.Unlock()
	nodeIds := n.p.nodeconf.NodeIds(n.spaceId)
	for _, peerId := range nodeIds {
		n.responsiblePeers = append(n.responsiblePeers, responsiblePeer{peerId: peerId})
	}
	n.responsiblePeersUpdated.Store(time.Now())
	return n.responsiblePeers
}
