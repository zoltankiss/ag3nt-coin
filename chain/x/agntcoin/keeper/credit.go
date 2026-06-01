package keeper

import (
	"context"
	"errors"

	"cosmossdk.io/collections"
	errorsmod "cosmossdk.io/errors"
	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
	"github.com/zoltankiss/agntcoin/x/agntcoin/types"
)

// checkNewAccountCredit enforces the dust-spam guard up front: if `addrStr`
// has no existing Account record, the credited amount must be at least
// types.MinNewAccountCredit. Call this before mutating any state so a rejected
// payout does not leave the payer debited.
func (k Keeper) checkNewAccountCredit(ctx context.Context, addrStr string, amount uint64) error {
	_, err := k.Account.Get(ctx, addrStr)
	if err != nil {
		if !errors.Is(err, collections.ErrNotFound) {
			return errorsmod.Wrap(sdkerrors.ErrIO, err.Error())
		}
		if amount < types.MinNewAccountCredit {
			return errorsmod.Wrapf(
				sdkerrors.ErrInvalidRequest,
				"credit to new account must be at least %d", types.MinNewAccountCredit,
			)
		}
	}
	return nil
}

// creditAccount credits `amount` to the agntcoin balance of `addrStr`,
// auto-registering the recipient and creating an x/auth BaseAccount if the
// recipient does not yet exist. `addrBytes` must be the decoded form of
// `addrStr`.
//
// To bound dust-spam account creation, if the recipient does NOT already
// exist (no prior Account record), the credited amount must be at least
// types.MinNewAccountCredit; otherwise the call is rejected.
//
// This consolidates the recipient auto-register + x/auth-create logic
// originally written inline in the Transfer handler so it can be reused by
// escrow payouts.
func (k Keeper) creditAccount(ctx context.Context, addrStr string, addrBytes []byte, amount uint64) error {
	to, err := k.Account.Get(ctx, addrStr)
	if err != nil {
		if !errors.Is(err, collections.ErrNotFound) {
			return errorsmod.Wrap(sdkerrors.ErrIO, err.Error())
		}
		// Brand-new recipient: enforce the spam guard before creating it.
		if amount < types.MinNewAccountCredit {
			return errorsmod.Wrapf(
				sdkerrors.ErrInvalidRequest,
				"credit to new account must be at least %d", types.MinNewAccountCredit,
			)
		}
		to = types.Account{
			Address:       addrStr,
			Balance:       0,
			Nonce:         0,
			Registered:    true,
			FaucetClaimed: false,
		}
	} else if !to.Registered {
		// Account record exists but was never registered; mark it registered.
		to.Registered = true
	}

	// Ensure the recipient has an x/auth BaseAccount so they can later sign
	// (spend) on their own.
	recipientAddr := sdk.AccAddress(addrBytes)
	if k.authKeeper != nil && !k.authKeeper.HasAccount(ctx, recipientAddr) {
		acc := k.authKeeper.NewAccountWithAddress(ctx, recipientAddr)
		k.authKeeper.SetAccount(ctx, acc)
	}

	to.Balance += amount
	if err := k.Account.Set(ctx, addrStr, to); err != nil {
		return errorsmod.Wrap(sdkerrors.ErrIO, err.Error())
	}
	return nil
}

// ensureRegistered makes sure `addrStr` has a registered Account and an x/auth
// BaseAccount WITHOUT crediting any balance. Used at escrow-LOCK time so a
// brand-new payee can sign (e.g. escrow-submit) on its very first job, before
// any funds have been released to it — closing the "first job is unprotectable"
// hole. The same dust-spam guard applies: a brand-new payee is only created if
// `guardAmount` (the escrow amount) is at least MinNewAccountCredit.
func (k Keeper) ensureRegistered(ctx context.Context, addrStr string, addrBytes []byte, guardAmount uint64) error {
	to, err := k.Account.Get(ctx, addrStr)
	if err != nil {
		if !errors.Is(err, collections.ErrNotFound) {
			return errorsmod.Wrap(sdkerrors.ErrIO, err.Error())
		}
		if guardAmount < types.MinNewAccountCredit {
			return errorsmod.Wrapf(
				sdkerrors.ErrInvalidRequest,
				"escrow to a brand-new payee must be at least %d", types.MinNewAccountCredit,
			)
		}
		to = types.Account{Address: addrStr, Balance: 0, Nonce: 0, Registered: true, FaucetClaimed: false}
	} else if !to.Registered {
		to.Registered = true
	}

	recipientAddr := sdk.AccAddress(addrBytes)
	if k.authKeeper != nil && !k.authKeeper.HasAccount(ctx, recipientAddr) {
		acc := k.authKeeper.NewAccountWithAddress(ctx, recipientAddr)
		k.authKeeper.SetAccount(ctx, acc)
	}
	if err := k.Account.Set(ctx, addrStr, to); err != nil {
		return errorsmod.Wrap(sdkerrors.ErrIO, err.Error())
	}
	return nil
}
