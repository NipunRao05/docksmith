package cli

import (
	"errors"
	"docksmith/internal/runtime"
)

func HandleRun(args []string) error {
	if len(args) < 1 {
		return errors.New("usage: docksmith run [-e KEY=VALUE] <name:tag> [cmd]")
	}
	envs := []string{}
	image := ""
	var cmdOverride []string

	for i := 0; i < len(args); i++ {
		if args[i] == "-e" && i+1 < len(args) {
			envs = append(envs, args[i+1])
			i++
		} else if image == "" {
			image = args[i]
		} else {
			// everything after image name is cmd override
			cmdOverride = args[i:]
			break
		}
	}
	return runtime.Run(image, envs, cmdOverride)
}
