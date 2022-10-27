package list

import (
	"bytes"
	"errors"
	"fmt"
	"github.com/anytypeio/go-anytype-infrastructure-experiments/common/app/logger"
	"github.com/anytypeio/go-anytype-infrastructure-experiments/common/pkg/acl/aclrecordproto"
	"github.com/anytypeio/go-anytype-infrastructure-experiments/common/pkg/acl/common"
	"github.com/anytypeio/go-anytype-infrastructure-experiments/common/util/keys"
	"github.com/anytypeio/go-anytype-infrastructure-experiments/common/util/keys/asymmetric/encryptionkey"
	"github.com/anytypeio/go-anytype-infrastructure-experiments/common/util/keys/asymmetric/signingkey"
	"github.com/anytypeio/go-anytype-infrastructure-experiments/common/util/keys/symmetric"
	"github.com/gogo/protobuf/proto"
	"go.uber.org/zap"
	"hash/fnv"
)

var log = logger.NewNamed("acllist").Sugar()

var ErrNoSuchUser = errors.New("no such user")
var ErrFailedToDecrypt = errors.New("failed to decrypt key")
var ErrUserRemoved = errors.New("user was removed from the document")
var ErrDocumentForbidden = errors.New("your user was forbidden access to the document")
var ErrUserAlreadyExists = errors.New("user already exists")
var ErrNoSuchRecord = errors.New("no such record")
var ErrNoSuchInvite = errors.New("no such invite")
var ErrOldInvite = errors.New("invite is too old")
var ErrInsufficientPermissions = errors.New("insufficient permissions")
var ErrNoReadKey = errors.New("acl state doesn't have a read key")
var ErrInvalidSignature = errors.New("signature is invalid")
var ErrIncorrectRoot = errors.New("incorrect root")

type UserPermissionPair struct {
	Identity   string
	Permission aclrecordproto.ACLUserPermissions
}

type ACLState struct {
	id                 string
	currentReadKeyHash uint64
	userReadKeys       map[uint64]*symmetric.Key
	userStates         map[string]*aclrecordproto.ACLUserState
	userInvites        map[string]*aclrecordproto.ACLUserInvite
	encryptionKey      encryptionkey.PrivKey
	signingKey         signingkey.PrivKey
	totalReadKeys      int

	identity            string
	permissionsAtRecord map[string][]UserPermissionPair

	keychain *common.Keychain
}

func newACLStateWithKeys(
	id string,
	signingKey signingkey.PrivKey,
	encryptionKey encryptionkey.PrivKey) (*ACLState, error) {
	identity, err := signingKey.GetPublic().Raw()
	if err != nil {
		return nil, err
	}
	return &ACLState{
		id:                  id,
		identity:            string(identity),
		signingKey:          signingKey,
		encryptionKey:       encryptionKey,
		userReadKeys:        make(map[uint64]*symmetric.Key),
		userStates:          make(map[string]*aclrecordproto.ACLUserState),
		userInvites:         make(map[string]*aclrecordproto.ACLUserInvite),
		permissionsAtRecord: make(map[string][]UserPermissionPair),
	}, nil
}

func newACLState(id string) *ACLState {
	return &ACLState{
		id:                  id,
		userReadKeys:        make(map[uint64]*symmetric.Key),
		userStates:          make(map[string]*aclrecordproto.ACLUserState),
		userInvites:         make(map[string]*aclrecordproto.ACLUserInvite),
		permissionsAtRecord: make(map[string][]UserPermissionPair),
	}
}

func (st *ACLState) CurrentReadKeyHash() uint64 {
	return st.currentReadKeyHash
}

func (st *ACLState) CurrentReadKey() (*symmetric.Key, error) {
	key, exists := st.userReadKeys[st.currentReadKeyHash]
	if !exists {
		return nil, ErrNoReadKey
	}
	return key, nil
}

func (st *ACLState) UserReadKeys() map[uint64]*symmetric.Key {
	return st.userReadKeys
}

func (st *ACLState) PermissionsAtRecord(id string, identity string) (UserPermissionPair, error) {
	permissions, ok := st.permissionsAtRecord[id]
	if !ok {
		log.Errorf("missing record at id %s", id)
		return UserPermissionPair{}, ErrNoSuchRecord
	}

	for _, perm := range permissions {
		if perm.Identity == identity {
			return perm, nil
		}
	}
	return UserPermissionPair{}, ErrNoSuchUser
}

