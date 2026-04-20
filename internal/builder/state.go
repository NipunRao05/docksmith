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
	
	// Track instruction text for each layer (digest -> instruction text)
	LayerCreatedBy map[string]string
	// Track file hashes for delta detection
	PreviousFileHashes map[string]string
}
