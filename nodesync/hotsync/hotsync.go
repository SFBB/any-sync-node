//go:generate mockgen -destination mock_hotsync/mock_hotsync.go github.com/anytypeio/any-sync-node/nodesync/hotsync HotSync
package hotsync

import (
	"context"
	"github.com/anytypeio/any-sync-node/nodespace"
	"github.com/anytypeio/any-sync/app"
	"github.com/anytypeio/any-sync/app/logger"
	"github.com/anytypeio/any-sync/app/ocache"
	"github.com/anytypeio/any-sync/util/periodicsync"
	"go.uber.org/zap"
	"golang.org/x/exp/slices"
	"sync"
	"sync/atomic"
)

var log = logger.NewNamed(CName)

const (
	defaultSimRequests = 300
	CName              = "node.nodesync.hotsync"
)

type HotSync interface {
	app.ComponentRunnable
	UpdateQueue(changedIds []string)
	SetMetric(hit, miss *atomic.Uint32)
}

func New() HotSync {
	return new(hotSync)
}

type idProvider interface {
	Id() string
}

type hotSync struct {
	spaceQueue       []string
	syncQueue        map[string]struct{}
	simultaneousSync int
	hit              *atomic.Uint32
	miss             *atomic.Uint32

	spaceService nodespace.Service
	periodicSync periodicsync.PeriodicSync
	mx           sync.Mutex
}

func (h *hotSync) Init(a *app.App) (err error) {
	h.simultaneousSync = a.MustComponent("config").(configGetter).GetHotSync().SimultaneousRequests
	if h.simultaneousSync == 0 {
		h.simultaneousSync = defaultSimRequests
	}
	h.syncQueue = map[string]struct{}{}
	h.spaceService = a.MustComponent(nodespace.CName).(nodespace.Service)
	h.periodicSync = periodicsync.NewPeriodicSync(10, 0, h.checkCache, log)
	return
}

func (h *hotSync) Name() (name string) {
	return CName
}

func (h *hotSync) Run(ctx context.Context) (err error) {
	h.periodicSync.Run()
	return
}

func (h *hotSync) Close(ctx context.Context) (err error) {
	h.periodicSync.Close()
	return
}

func (h *hotSync) SetMetric(hit, miss *atomic.Uint32) {
	h.hit, h.miss = hit, miss
}

func (h *hotSync) UpdateQueue(changedIds []string) {
	h.mx.Lock()
	defer h.mx.Unlock()
	slices.Sort(changedIds)
	for _, id := range h.spaceQueue {
		if idx, exists := slices.BinarySearch(changedIds, id); exists {
			changedIds[idx] = ""
		}
	}
	for _, id := range changedIds {
		if id != "" {
			h.spaceQueue = append(h.spaceQueue, id)
		}
	}
}

func (h *hotSync) checkCache(ctx context.Context) (err error) {
	log.Debug("checking cache", zap.Int("space queue len", len(h.spaceQueue)), zap.Int("sync queue len", len(h.syncQueue)))
	removed := h.checkRemoved(ctx)
	log.Debug("removed inactive", zap.Int("removed", removed))
	h.mx.Lock()
	newBatchLen := h.batchLen()
	var cp []string
	cp = append(cp, h.spaceQueue[:newBatchLen]...)
	h.spaceQueue = h.spaceQueue[newBatchLen:]
	h.mx.Unlock()
	for _, id := range cp {
		_, err = h.spaceService.GetSpace(ctx, id)
		if err != nil {
			h.hit.Add(1)
			continue
		}
		h.miss.Add(1)
		h.syncQueue[id] = struct{}{}
	}
	return nil
}

func (h *hotSync) checkRemoved(ctx context.Context) (removed int) {
	cache := h.spaceService.Cache()
	allIds := map[string]struct{}{}
	cache.ForEach(func(v ocache.Object) (isContinue bool) {
		spc := v.(idProvider)
		allIds[spc.Id()] = struct{}{}
		return true
	})
	for id := range h.syncQueue {
		if _, exists := allIds[id]; !exists {
			removed++
			delete(h.syncQueue, id)
		}
	}
	return
}

func (h *hotSync) batchLen() int {
	newBatchLen := h.simultaneousSync - len(h.syncQueue)
	return min(newBatchLen, len(h.spaceQueue))
}

func min(x int, y int) int {
	if x < y {
		return x
	} else {
		return y
	}
}
