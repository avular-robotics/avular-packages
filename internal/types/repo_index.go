package types

type RepoIndexFile struct {
	Apt map[string][]string `yaml:"apt"`
	Pip map[string][]string `yaml:"pip"`
}
