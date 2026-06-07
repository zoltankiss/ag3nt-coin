package keeper

import (
	"fmt"

	"cosmossdk.io/collections"
	"cosmossdk.io/core/address"
	corestore "cosmossdk.io/core/store"
	"github.com/cosmos/cosmos-sdk/codec"

	"github.com/zoltankiss/agntcoin/x/agntcoin/types"
)

type Keeper struct {
	storeService corestore.KVStoreService
	cdc          codec.Codec
	addressCodec address.Codec
	authKeeper   types.AuthKeeper
	// Address capable of executing a MsgUpdateParams message.
	// Typically, this should be the x/gov module account.
	authority []byte

	Schema                 collections.Schema
	Params                 collections.Item[types.Params]
	EmissionState          collections.Item[types.EmissionState]
	Account                collections.Map[string, types.Account]
	VouchSeq               collections.Sequence
	Vouch                  collections.Map[uint64, types.Vouch]
	EscrowSeq              collections.Sequence
	Escrow                 collections.Map[uint64, types.Escrow]
	DisputeSeq             collections.Sequence
	Dispute                collections.Map[uint64, types.Dispute]
	BondSeq                collections.Sequence
	Bond                   collections.Map[uint64, types.Bond]
	GateSeq                collections.Sequence
	Gate                   collections.Map[uint64, types.Gate]
	ContributionAwardSeq   collections.Sequence
	ContributionAward      collections.Map[uint64, types.ContributionAward]
	ScopedEvidenceVouchSeq collections.Sequence
	ScopedEvidenceVouch    collections.Map[uint64, types.ScopedEvidenceVouch]
}

func NewKeeper(
	storeService corestore.KVStoreService,
	cdc codec.Codec,
	addressCodec address.Codec,
	authority []byte,
	authKeeper types.AuthKeeper,

) Keeper {
	if _, err := addressCodec.BytesToString(authority); err != nil {
		panic(fmt.Sprintf("invalid authority address %s: %s", authority, err))
	}

	sb := collections.NewSchemaBuilder(storeService)

	k := Keeper{
		storeService: storeService,
		cdc:          cdc,
		addressCodec: addressCodec,
		authKeeper:   authKeeper,
		authority:    authority,

		Params:        collections.NewItem(sb, types.ParamsKey, "params", codec.CollValue[types.Params](cdc)),
		EmissionState: collections.NewItem(sb, types.EmissionStateKey, "emissionState", codec.CollValue[types.EmissionState](cdc)),
		Account:       collections.NewMap(sb, types.AccountKey, "account", collections.StringKey, codec.CollValue[types.Account](cdc)), Vouch: collections.NewMap(sb, types.VouchKey, "vouch", collections.Uint64Key, codec.CollValue[types.Vouch](cdc)),
		VouchSeq:               collections.NewSequence(sb, types.VouchCountKey, "vouchSequence"),
		Escrow:                 collections.NewMap(sb, types.EscrowKey, "escrow", collections.Uint64Key, codec.CollValue[types.Escrow](cdc)),
		EscrowSeq:              collections.NewSequence(sb, types.EscrowCountKey, "escrowSequence"),
		Dispute:                collections.NewMap(sb, types.DisputeKey, "dispute", collections.Uint64Key, codec.CollValue[types.Dispute](cdc)),
		DisputeSeq:             collections.NewSequence(sb, types.DisputeCountKey, "disputeSequence"),
		Bond:                   collections.NewMap(sb, types.BondKey, "bond", collections.Uint64Key, codec.CollValue[types.Bond](cdc)),
		BondSeq:                collections.NewSequence(sb, types.BondCountKey, "bondSequence"),
		Gate:                   collections.NewMap(sb, types.GateKey, "gate", collections.Uint64Key, codec.CollValue[types.Gate](cdc)),
		GateSeq:                collections.NewSequence(sb, types.GateCountKey, "gateSequence"),
		ContributionAward:      collections.NewMap(sb, types.ContributionAwardKey, "contributionAward", collections.Uint64Key, codec.CollValue[types.ContributionAward](cdc)),
		ContributionAwardSeq:   collections.NewSequence(sb, types.ContributionAwardCountKey, "contributionAwardSequence"),
		ScopedEvidenceVouch:    collections.NewMap(sb, types.ScopedEvidenceVouchKey, "scopedEvidenceVouch", collections.Uint64Key, codec.CollValue[types.ScopedEvidenceVouch](cdc)),
		ScopedEvidenceVouchSeq: collections.NewSequence(sb, types.ScopedEvidenceVouchCountKey, "scopedEvidenceVouchSequence"),
	}
	schema, err := sb.Build()
	if err != nil {
		panic(err)
	}
	k.Schema = schema

	return k
}

// GetAuthority returns the module's authority.
func (k Keeper) GetAuthority() []byte {
	return k.authority
}
