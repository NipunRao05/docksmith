package main

import (
	"fmt"
	"os"
	"docksmith/internal/cli"
	"docksmith/internal/runtime"
)

func main() {
	// Internal chroot re-exec handler
	if len(os.Args) > 1 && os.Args[1] == "__chroot__" {
		if err := runtime.RunChroot(os.Args[2:]); err != nil {
			fmt.Fprintln(os.Stderr, "chroot error:", err)
			os.Exit(1)
		}
		os.Exit(0)
	}

	if len(os.Args) < 2 {
		fmt.Println("Usage: docksmith <command>")
		os.Exit(1)
	}

	err := cli.HandleCommand(os.Args)
	if err != nil {
		fmt.Println("Error:", err)
		os.Exit(1)
	}
}
