package cli

import (
	"errors"
	"fmt"
	"docksmith/internal/builder"
)

func HandleBuild(args []string) error {
	if len(args) < 3 {
		return errors.New("Usage: docksmith build -t <name:tag> <context>")
	}

	noCache := false
	filteredArgs := []string{}
	for _, a := range args {
		if a == "--no-cache" {
			noCache = true
		} else {
			filteredArgs = append(filteredArgs, a)
		}
	}

	if filteredArgs[0] != "-t" {
		return errors.New("missing -t flag")
	}
	tag := filteredArgs[1]
	context := filteredArgs[2]

	fmt.Println("Building image:", tag)
	engine := builder.NewEngine()
	engine.NoCache = noCache
	return engine.Build(tag, context)
}
