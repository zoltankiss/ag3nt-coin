package keeper

// isHexSHA256 reports whether s is a well-formed hex-encoded sha256 digest
// (64 hex chars). Every hash the chain accepts as a commitment must pass —
// a malformed "hash" is worse than none: it looks like evidence but no juror
// can ever re-check bytes against it.
func isHexSHA256(s string) bool {
	if len(s) != 64 {
		return false
	}
	for i := 0; i < len(s); i++ {
		c := s[i]
		if (c < '0' || c > '9') && (c < 'a' || c > 'f') && (c < 'A' || c > 'F') {
			return false
		}
	}
	return true
}
