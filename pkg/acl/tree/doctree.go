package tree

import (
	"context"
	"errors"
	"github.com/anytypeio/go-anytype-infrastructure-experiments/pkg/acl/aclchanges/aclpb"
	"github.com/anytypeio/go-anytype-infrastructure-experiments/pkg/acl/list"
	"github.com/anytypeio/go-anytype-infrastructure-experiments/pkg/acl/storage"
	"github.com/anytypeio/go-anytype-infrastructure-experiments/util/cid"
	"github.com/anytypeio/go-anytype-infrastructure-experiments/util/slice"
	"github.com/gogo/protobuf/proto"
	"go.uber.org/zap"
	"sync"
	"time"
)

type TreeUpdateListener interface {
	Update(tree DocTree)
	Rebuild(tree DocTree)
}

type RWLocker interface {
	sync.Locker
	RLock()
	RUnlock()
}

var (
	ErrHasInvalidChanges = errors.New("the change is invalid")
	ErrNoCommonSnapshot  = errors.New("trees doesn't have a common snapshot")
)

type AddResultSummary int

const (
	AddResultSummaryNothing AddResultSummary = iota
	AddResultSummaryAppend
	AddResultSummaryRebuild
)

type AddResult struct {
	OldHeads []string
	Heads    []string
	Added    []*aclpb.RawChange

	Summary AddResultSummary
}

type DocTree interface {
	RWLocker
	CommonTree
	AddContent(ctx context.Context, aclList list.ACLList, content SignableChangeContent) (*aclpb.RawChange, error)
	AddRawChanges(ctx context.Context, aclList list.ACLList, changes ...*aclpb.RawChange) (AddResult, error)
}

type docTree struct {
	treeStorage    storage.TreeStorage
	updateListener TreeUpdateListener

	id     string
	header *aclpb.Header
	tree   *Tree

	treeBuilder *treeBuilder
	validator   DocTreeValidator
	kch         *keychain

	// buffers
	difSnapshotBuf  []*aclpb.RawChange
	tmpChangesBuf   []*Change
	newSnapshotsBuf []*Change
	notSeenIdxBuf   []int

	snapshotPath []string

	sync.RWMutex
}

func BuildDocTree(t storage.TreeStorage, listener TreeUpdateListener, aclList list.ACLList) (DocTree, error) {
	treeBuilder := newTreeBuilder(t)
	validator := newTreeValidator()

	docTree := &docTree{
		treeStorage:    t,
		tree:           nil,
		treeBuilder:    treeBuilder,
		validator:      validator,
		updateListener: listener,
		tmpChangesBuf:  make([]*Change, 0, 10),
		difSnapshotBuf: make([]*aclpb.RawChange, 0, 10),
		notSeenIdxBuf:  make([]int, 0, 10),
		kch:            newKeychain(),
	}
	err := docTree.rebuildFromStorage(aclList, nil)
	if err != nil {
		return nil, err
	}
	storageHeads, err := t.Heads()
	if err != nil {
		return nil, err
	}
	// comparing rebuilt heads with heads in storage
	// in theory it can happen that we didn't set heads because the process has crashed
	// therefore we want to set them later
	if !slice.UnsortedEquals(storageHeads, docTree.tree.Heads()) {
		log.With(zap.Strings("storage", storageHeads), zap.Strings("rebuilt", docTree.tree.Heads())).
			Errorf("the heads in storage and tree are different")
		err = t.SetHeads(docTree.tree.Heads())
		if err != nil {
			return nil, err
		}
	}

	docTree.id, err = t.ID()
	if err != nil {
		return nil, err
	}
	docTree.header, err = t.Header()
	if err != nil {
		return nil, err
	}

	if listener != nil {
		listener.Rebuild(docTree)
	}

	return docTree, nil
}

func (d *docTree) rebuildFromStorage(aclList list.ACLList, newChanges []*Change) (err error) {
	d.treeBuilder.Init(d.kch)

	d.tree, err = d.treeBuilder.Build(newChanges)
	if err != nil {
		return
	}

	// during building the tree we may have marked some changes as possible roots,
	// but obviously they are not roots, because of the way how we construct the tree
	d.tree.clearPossibleRoots()

	return d.validator.ValidateTree(d.tree, aclList)
}

func (d *docTree) ID() string {
	return d.id
}

func (d *docTree) Header() *aclpb.Header {
	return d.header
}

func (d *docTree) Storage() storage.TreeStorage {
	return d.treeStorage
}

