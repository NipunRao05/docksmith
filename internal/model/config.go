package model

type Config struct {
	Env        []string `json:"Env"`
	Cmd        []string `json:"Cmd"`
	WorkingDir string   `json:"WorkingDir"`
}
