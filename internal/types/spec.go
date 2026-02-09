package types

type Metadata struct {
	Name        string   `yaml:"name"`
	Version     string   `yaml:"version"`
	Owners      []string `yaml:"owners"`
	Description string   `yaml:"description,omitempty"`
}

type ComposeRef struct {
	Name    string `yaml:"name"`
	Version string `yaml:"version"`
	Source  string `yaml:"source"`
	Path    string `yaml:"path"`
}

type PackageXMLInput struct {
	Tags       []string `yaml:"tags"`
	Prefix     string   `yaml:"prefix,omitempty"`
	Enabled    bool     `yaml:"enabled"`
	IncludeSrc bool     `yaml:"include_src,omitempty"`

	// SchemaFiles lists paths to schema.yaml files that map abstract
	// ROS dependency keys to concrete typed packages.  Files are loaded
	// in order; later files override earlier ones per key.
	// When non-empty, standard ROS tags (<depend>, <exec_depend>, etc.)
	// are parsed and resolved through the schema.
	SchemaFiles []string `yaml:"schema_files,omitempty"`
}

type ManualInputs struct {
	Apt    []string `yaml:"apt"`
	Python []string `yaml:"python"`
}

type Inputs struct {
	PackageXML PackageXMLInput `yaml:"package_xml"`
	Manual     ManualInputs    `yaml:"manual"`
}

type PackagingGroup struct {
	Name    string        `yaml:"name"`
	Mode    PackagingMode `yaml:"mode"`
	Scope   string        `yaml:"scope"`
	Matches []string      `yaml:"matches"`
	Targets []string      `yaml:"targets"`
	Pins    []string      `yaml:"pins,omitempty"`
}

type Packaging struct {
	Groups []PackagingGroup `yaml:"groups"`
}

type ResolutionDirective struct {
	Dependency string `yaml:"dependency"`
	Action     string `yaml:"action"`
	Value      string `yaml:"value,omitempty"`
	Reason     string `yaml:"reason"`
	Owner      string `yaml:"owner"`
	ExpiresAt  string `yaml:"expires_at,omitempty"`
}

type PublishRepository struct {
	Name           string `yaml:"name"`
	Channel        string `yaml:"channel"`
	SnapshotPrefix string `yaml:"snapshot_prefix"`
	SigningKey     string `yaml:"signing_key"`
}

type Publish struct {
	Repository PublishRepository `yaml:"repository"`
}

type Spec struct {
	APIVersion  string                `yaml:"api_version"`
	Kind        SpecKind              `yaml:"kind"`
	Metadata    Metadata              `yaml:"metadata"`
	Compose     []ComposeRef          `yaml:"compose"`
	Inputs      Inputs                `yaml:"inputs"`
	Packaging   Packaging             `yaml:"packaging"`
	Resolutions []ResolutionDirective `yaml:"resolutions"`
	Publish     Publish               `yaml:"publish"`
}