func (d *docTree) AddContent(ctx context.Context, aclList list.ACLList, content SignableChangeContent) (rawChange *aclpb.RawChange, err error) {
	defer func() {
		if d.updateListener != nil {
			d.updateListener.Update(d)
		}
	}()
	state := aclList.ACLState() // special method for own keys
	aclChange := &aclpb.Change{
		TreeHeadIds:        d.tree.Heads(),
		AclHeadId:          aclList.Head().Id,
		SnapshotBaseId:     d.tree.RootId(),
		CurrentReadKeyHash: state.CurrentReadKeyHash(),
		Timestamp:          int64(time.Now().Nanosecond()),
		Identity:           content.Identity,
		IsSnapshot:         content.IsSnapshot,
	}

	marshalledData, err := content.Proto.Marshal()
	if err != nil {
		return nil, err
	}

	readKey, err := state.CurrentReadKey()
	if err != nil {
		return nil, err
	}

	encrypted, err := readKey.Encrypt(marshalledData)
	if err != nil {
		return nil, err
	}
	aclChange.ChangesData = encrypted

	fullMarshalledChange, err := proto.Marshal(aclChange)
	if err != nil {
		return nil, err
	}

	signature, err := content.Key.Sign(fullMarshalledChange)
	if err != nil {
		return nil, err
	}

	id, err := cid.NewCIDFromBytes(fullMarshalledChange)
	if err != nil {
		return nil, err
	}

	docChange := NewChange(id, aclChange, signature)
	docChange.ParsedModel = content

	if content.IsSnapshot {
		// clearing tree, because we already fixed everything in the last snapshot
		d.tree = &Tree{}
	}
	err = d.tree.AddMergedHead(docChange)
	if err != nil {
		panic(err)
	}
	rawChange = &aclpb.RawChange{
		Payload:   fullMarshalledChange,
		Signature: docChange.Signature(),
		Id:        docChange.Id,
	}

	err = d.treeStorage.AddRawChange(rawChange)
	if err != nil {
		return
	}

	err = d.treeStorage.SetHeads([]string{docChange.Id})
	return
}

func (d *docTree) AddRawChanges(ctx context.Context, aclList list.ACLList, rawChanges ...*aclpb.RawChange) (addResult AddResult, err error) {
	var mode Mode
	mode, addResult, err = d.addRawChanges(ctx, aclList, rawChanges...)
	if err != nil {
		return
	}

	// reducing tree if we have new roots
	d.tree.reduceTree()

	// adding to database all the added changes only after they are good
	for _, ch := range addResult.Added {
		err = d.treeStorage.AddRawChange(ch)
		if err != nil {
			return
		}
	}

	// setting heads
	err = d.treeStorage.SetHeads(d.tree.Heads())
	if err != nil {
		return
	}

	if d.updateListener == nil {
		return
	}

	switch mode {
	case Append:
		d.updateListener.Update(d)
	case Rebuild:
		d.updateListener.Rebuild(d)
	default:
		break
	}
	return
}

func (d *docTree) addRawChanges(ctx context.Context, aclList list.ACLList, rawChanges ...*aclpb.RawChange) (mode Mode, addResult AddResult, err error) {
	// resetting buffers
	d.tmpChangesBuf = d.tmpChangesBuf[:0]
	d.notSeenIdxBuf = d.notSeenIdxBuf[:0]
	d.difSnapshotBuf = d.difSnapshotBuf[:0]
	d.newSnapshotsBuf = d.newSnapshotsBuf[:0]

	// this will be returned to client so we shouldn't use buffer here
	prevHeadsCopy := make([]string, 0, len(d.tree.Heads()))
	copy(prevHeadsCopy, d.tree.Heads())

	// filtering changes, verifying and unmarshalling them
	for idx, ch := range rawChanges {
		if d.HasChange(ch.Id) {
			continue
		}

		var change *Change
		change, err = NewVerifiedChangeFromRaw(ch, d.kch)
		if err != nil {
			return
		}

		if change.IsSnapshot {
			d.newSnapshotsBuf = append(d.newSnapshotsBuf, change)
		}
		d.tmpChangesBuf = append(d.tmpChangesBuf, change)
		d.notSeenIdxBuf = append(d.notSeenIdxBuf, idx)
	}

	// if no new changes, then returning
	if len(d.notSeenIdxBuf) == 0 {
		addResult = AddResult{
			OldHeads: prevHeadsCopy,
			Heads:    prevHeadsCopy,
			Summary:  AddResultSummaryNothing,
		}
		return
	}

	headsCopy := func() []string {
		newHeads := make([]string, 0, len(d.tree.Heads()))
		copy(newHeads, d.tree.Heads())
		return newHeads
	}

	// returns changes that we added to the tree
	getAddedChanges := func() []*aclpb.RawChange {
		var added []*aclpb.RawChange
		for _, idx := range d.notSeenIdxBuf {
			rawChange := rawChanges[idx]
			if _, exists := d.tree.attached[rawChange.Id]; exists {
				added = append(added, rawChange)
			}
		}
		return added
	}

	rollback := func() {
		for _, ch := range d.tmpChangesBuf {
			if _, exists := d.tree.attached[ch.Id]; exists {
				delete(d.tree.attached, ch.Id)
			} else if _, exists := d.tree.unAttached[ch.Id]; exists {
				delete(d.tree.unAttached, ch.Id)
			}
		}
	}

	// checks if we need to go to database
	isOldSnapshot := func(ch *Change) bool {
		if ch.SnapshotId == d.tree.RootId() {
			return false
		}
		for _, sn := range d.newSnapshotsBuf {
			// if change refers to newly received snapshot
			if ch.SnapshotId == sn.Id {
				return false
			}
		}
		return true
	}

	// checking if we have some changes with different snapshot and then rebuilding
	for _, ch := range d.tmpChangesBuf {
		if isOldSnapshot(ch) {
			err = d.rebuildFromStorage(aclList, d.tmpChangesBuf)
			if err != nil {
				// rebuilding without new changes
				d.rebuildFromStorage(aclList, nil)
				return
			}

			addResult = AddResult{
				OldHeads: prevHeadsCopy,
				Heads:    headsCopy(),
				Added:    getAddedChanges(),
				Summary:  AddResultSummaryRebuild,
			}
			return
		}
	}

	// normal mode of operation, where we don't need to rebuild from database
	mode = d.tree.Add(d.tmpChangesBuf...)
	switch mode {
	case Nothing:
		addResult = AddResult{
			OldHeads: prevHeadsCopy,
			Heads:    prevHeadsCopy,
			Summary:  AddResultSummaryNothing,
		}
		return

	default:
		// just rebuilding the state from start without reloading everything from tree storage
		// as an optimization we could've started from current heads, but I didn't implement that
		err = d.validator.ValidateTree(d.tree, aclList)
		if err != nil {
			rollback()
			err = ErrHasInvalidChanges
			return
		}

		addResult = AddResult{
			OldHeads: prevHeadsCopy,
			Heads:    headsCopy(),
			Added:    getAddedChanges(),
			Summary:  AddResultSummaryAppend,
		}
	}
	return
}

