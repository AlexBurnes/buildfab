package container

// ContainerConfig represents container configuration
type ContainerConfig struct {
	Engine     string            `yaml:"engine"`
	Image      ContainerImage    `yaml:"image"`
	Workdir    string            `yaml:"workdir"`
	CPU        int               `yaml:"cpu"`         // Simple CPU count: 2 -> --cpus 2.0 --cpuset-cpus "0,1"
	Memory     string            `yaml:"memory"`
	Mounts     []ContainerMount  `yaml:"mounts"`
	Artifacts  ContainerArtifacts `yaml:"artifacts"`
	Env        map[string]string `yaml:"env"`
	EnvFile    string            `yaml:"env_file"`
	User       string            `yaml:"user"`
	Network    string            `yaml:"network"`
	Cache      map[string]string `yaml:"cache"`
	RunStage   string            `yaml:"run_stage"`
	RunAction  string            `yaml:"run_action"`
	Run        string            `yaml:"run"`
}

// ContainerImage represents container image configuration
type ContainerImage struct {
	From  string           `yaml:"from"`
	Build *ContainerBuild  `yaml:"build"`
	Slim  *ContainerSlim   `yaml:"slim"`
}

// ContainerBuild represents container build configuration
type ContainerBuild struct {
	Dockerfile string            `yaml:"dockerfile"`
	Context    string            `yaml:"context"`
	Args       map[string]string `yaml:"args"`
	Tags       []string          `yaml:"tags"`
	Network    string            `yaml:"network"`
	Progress   string            `yaml:"progress"`
}

// ContainerSlim represents container slim configuration
type ContainerSlim struct {
	Target    string   `yaml:"target"`
	Tags      []string `yaml:"tags"`
	Network   string   `yaml:"network"`
	HttpProbe bool     `yaml:"http_probe"`
	Exec      string   `yaml:"exec"`
}

// ContainerMount represents container mount configuration
type ContainerMount struct {
	Type   string `yaml:"type"`
	Source string `yaml:"source"`
	Target string `yaml:"target"`
	RO     bool   `yaml:"ro"`
}

// ContainerArtifacts represents container artifact collection configuration
type ContainerArtifacts struct {
	Output string   `yaml:"output"`
	Path   []string `yaml:"path"`
}

// ContainerResult represents the result of container execution
type ContainerResult struct {
	ContainerID string
	ExitCode    int
	Output      string
	Error       string
	Artifacts   []string
}
