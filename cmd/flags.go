package cmd

import "github.com/antonito/gfile/internal/utils"

type globalFlags struct {
	stunServer string
}

func (flags *globalFlags) ResolvedSTUN() (string, error) {
	if flags.stunServer == "" {
		return "", nil
	}
	if err := utils.ParseSTUN(flags.stunServer); err != nil {
		return "", err
	}

	return flags.stunServer, nil
}