func (d *docTree) Iterate(f func(change *Change) bool) {
	d.tree.Iterate(d.tree.RootId(), f)
}

func (d *docTree) IterateFrom(s string, f func(change *Change) bool) {
	d.tree.Iterate(s, f)
}

func (d *docTree) HasChange(s string) bool {
	_, attachedExists := d.tree.attached[s]
	_, unattachedExists := d.tree.unAttached[s]
	return attachedExists || unattachedExists
}

func (d *docTree) Heads() []string {
	return d.tree.Heads()
}

func (d *docTree) Root() *Change {
	return d.tree.Root()
}

func (d *docTree) Close() error {
	return nil
}

func (d *docTree) SnapshotPath() []string {
	if d.snapshotPathIsActual() {
		return d.snapshotPath
	}

	var path []string
	// TODO: think that the user may have not all of the snapshots locally
	currentSnapshotId := d.tree.RootId()
	for currentSnapshotId != "" {
		sn, err := d.treeBuilder.loadChange(currentSnapshotId)
		if err != nil {
			break
		}
		path = append(path, currentSnapshotId)
		currentSnapshotId = sn.SnapshotId
	}
	d.snapshotPath = path

	return path
}

func (d *docTree) ChangesAfterCommonSnapshot(theirPath []string) ([]*aclpb.RawChange, error) {
	var (
		needFullDocument = len(theirPath) == 0
		ourPath          = d.SnapshotPath()
		// by default returning everything we have
		commonSnapshot = ourPath[len(ourPath)-1]
		err            error
	)

	// if this is non-empty request
	if !needFullDocument {
		commonSnapshot, err = commonSnapshotForTwoPaths(ourPath, theirPath)
		if err != nil {
			return nil, err
		}
	}

	log.With(
		zap.Strings("heads", d.tree.Heads()),
		zap.String("breakpoint", commonSnapshot),
		zap.String("id", d.id)).
		Debug("getting all changes from common snapshot")

	if commonSnapshot == d.tree.RootId() {
		return d.getChangesFromTree()
	} else {
		return d.getChangesFromDB(commonSnapshot, needFullDocument)
	}
}

func (d *docTree) getChangesFromTree() (rawChanges []*aclpb.RawChange, err error) {
	d.tree.dfsPrev(d.tree.HeadsChanges(), func(ch *Change) bool {
		var marshalled []byte
		marshalled, err = ch.Content.Marshal()
		if err != nil {
			return false
		}
		raw := &aclpb.RawChange{
			Payload:   marshalled,
			Signature: ch.Signature(),
			Id:        ch.Id,
		}
		rawChanges = append(rawChanges, raw)
		return true
	}, func(changes []*Change) {})

	return
}

func (d *docTree) getChangesFromDB(commonSnapshot string, needStartSnapshot bool) (rawChanges []*aclpb.RawChange, err error) {
	load := func(id string) (*Change, error) {
		raw, err := d.treeStorage.GetRawChange(context.Background(), id)
		if err != nil {
			return nil, err
		}

		ch, err := NewChangeFromRaw(raw)
		if err != nil {
			return nil, err
		}

		rawChanges = append(rawChanges, raw)
		return ch, nil
	}

	_, err = d.treeBuilder.dfs(d.tree.Heads(), commonSnapshot, load)
	if err != nil {
		return
	}

	if needStartSnapshot {
		// adding snapshot to raw changes
		_, err = load(commonSnapshot)
	}

	return
}

func (d *docTree) snapshotPathIsActual() bool {
	return len(d.snapshotPath) != 0 && d.snapshotPath[len(d.snapshotPath)-1] == d.tree.RootId()
}

func (d *docTree) DebugDump() (string, error) {
	return d.tree.Graph(NoOpDescriptionParser)
}
