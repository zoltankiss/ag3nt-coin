package keeper_test

import (
	"context"
	"testing"

	"cosmossdk.io/core/address"
	storetypes "cosmossdk.io/store/types"
	addresscodec "github.com/cosmos/cosmos-sdk/codec/address"
	"github.com/cosmos/cosmos-sdk/runtime"
	"github.com/cosmos/cosmos-sdk/testutil"
	sdk "github.com/cosmos/cosmos-sdk/types"
	moduletestutil "github.com/cosmos/cosmos-sdk/types/module/testutil"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"

	"github.com/zoltankiss/agntcoin/x/agntcoin/keeper"
	module "github.com/zoltankiss/agntcoin/x/agntcoin/module"
	"github.com/zoltankiss/agntcoin/x/agntcoin/types"
)

// mockAuthKeeper is a minimal in-memory implementation of types.AuthKeeper for tests.
type mockAuthKeeper struct {
	codec    address.Codec
	accounts map[string]sdk.AccountI
}

func newMockAuthKeeper(codec address.Codec) *mockAuthKeeper {
	return &mockAuthKeeper{codec: codec, accounts: map[string]sdk.AccountI{}}
}

func (m *mockAuthKeeper) AddressCodec() address.Codec { return m.codec }

func (m *mockAuthKeeper) GetAccount(_ context.Context, addr sdk.AccAddress) sdk.AccountI {
	return m.accounts[addr.String()]
}

func (m *mockAuthKeeper) HasAccount(_ context.Context, addr sdk.AccAddress) bool {
	_, ok := m.accounts[addr.String()]
	return ok
}

func (m *mockAuthKeeper) NewAccountWithAddress(_ context.Context, addr sdk.AccAddress) sdk.AccountI {
	acc := &authtypes.BaseAccount{}
	_ = acc.SetAddress(addr)
	return acc
}

func (m *mockAuthKeeper) SetAccount(_ context.Context, acc sdk.AccountI) {
	m.accounts[acc.GetAddress().String()] = acc
}

type fixture struct {
	ctx          context.Context
	keeper       keeper.Keeper
	addressCodec address.Codec
}

func initFixture(t *testing.T) *fixture {
	t.Helper()

	encCfg := moduletestutil.MakeTestEncodingConfig(module.AppModule{})
	addressCodec := addresscodec.NewBech32Codec(sdk.GetConfig().GetBech32AccountAddrPrefix())
	storeKey := storetypes.NewKVStoreKey(types.StoreKey)

	storeService := runtime.NewKVStoreService(storeKey)
	ctx := testutil.DefaultContextWithDB(t, storeKey, storetypes.NewTransientStoreKey("transient_test")).Ctx

	authority := authtypes.NewModuleAddress(types.GovModuleName)

	k := keeper.NewKeeper(
		storeService,
		encCfg.Codec,
		addressCodec,
		authority,
		newMockAuthKeeper(addressCodec),
	)

	// Initialize params
	if err := k.Params.Set(ctx, types.DefaultParams()); err != nil {
		t.Fatalf("failed to set params: %v", err)
	}

	return &fixture{
		ctx:          ctx,
		keeper:       k,
		addressCodec: addressCodec,
	}
}
