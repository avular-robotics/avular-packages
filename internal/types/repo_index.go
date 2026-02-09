package types

type RepoIndexFile struct {
	Apt         map[string][]string            `yaml:"apt"`
	AptPackages map[string][]AptPackageVersion `yaml:"apt_packages,omitempty"`
	Pip         map[string][]string            `yaml:"pip"`
}

type AptPackageVersion struct {
	Version    string   `yaml:"version"`
	Depends    []string `yaml:"depends,omitempty"`
	PreDepends []string `yaml:"pre_depends,omitempty"`
	Provides   []string `yaml:"provides,omitempty"`
}