func (st *ACLState) applyRecord(record *ACLRecord) (err error) {
	if record.Id == st.id {
		root, ok := record.Model.(*aclrecordproto.ACLRoot)
		if !ok {
			return ErrIncorrectRoot
		}
		st.permissionsAtRecord[record.Id] = []UserPermissionPair{
			{Identity: string(root.Identity), Permission: aclrecordproto.ACLUserPermissions_Admin},
		}
		return st.applyRoot(root)
	}
	aclData := &aclrecordproto.ACLData{}

	if record.Model != nil {
		aclData = record.Model.(*aclrecordproto.ACLData)
	} else {
		err = proto.Unmarshal(record.Data, aclData)
		if err != nil {
			return
		}
		record.Model = aclData
	}

	err = st.applyChangeData(aclData, record.CurrentReadKeyHash, record.Identity)
	if err != nil {
		return
	}

	// getting all permissions for users at record
	var permissions []UserPermissionPair
	for _, state := range st.userStates {
		permission := UserPermissionPair{
			Identity:   string(state.Identity),
			Permission: state.Permissions,
		}
		permissions = append(permissions, permission)
	}

	st.permissionsAtRecord[record.Id] = permissions
	return nil
}

func (st *ACLState) applyRoot(root *aclrecordproto.ACLRoot) (err error) {
	if st.signingKey != nil && st.encryptionKey != nil && st.identity == string(root.Identity) {
		err = st.saveReadKeyFromRoot(root)
		if err != nil {
			return
		}
	}

	// adding user to the list
	userState := &aclrecordproto.ACLUserState{
		Identity:      root.Identity,
		EncryptionKey: root.EncryptionKey,
		Permissions:   aclrecordproto.ACLUserPermissions_Admin,
	}
	st.userStates[string(root.Identity)] = userState
	return
}

func (st *ACLState) saveReadKeyFromRoot(root *aclrecordproto.ACLRoot) (err error) {
	var readKey *symmetric.Key
	if len(root.GetDerivationScheme()) != 0 {
		var encPubKey []byte
		encPubKey, err = st.encryptionKey.GetPublic().Raw()
		if err != nil {
			return
		}

		readKey, err = aclrecordproto.ACLReadKeyDerive([]byte(st.identity), encPubKey)
		if err != nil {
			return
		}
	} else {
		readKey, _, err = st.decryptReadKeyAndHash(root.EncryptedReadKey)
		if err != nil {
			return
		}
	}

	hasher := fnv.New64()
	_, err = hasher.Write(readKey.Bytes())
	if err != nil {
		return
	}
	if hasher.Sum64() != root.CurrentReadKeyHash {
		return ErrIncorrectRoot
	}
	st.currentReadKeyHash = root.CurrentReadKeyHash
	st.userReadKeys[root.CurrentReadKeyHash] = readKey
	st.totalReadKeys++

	return
}

func (st *ACLState) applyChangeData(changeData *aclrecordproto.ACLData, hash uint64, identity []byte) (err error) {
	defer func() {
		if err != nil {
			return
		}
		if hash != st.currentReadKeyHash {
			st.totalReadKeys++
			st.currentReadKeyHash = hash
		}
	}()

	if !st.isUserJoin(changeData) {
		// we check signature when we add this to the List, so no need to do it here
		if _, exists := st.userStates[string(identity)]; !exists {
			err = ErrNoSuchUser
			return
		}

		if !st.hasPermission(identity, aclrecordproto.ACLUserPermissions_Admin) {
			err = fmt.Errorf("user %s must have admin permissions", identity)
			return
		}
	}

	for _, ch := range changeData.GetAclContent() {
		if err = st.applyChangeContent(ch); err != nil {
			log.Info("error while applying changes: %v; ignore", zap.Error(err))
			return err
		}
	}

	return nil
}

func (st *ACLState) applyChangeContent(ch *aclrecordproto.ACLContentValue) error {
	switch {
	case ch.GetUserPermissionChange() != nil:
		return st.applyUserPermissionChange(ch.GetUserPermissionChange())
	case ch.GetUserAdd() != nil:
		return st.applyUserAdd(ch.GetUserAdd())
	case ch.GetUserRemove() != nil:
		return st.applyUserRemove(ch.GetUserRemove())
	case ch.GetUserInvite() != nil:
		return st.applyUserInvite(ch.GetUserInvite())
	case ch.GetUserJoin() != nil:
		return st.applyUserJoin(ch.GetUserJoin())
	default:
		return fmt.Errorf("unexpected change type: %v", ch)
	}
}

func (st *ACLState) applyUserPermissionChange(ch *aclrecordproto.ACLUserPermissionChange) error {
	chIdentity := string(ch.Identity)
	state, exists := st.userStates[chIdentity]
	if !exists {
		return ErrNoSuchUser
	}

	state.Permissions = ch.Permissions
	return nil
}

func (st *ACLState) applyUserInvite(ch *aclrecordproto.ACLUserInvite) error {
	st.userInvites[string(ch.AcceptPublicKey)] = ch
	return nil
}

