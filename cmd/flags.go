package cmd

import "github.com/antonito/gfile/internal/utils"

type globalFlags struct {
	stunServers []string
	mdns        bool
}

// ResolvedSTUNs validates each --stun entry and returns the filtered list.
// Empty entries (e.g. from --stun="") are dropped so users can opt out of
// STUN entirely and rely on host/mDNS candidates only.
func (flags *globalFlags) ResolvedSTUNs() ([]string, error) {
	out := make([]string, 0, len(flags.stunServers))
	for _, s := range flags.stunServers {
		if s == "" {
			continue
		}
		if err := utils.ParseSTUN(s); err != nil {
			return nil, err
		}
		out = append(out, s)
	}
	return out, nil
}
