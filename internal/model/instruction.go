package model

type Instruction struct {
	Type string
	Args []string
	Raw string
	Line int
}