func (st *ACLState) applyUserJoin(ch *aclrecordproto.ACLUserJoin) error {
	invite, exists := st.userInvites[string(ch.AcceptPubKey)]
	if !exists {
		return fmt.Errorf("no such invite with such public key %s", keys.EncodeBytesToString(ch.AcceptPubKey))
	}
	chIdentity := string(ch.Identity)

	if _, exists = st.userStates[chIdentity]; exists {
		return ErrUserAlreadyExists
	}

	// validating signature
	signature := ch.GetAcceptSignature()
	verificationKey, err := signingkey.NewSigningEd25519PubKeyFromBytes(invite.AcceptPublicKey)
	if err != nil {
		return fmt.Errorf("public key verifying invite accepts is given in incorrect format: %v", err)
	}

	res, err := verificationKey.Verify(ch.Identity, signature)
	if err != nil {
		return fmt.Errorf("verification returned error: %w", err)
	}
	if !res {
		return ErrInvalidSignature
	}

	// if ourselves -> we need to decrypt the read keys
	if st.identity == chIdentity {
		for _, key := range ch.EncryptedReadKeys {
			key, hash, err := st.decryptReadKeyAndHash(key)
			if err != nil {
				return ErrFailedToDecrypt
			}

			st.userReadKeys[hash] = key
		}
	}

	// adding user to the list
	userState := &aclrecordproto.ACLUserState{
		Identity:      ch.Identity,
		EncryptionKey: ch.EncryptionKey,
		Permissions:   invite.Permissions,
	}
	st.userStates[chIdentity] = userState
	return nil
}

func (st *ACLState) applyUserAdd(ch *aclrecordproto.ACLUserAdd) error {
	chIdentity := string(ch.Identity)
	if _, exists := st.userStates[chIdentity]; exists {
		return ErrUserAlreadyExists
	}

	st.userStates[chIdentity] = &aclrecordproto.ACLUserState{
		Identity:      ch.Identity,
		EncryptionKey: ch.EncryptionKey,
		Permissions:   ch.Permissions,
	}

	if chIdentity == st.identity {
		for _, key := range ch.EncryptedReadKeys {
			key, hash, err := st.decryptReadKeyAndHash(key)
			if err != nil {
				return ErrFailedToDecrypt
			}

			st.userReadKeys[hash] = key
		}
	}

	return nil
}

func (st *ACLState) applyUserRemove(ch *aclrecordproto.ACLUserRemove) error {
	chIdentity := string(ch.Identity)
	if chIdentity == st.identity {
		return ErrDocumentForbidden
	}

	if _, exists := st.userStates[chIdentity]; !exists {
		return ErrNoSuchUser
	}

	delete(st.userStates, chIdentity)

	for _, replace := range ch.ReadKeyReplaces {
		repIdentity := string(replace.Identity)
		// if this is our identity then we have to decrypt the key
		if repIdentity == st.identity {
			key, hash, err := st.decryptReadKeyAndHash(replace.EncryptedReadKey)
			if err != nil {
				return ErrFailedToDecrypt
			}

			st.userReadKeys[hash] = key
			break
		}
	}
	return nil
}

func (st *ACLState) decryptReadKeyAndHash(msg []byte) (*symmetric.Key, uint64, error) {
	decrypted, err := st.encryptionKey.Decrypt(msg)
	if err != nil {
		return nil, 0, ErrFailedToDecrypt
	}

	key, err := symmetric.FromBytes(decrypted)
	if err != nil {
		return nil, 0, ErrFailedToDecrypt
	}

	hasher := fnv.New64()
	hasher.Write(decrypted)
	return key, hasher.Sum64(), nil
}

func (st *ACLState) hasPermission(identity []byte, permission aclrecordproto.ACLUserPermissions) bool {
	state, exists := st.userStates[string(identity)]
	if !exists {
		return false
	}

	return state.Permissions == permission
}

func (st *ACLState) isUserJoin(data *aclrecordproto.ACLData) bool {
	// if we have a UserJoin, then it should always be the first one applied
	return data.GetAclContent() != nil && data.GetAclContent()[0].GetUserJoin() != nil
}

func (st *ACLState) isUserAdd(data *aclrecordproto.ACLData, identity []byte) bool {
	// if we have a UserAdd, then it should always be the first one applied
	userAdd := data.GetAclContent()[0].GetUserAdd()
	return data.GetAclContent() != nil && userAdd != nil && bytes.Compare(userAdd.GetIdentity(), identity) == 0
}

func (st *ACLState) UserStates() map[string]*aclrecordproto.ACLUserState {
	return st.userStates
}

func (st *ACLState) Invite(acceptPubKey []byte) (invite *aclrecordproto.ACLUserInvite, err error) {
	invite, exists := st.userInvites[string(acceptPubKey)]
	if !exists {
		err = ErrNoSuchInvite
		return
	}
	if len(invite.EncryptedReadKeys) != st.totalReadKeys {
		err = ErrOldInvite
	}
	return
}

func (st *ACLState) UserKeys() (encKey encryptionkey.PrivKey, signKey signingkey.PrivKey) {
	return st.encryptionKey, st.signingKey
}

func (st *ACLState) Identity() []byte {
	return []byte(st.identity)
}
