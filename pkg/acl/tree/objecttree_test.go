package tree

import (
	"github.com/anytypeio/go-anytype-infrastructure-experiments/pkg/acl/aclchanges/aclpb"
	"github.com/anytypeio/go-anytype-infrastructure-experiments/pkg/acl/list"
	"github.com/anytypeio/go-anytype-infrastructure-experiments/pkg/acl/storage"
	"github.com/anytypeio/go-anytype-infrastructure-experiments/pkg/acl/testutils/acllistbuilder"
	"github.com/anytypeio/go-anytype-infrastructure-experiments/util/keys/asymmetric/signingkey"
	"github.com/gogo/protobuf/proto"
	"github.com/stretchr/testify/require"
	"testing"
)

type mockChangeCreator struct{}

func (c *mockChangeCreator) createRaw(id, aclId, snapshotId string, isSnapshot bool, prevIds ...string) *aclpb.RawChange {
	aclChange := &aclpb.Change{
		TreeHeadIds:    prevIds,
		AclHeadId:      aclId,
		SnapshotBaseId: snapshotId,
		ChangesData:    nil,
		IsSnapshot:     isSnapshot,
	}
	res, _ := aclChange.Marshal()
	return &aclpb.RawChange{
		Payload:   res,
		Signature: nil,
		Id:        id,
	}
}

func (c *mockChangeCreator) createNewTreeStorage(treeId, aclListId, aclHeadId, firstChangeId string) storage.TreeStorage {
	firstChange := c.createRaw(firstChangeId, aclHeadId, "", true)
	header := &aclpb.Header{
		FirstId:     firstChangeId,
		AclListId:   aclListId,
		WorkspaceId: "",
		DocType:     aclpb.Header_DocTree,
	}
	treeStorage, _ := storage.NewInMemoryTreeStorage(treeId, header, []*aclpb.RawChange{firstChange})
	return treeStorage
}

type mockChangeBuilder struct{}

func (c *mockChangeBuilder) ConvertFromRaw(rawChange *aclpb.RawChange) (ch *Change, err error) {
	unmarshalled := &aclpb.Change{}
	err = proto.Unmarshal(rawChange.Payload, unmarshalled)
	if err != nil {
		return nil, err
	}

	ch = NewChange(rawChange.Id, unmarshalled, rawChange.Signature)
	return
}
func (c *mockChangeBuilder) ConvertFromRawAndVerify(rawChange *aclpb.RawChange) (ch *Change, err error) {
	return c.ConvertFromRaw(rawChange)
}

func (c *mockChangeBuilder) BuildContent(payload BuilderContent) (ch *Change, raw *aclpb.RawChange, err error) {
	panic("implement me")
}

type mockChangeValidator struct{}

func (m *mockChangeValidator) ValidateTree(tree *Tree, aclList list.ACLList) error {
	return nil
}

func prepareACLList(t *testing.T) list.ACLList {
	st, err := acllistbuilder.NewListStorageWithTestName("userjoinexample.yml")
	require.NoError(t, err, "building storage should not result in error")

	aclList, err := list.BuildACLList(signingkey.NewEDPubKeyDecoder(), st)
	require.NoError(t, err, "building acl list should be without error")

	return aclList
}

func TestObjectTree_Build(t *testing.T) {
	aclList := prepareACLList(t)
	changeCreator := &mockChangeCreator{}
	treeStorage := changeCreator.createNewTreeStorage("treeId", aclList.ID(), aclList.Head().Id, "0")
	changeBuilder := &mockChangeBuilder{}
	deps := objectTreeDeps{
		changeBuilder:  changeBuilder,
		treeBuilder:    newTreeBuilder(treeStorage, changeBuilder),
		treeStorage:    treeStorage,
		updateListener: nil,
		validator:      &mockChangeValidator{},
		aclList:        aclList,
	}

	_, err := buildObjectTree(deps)
	require.NoError(t, err, "building tree should be without error")
}
