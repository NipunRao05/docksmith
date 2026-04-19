package cli

import (
	"errors"
)

func HandleCommand(args []string) error {

	if len(args) < 2 {
		return errors.New("no command provided")
	}

	command := args[1]

	switch command {

	case "build":
		return HandleBuild(args[2:])
	case "run":
		return HandleRun(args[2:])
	case "images":
		return HandleImages()
	case "import":
		return HandleImport(args[2:])
	case "rmi":
		return HandleRMI(args[2:])
	default:
		return errors.New("Unknown command" + command)
	}
}
