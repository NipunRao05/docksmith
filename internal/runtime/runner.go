package runtime

import "fmt"

type Runner struct{}

func NewRunner() *Runner {
	return &Runner{}
}

func (r *Runner) Run(image string, cmd []string) error {
	fmt.Println("Running container", image)

	fmt.Println("Container exited")
	return nil
}
