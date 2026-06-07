package keeper

import (
	"context"
	"errors"
	"strconv"

	"cosmossdk.io/collections"
	errorsmod "cosmossdk.io/errors"
	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
	"github.com/zoltankiss/agntcoin/x/agntcoin/types"
)

func sdkBlockTime(ctx context.Context) int64 {
	return sdk.UnwrapSDKContext(ctx).BlockTime().Unix()
}

func nextEpochScheduled(e types.EmissionState) uint64 {
	remaining := types.RemainingScheduled(e)
	if remaining == 0 {
		return 0
	}
	scheduled := remaining / e.EpochRewardDivisor
	if scheduled == 0 {
		return remaining
	}
	return scheduled
}

func (k Keeper) getEmissionState(ctx context.Context) (types.EmissionState, error) {
	e, err := k.EmissionState.Get(ctx)
	if err != nil {
		if !errors.Is(err, collections.ErrNotFound) {
			return types.EmissionState{}, errorsmod.Wrap(sdkerrors.ErrIO, err.Error())
		}
		e = types.DefaultEmissionState(sdkBlockTime(ctx))
		if err := k.EmissionState.Set(ctx, e); err != nil {
			return types.EmissionState{}, errorsmod.Wrap(sdkerrors.ErrIO, err.Error())
		}
	}
	if e.MaxSupply == 0 {
		e = types.DefaultEmissionState(sdkBlockTime(ctx))
		if err := k.EmissionState.Set(ctx, e); err != nil {
			return types.EmissionState{}, errorsmod.Wrap(sdkerrors.ErrIO, err.Error())
		}
	}
	return e, nil
}

func (k Keeper) peekEmissionState(ctx context.Context) (types.EmissionState, error) {
	e, err := k.EmissionState.Get(ctx)
	if err != nil {
		if !errors.Is(err, collections.ErrNotFound) {
			return types.EmissionState{}, errorsmod.Wrap(sdkerrors.ErrIO, err.Error())
		}
		return types.DefaultEmissionState(sdkBlockTime(ctx)), nil
	}
	if e.MaxSupply == 0 {
		return types.DefaultEmissionState(sdkBlockTime(ctx)), nil
	}
	return e, nil
}

func (k Keeper) syncEmissionState(ctx context.Context) (types.EmissionState, error) {
	e, err := k.getEmissionState(ctx)
	if err != nil {
		return types.EmissionState{}, err
	}
	now := sdkBlockTime(ctx)
	if e.GenesisTime == 0 {
		e.GenesisTime = now
	}
	if e.EpochStartedAt == 0 {
		e.EpochStartedAt = now
	}
	if e.CurrentEpoch == 0 {
		e.CurrentEpoch = 1
	}
	if e.EpochLengthSeconds == 0 {
		e.EpochLengthSeconds = types.EpochLengthSeconds
	}
	if e.EpochRewardDivisor == 0 {
		e.EpochRewardDivisor = types.EpochRewardDivisor
	}
	if e.EpochScheduled == 0 && types.RemainingScheduled(e) > 0 {
		e.EpochScheduled = nextEpochScheduled(e)
	}

	epochLen := int64(e.EpochLengthSeconds)
	for epochLen > 0 && now >= e.EpochStartedAt+epochLen {
		burned := types.RemainingEpoch(e)
		e.TotalBurnedUnclaimed += burned
		e.CurrentEpoch++
		e.EpochStartedAt += epochLen
		e.EpochMined = 0
		e.EpochScheduled = nextEpochScheduled(e)
	}

	if err := k.EmissionState.Set(ctx, e); err != nil {
		return types.EmissionState{}, errorsmod.Wrap(sdkerrors.ErrIO, err.Error())
	}
	return e, nil
}

func (k Keeper) mintProtocolReward(ctx context.Context, recipient string, recipientBytes []byte, amount uint64, rail string) error {
	if amount == 0 {
		return errorsmod.Wrap(sdkerrors.ErrInvalidRequest, "mint amount must be positive")
	}
	if err := k.checkNewAccountCredit(ctx, recipient, amount); err != nil {
		return err
	}
	e, err := k.syncEmissionState(ctx)
	if err != nil {
		return err
	}
	if amount > types.RemainingScheduled(e) {
		return errorsmod.Wrapf(sdkerrors.ErrInvalidRequest, "mint exceeds remaining max supply (%d)", types.RemainingScheduled(e))
	}
	if amount > types.RemainingEpoch(e) {
		return errorsmod.Wrapf(sdkerrors.ErrInvalidRequest, "mint exceeds remaining epoch reward (%d)", types.RemainingEpoch(e))
	}

	e.TotalMined += amount
	e.EpochMined += amount
	if err := k.EmissionState.Set(ctx, e); err != nil {
		return errorsmod.Wrap(sdkerrors.ErrIO, err.Error())
	}
	if err := k.creditAccount(ctx, recipient, recipientBytes, amount); err != nil {
		return err
	}

	sdk.UnwrapSDKContext(ctx).EventManager().EmitEvent(
		sdk.NewEvent("agntcoin_protocol_mint",
			sdk.NewAttribute("recipient", recipient),
			sdk.NewAttribute("amount", strconv.FormatUint(amount, 10)),
			sdk.NewAttribute("rail", rail),
			sdk.NewAttribute("epoch", strconv.FormatUint(e.CurrentEpoch, 10)),
			sdk.NewAttribute("epoch_mined", strconv.FormatUint(e.EpochMined, 10)),
			sdk.NewAttribute("epoch_scheduled", strconv.FormatUint(e.EpochScheduled, 10)),
			sdk.NewAttribute("total_mined", strconv.FormatUint(e.TotalMined, 10)),
			sdk.NewAttribute("total_burned_unclaimed", strconv.FormatUint(e.TotalBurnedUnclaimed, 10)),
		),
	)
	return nil
}
