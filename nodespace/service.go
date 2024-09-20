//go:generate mockgen -destination mock_nodespace/mock_nodespace.go github.com/anyproto/any-sync-node/nodespace Service,NodeSpace
package nodespace

import (
	"context"
	"time"

	"github.com/anyproto/any-sync/app"
	"github.com/anyproto/any-sync/app/logger"
	"github.com/anyproto/any-sync/app/ocache"
	"github.com/anyproto/any-sync/commonspace"
	"github.com/anyproto/any-sync/commonspace/config"
	"github.com/anyproto/any-sync/commonspace/spacestorage"
	"github.com/anyproto/any-sync/commonspace/spacesyncproto"
	"github.com/anyproto/any-sync/commonspace/syncstatus"
	"github.com/anyproto/any-sync/consensus/consensusclient"
	"github.com/anyproto/any-sync/coordinator/coordinatorclient"
	"github.com/anyproto/any-sync/metric"
	"github.com/anyproto/any-sync/net/rpc/server"
	"github.com/anyproto/any-sync/net/streampool"
	"github.com/anyproto/any-sync/nodeconf"
	"go.uber.org/zap"

	"errors"
	"github.com/anyproto/any-sync-node/nodehead"
	"github.com/anyproto/any-sync-node/nodespace/treesyncer"
	"github.com/anyproto/any-sync-node/nodestorage"
)

const CName = "node.nodespace"

var log = logger.NewNamed(CName)

func New() Service {
	return &service{}
}

type Service interface {
	GetSpace(ctx context.Context, id string) (NodeSpace, error)
	PickSpace(ctx context.Context, id string) (NodeSpace, error)
	Cache() ocache.OCache
	GetStats(ctx context.Context, id string) (SpaceStats, error)
	app.ComponentRunnable
}

type service struct {
	conf                 config.Config
	spaceCache           ocache.OCache
	commonSpace          commonspace.SpaceService
	confService          nodeconf.Service
	consClient           consensusclient.Service
	spaceStorageProvider nodestorage.NodeStorage
	streamPool           streampool.StreamPool
	nodeHead             nodehead.NodeHead
	metric               metric.Metric
	coordClient          coordinatorclient.CoordinatorClient
}

func (s *service) Init(a *app.App) (err error) {
	s.conf = a.MustComponent("config").(config.ConfigGetter).GetSpace()
	s.commonSpace = a.MustComponent(commonspace.CName).(commonspace.SpaceService)
	s.confService = a.MustComponent(nodeconf.CName).(nodeconf.Service)
	s.spaceStorageProvider = a.MustComponent(spacestorage.CName).(nodestorage.NodeStorage)
	s.nodeHead = a.MustComponent(nodehead.CName).(nodehead.NodeHead)
	s.consClient = a.MustComponent(consensusclient.CName).(consensusclient.Service)
	s.streamPool = a.MustComponent(streampool.CName).(streampool.StreamPool)
	s.spaceCache = ocache.New(
		s.loadSpace,
		ocache.WithLogger(log.Sugar()),
		ocache.WithGCPeriod(time.Minute),
		ocache.WithTTL(time.Duration(s.conf.GCTTL)*time.Second),
		ocache.WithPrometheus(a.MustComponent(metric.CName).(metric.Metric).Registry(), "space", "cache"),
	)
	s.metric = a.MustComponent(metric.CName).(metric.Metric)
	s.coordClient = app.MustComponent[coordinatorclient.CoordinatorClient](a)
	return spacesyncproto.DRPCRegisterSpaceSync(a.MustComponent(server.CName).(server.DRPCServer), &rpcHandler{s})
}

func (s *service) Name() (name string) {
	return CName
}

func (s *service) Run(ctx context.Context) (err error) {
	return
}

func (s *service) PickSpace(ctx context.Context, id string) (NodeSpace, error) {
	v, err := s.spaceCache.Pick(ctx, id)
	if err != nil {
		return nil, err
	}
	return v.(NodeSpace), nil
}

type ChangeSizeStats struct {
	MaxLen int
	P95    int
	Avg    float64
	Median float64
}

type SpaceStats struct {
	DocsCount  int
	ChangeSize ChangeSizeStats
}

type spaceStorageStats interface {
	GetMaxChangeLen() (int, error)
}

var (
	ErrDoesntSupportStats   = errors.New("SpaceStorage doesn't support spaceStorageStats")
	ErrSpaceStorageIsLocked = errors.New("SpaceStorage is locked, try again later")
)

// TODO: handle "space is missing" when space id is wrong
func (s *service) GetStats(ctx context.Context, id string) (spaceStats SpaceStats, err error) {
	// TODO: this takes 30 seconds
	// when coordinator is not connected, it waits.
	space, getSpaceErr := s.GetSpace(ctx, id)
	defer func() {
		if getSpaceErr == nil {
			if closeErr := space.Close(); closeErr != nil {
				err = errors.Join(err, closeErr)
			}
		}
	}()

	if getSpaceErr != nil {
		err = getSpaceErr
		return
	}

	// TODO: cast doesn't work for some reason
	storage, ok := space.Storage().(spaceStorageStats)
	if ok {
		var maxLen int
		maxLen, err = storage.GetMaxChangeLen()
		if err != nil {
			return
		}

		changeSize := ChangeSizeStats{
			MaxLen: maxLen,
		}

		spaceStats = SpaceStats{
			ChangeSize: changeSize,
		}

		return
	} else {
		err = ErrDoesntSupportStats
		return
	}
}

func (s *service) GetSpace(ctx context.Context, id string) (NodeSpace, error) {
	v, err := s.spaceCache.Get(ctx, id)
	if err != nil {
		return nil, err
	}
	space := v.(NodeSpace)
	return space, nil
}

func (s *service) loadSpace(ctx context.Context, id string) (value ocache.Object, err error) {
	defer func() {
		log.InfoCtx(ctx, "space loaded", zap.String("id", id), zap.Error(err))
	}()
	if err = s.checkDeletionStatus(id); err != nil {
		return nil, err
	}
	cc, err := s.commonSpace.NewSpace(ctx, id, commonspace.Deps{
		TreeSyncer: treesyncer.New(id),
		SyncStatus: syncstatus.NewNoOpSyncStatus(),
	})
	if err != nil {
		return
	}
	ns, err := newNodeSpace(cc, s.consClient, s.spaceStorageProvider)
	if err != nil {
		return
	}
	if err = ns.Init(ctx); err != nil {
		return
	}
	return ns, nil
}

func (s *service) checkDeletionStatus(spaceId string) (err error) {
	delStorage := s.spaceStorageProvider.DeletionStorage()
	status, err := delStorage.SpaceStatus(spaceId)
	if err != nil {
		if err == nodestorage.ErrUnknownSpaceId {
			return nil
		}
		return err
	}
	if status == nodestorage.SpaceStatusRemove {
		return spacesyncproto.ErrSpaceIsDeleted
	}
	return nil
}

func (s *service) Close(ctx context.Context) (err error) {
	return s.spaceCache.Close()
}

func (s *service) Cache() ocache.OCache {
	return s.spaceCache
}
