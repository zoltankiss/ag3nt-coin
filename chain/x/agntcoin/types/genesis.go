package types

import "fmt"

// DefaultGenesis returns the default genesis state
func DefaultGenesis() *GenesisState {
	return &GenesisState{
		Params:        DefaultParams(),
		EmissionState: DefaultEmissionState(0),
		AccountMap:    []Account{}, VouchList: []Vouch{}, EscrowList: []Escrow{}, ContributionAwardList: []ContributionAward{}, ScopedEvidenceVouchList: []ScopedEvidenceVouch{}}
}

func DefaultEmissionState(startedAt int64) EmissionState {
	return EmissionState{
		MaxSupply:            MaxSupply,
		EpochLengthSeconds:   EpochLengthSeconds,
		EpochRewardDivisor:   EpochRewardDivisor,
		GenesisTime:          startedAt,
		CurrentEpoch:         1,
		EpochStartedAt:       startedAt,
		EpochScheduled:       MaxSupply / EpochRewardDivisor,
		EpochMined:           0,
		TotalMined:           0,
		TotalBurnedUnclaimed: 0,
	}
}

func RemainingScheduled(e EmissionState) uint64 {
	if e.TotalMined >= e.MaxSupply {
		return 0
	}
	remainingAfterMined := e.MaxSupply - e.TotalMined
	if e.TotalBurnedUnclaimed >= remainingAfterMined {
		return 0
	}
	return remainingAfterMined - e.TotalBurnedUnclaimed
}

func RemainingEpoch(e EmissionState) uint64 {
	if e.EpochMined >= e.EpochScheduled {
		return 0
	}
	return e.EpochScheduled - e.EpochMined
}

// Validate performs basic genesis state validation returning an error upon any
// failure.
func (gs GenesisState) Validate() error {
	accountIndexMap := make(map[string]struct{})

	for _, elem := range gs.AccountMap {
		index := fmt.Sprint(elem.Address)
		if _, ok := accountIndexMap[index]; ok {
			return fmt.Errorf("duplicated index for account")
		}
		accountIndexMap[index] = struct{}{}
	}
	vouchIdMap := make(map[uint64]bool)
	vouchCount := gs.GetVouchCount()
	for _, elem := range gs.VouchList {
		if _, ok := vouchIdMap[elem.Id]; ok {
			return fmt.Errorf("duplicated id for vouch")
		}
		if elem.Id >= vouchCount {
			return fmt.Errorf("vouch id should be lower or equal than the last id")
		}
		vouchIdMap[elem.Id] = true
	}

	escrowIdMap := make(map[uint64]bool)
	escrowCount := gs.GetEscrowCount()
	for _, elem := range gs.EscrowList {
		if _, ok := escrowIdMap[elem.Id]; ok {
			return fmt.Errorf("duplicated id for escrow")
		}
		if elem.Id >= escrowCount {
			return fmt.Errorf("escrow id should be lower or equal than the last id")
		}
		escrowIdMap[elem.Id] = true
	}

	contributionAwardIdMap := make(map[uint64]bool)
	contributionAwardCount := gs.GetContributionAwardCount()
	for _, elem := range gs.ContributionAwardList {
		if _, ok := contributionAwardIdMap[elem.Id]; ok {
			return fmt.Errorf("duplicated id for contribution award")
		}
		if elem.Id >= contributionAwardCount {
			return fmt.Errorf("contribution award id should be lower or equal than the last id")
		}
		contributionAwardIdMap[elem.Id] = true
	}

	scopedEvidenceVouchIdMap := make(map[uint64]bool)
	scopedEvidenceVouchCount := gs.GetScopedEvidenceVouchCount()
	for _, elem := range gs.ScopedEvidenceVouchList {
		if _, ok := scopedEvidenceVouchIdMap[elem.Id]; ok {
			return fmt.Errorf("duplicated id for scoped evidence vouch")
		}
		if elem.Id >= scopedEvidenceVouchCount {
			return fmt.Errorf("scoped evidence vouch id should be lower or equal than the last id")
		}
		scopedEvidenceVouchIdMap[elem.Id] = true
	}

	if gs.EmissionState.MaxSupply == 0 {
		return fmt.Errorf("emission max_supply is required")
	}
	if gs.EmissionState.EpochLengthSeconds == 0 {
		return fmt.Errorf("emission epoch_length_seconds is required")
	}
	if gs.EmissionState.EpochRewardDivisor == 0 {
		return fmt.Errorf("emission epoch_reward_divisor is required")
	}
	if gs.EmissionState.CurrentEpoch == 0 {
		return fmt.Errorf("emission current_epoch must be >= 1")
	}
	if RemainingScheduled(gs.EmissionState) == 0 &&
		(gs.EmissionState.TotalMined > gs.EmissionState.MaxSupply ||
			gs.EmissionState.TotalBurnedUnclaimed > gs.EmissionState.MaxSupply-gs.EmissionState.TotalMined) {
		return fmt.Errorf("emission mined+burned exceeds max_supply")
	}
	if gs.EmissionState.EpochMined > gs.EmissionState.EpochScheduled {
		return fmt.Errorf("emission epoch_mined exceeds epoch_scheduled")
	}

	return gs.Params.Validate()
}
