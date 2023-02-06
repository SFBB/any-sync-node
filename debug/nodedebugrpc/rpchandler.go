package nodedebugrpc

import (
	"context"
	"github.com/anytypeio/any-sync-node/debug/nodedebugrpc/nodedebugrpcproto"
	"time"
)

type rpcHandler struct {
	s *nodeDebugRpc
}

func (r *rpcHandler) DumpTree(ctx context.Context, request *nodedebugrpcproto.DumpTreeRequest) (resp *nodedebugrpcproto.DumpTreeResponse, err error) {
	tree, err := r.s.treeCache.GetTree(context.Background(), request.SpaceId, request.DocumentId)
	if err != nil {
		return
	}
	// TODO: commented
	_ = tree
	/*
		dump, err := tree.DebugDump(nil)
		if err != nil {
			return
		}*/
	resp = &nodedebugrpcproto.DumpTreeResponse{
		//Dump: dump,
	}
	return
}

func (r *rpcHandler) AllTrees(ctx context.Context, request *nodedebugrpcproto.AllTreesRequest) (resp *nodedebugrpcproto.AllTreesResponse, err error) {
	space, err := r.s.spaceService.GetSpace(ctx, request.SpaceId)
	if err != nil {
		return
	}
	heads := space.DebugAllHeads()
	var trees []*nodedebugrpcproto.Tree
	for _, head := range heads {
		trees = append(trees, &nodedebugrpcproto.Tree{
			Id:    head.Id,
			Heads: head.Heads,
		})
	}
	resp = &nodedebugrpcproto.AllTreesResponse{Trees: trees}
	return
}

func (r *rpcHandler) AllSpaces(ctx context.Context, request *nodedebugrpcproto.AllSpacesRequest) (resp *nodedebugrpcproto.AllSpacesResponse, err error) {
	ids, err := r.s.storageService.AllSpaceIds()
	if err != nil {
		return
	}
	resp = &nodedebugrpcproto.AllSpacesResponse{SpaceIds: ids}
	return
}

func (r *rpcHandler) TreeParams(ctx context.Context, request *nodedebugrpcproto.TreeParamsRequest) (resp *nodedebugrpcproto.TreeParamsResponse, err error) {
	tree, err := r.s.treeCache.GetTree(context.Background(), request.SpaceId, request.DocumentId)
	if err != nil {
		return
	}
	resp = &nodedebugrpcproto.TreeParamsResponse{
		RootId:  tree.Root().Id,
		HeadIds: tree.Heads(),
	}
	return
}

func (r *rpcHandler) ForceNodeSync(ctx context.Context, request *nodedebugrpcproto.ForceNodeSyncRequest) (*nodedebugrpcproto.ForceNodeSyncResponse, error) {
	var errCh = make(chan error, 1)
	go func() {
		errCh <- r.s.nodeSync.Sync()
	}()
	select {
	case <-time.After(time.Millisecond * 100):
		return &nodedebugrpcproto.ForceNodeSyncResponse{}, nil
	case err := <-errCh:
		return nil, err
	}

}
