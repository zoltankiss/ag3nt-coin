package types

import "fmt"

// DefaultGenesis returns the default genesis state
func DefaultGenesis() *GenesisState {
	return &GenesisState{
		Params:     DefaultParams(),
		AccountMap: []Account{}, VouchList: []Vouch{}, EscrowList: []Escrow{}, ContributionAwardList: []ContributionAward{}, ScopedEvidenceVouchList: []ScopedEvidenceVouch{}}
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

	return gs.Params.Validate()
}
