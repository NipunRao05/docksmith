package builder

type BuildState struct {
	WorkingDir string
	Env        map[string]string
	Cmd        []string
	Layers     []string
	RootFS     string

	BaseImageDigest         string
	LastProducedLayerDigest string
	ProducedLayerCount      int
}
