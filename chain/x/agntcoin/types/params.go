package types

import "fmt"

// NewParams creates a new Params instance. The optional anchors become the
// reputation trust roots (see Params.Anchors).
func NewParams(anchors ...string) Params {
	return Params{Anchors: anchors}
}

// DefaultParams returns a default set of parameters. Anchors are empty by
// default; a chain seeds its founder anchor in the genesis params.
func DefaultParams() Params {
	return NewParams()
}

// Validate validates the set of params.
func (p Params) Validate() error {
	seen := make(map[string]struct{}, len(p.Anchors))
	for _, a := range p.Anchors {
		if a == "" {
			return fmt.Errorf("anchor address must not be empty")
		}
		if _, dup := seen[a]; dup {
			return fmt.Errorf("duplicate anchor address: %s", a)
		}
		seen[a] = struct{}{}
	}
	return nil
}
