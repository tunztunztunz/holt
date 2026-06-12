package config

type Profile struct {
	Version      int        `yaml:"version"`
	SiteName     string     `yaml:"site_name"`
	WorktreesDir string     `yaml:"worktrees_dir"`
	Copy         []string   `yaml:"copy"`
	Link         []string   `yaml:"link"`
	Env          []EnvBlock `yaml:"env"`
	Port         *PortBlock `yaml:"port"`
	Setup        []string   `yaml:"setup"`
	Teardown     []string   `yaml:"teardown"`
	Guards       []string   `yaml:"guards"`
	Plugins      []string   `yaml:"plugins"`
}

type EnvBlock struct {
	File   string            `yaml:"file"`
	Set    map[string]string `yaml:"set"`
	Ensure map[string]string `yaml:"ensure"`
}

type PortStrategy string

const (
	PortHash PortStrategy = "hash"
	PortFree PortStrategy = "free"
)

type PortBlock struct {
	Range    [2]int       `yaml:"range"`
	Strategy PortStrategy `yaml:"strategy"` // "hash" | "free"
}
