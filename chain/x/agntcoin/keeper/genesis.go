package keeper

import (
	"context"

	"github.com/zoltankiss/agntcoin/x/agntcoin/types"
)

// InitGenesis initializes the module's state from a provided genesis state.
func (k Keeper) InitGenesis(ctx context.Context, genState types.GenesisState) error {
	if genState.EmissionState.MaxSupply == 0 {
		genState.EmissionState = types.DefaultEmissionState(0)
	}
	if genState.EmissionState.GenesisTime == 0 {
		now := sdkBlockTime(ctx)
		genState.EmissionState.GenesisTime = now
		genState.EmissionState.EpochStartedAt = now
	}
	if err := k.EmissionState.Set(ctx, genState.EmissionState); err != nil {
		return err
	}

	for _, elem := range genState.AccountMap {
		if err := k.Account.Set(ctx, elem.Address, elem); err != nil {
			return err
		}
	}
	for _, elem := range genState.VouchList {
		if err := k.Vouch.Set(ctx, elem.Id, elem); err != nil {
			return err
		}
	}

	if err := k.VouchSeq.Set(ctx, genState.VouchCount); err != nil {
		return err
	}

	for _, elem := range genState.EscrowList {
		if err := k.Escrow.Set(ctx, elem.Id, elem); err != nil {
			return err
		}
	}

	if err := k.EscrowSeq.Set(ctx, genState.EscrowCount); err != nil {
		return err
	}

	for _, elem := range genState.ContributionAwardList {
		if err := k.ContributionAward.Set(ctx, elem.Id, elem); err != nil {
			return err
		}
	}

	if err := k.ContributionAwardSeq.Set(ctx, genState.ContributionAwardCount); err != nil {
		return err
	}

	for _, elem := range genState.ScopedEvidenceVouchList {
		if err := k.ScopedEvidenceVouch.Set(ctx, elem.Id, elem); err != nil {
			return err
		}
	}

	if err := k.ScopedEvidenceVouchSeq.Set(ctx, genState.ScopedEvidenceVouchCount); err != nil {
		return err
	}

	return k.Params.Set(ctx, genState.Params)
}

// ExportGenesis returns the module's exported genesis.
func (k Keeper) ExportGenesis(ctx context.Context) (*types.GenesisState, error) {
	var err error

	genesis := types.DefaultGenesis()
	genesis.Params, err = k.Params.Get(ctx)
	if err != nil {
		return nil, err
	}
	genesis.EmissionState, err = k.getEmissionState(ctx)
	if err != nil {
		return nil, err
	}
	if err := k.Account.Walk(ctx, nil, func(_ string, val types.Account) (stop bool, err error) {
		genesis.AccountMap = append(genesis.AccountMap, val)
		return false, nil
	}); err != nil {
		return nil, err
	}
	err = k.Vouch.Walk(ctx, nil, func(key uint64, elem types.Vouch) (bool, error) {
		genesis.VouchList = append(genesis.VouchList, elem)
		return false, nil
	})
	if err != nil {
		return nil, err
	}

	genesis.VouchCount, err = k.VouchSeq.Peek(ctx)
	if err != nil {
		return nil, err
	}

	err = k.Escrow.Walk(ctx, nil, func(key uint64, elem types.Escrow) (bool, error) {
		genesis.EscrowList = append(genesis.EscrowList, elem)
		return false, nil
	})
	if err != nil {
		return nil, err
	}

	genesis.EscrowCount, err = k.EscrowSeq.Peek(ctx)
	if err != nil {
		return nil, err
	}

	err = k.ContributionAward.Walk(ctx, nil, func(key uint64, elem types.ContributionAward) (bool, error) {
		genesis.ContributionAwardList = append(genesis.ContributionAwardList, elem)
		return false, nil
	})
	if err != nil {
		return nil, err
	}

	genesis.ContributionAwardCount, err = k.ContributionAwardSeq.Peek(ctx)
	if err != nil {
		return nil, err
	}

	err = k.ScopedEvidenceVouch.Walk(ctx, nil, func(key uint64, elem types.ScopedEvidenceVouch) (bool, error) {
		genesis.ScopedEvidenceVouchList = append(genesis.ScopedEvidenceVouchList, elem)
		return false, nil
	})
	if err != nil {
		return nil, err
	}

	genesis.ScopedEvidenceVouchCount, err = k.ScopedEvidenceVouchSeq.Peek(ctx)
	if err != nil {
		return nil, err
	}

	return genesis, nil
}
