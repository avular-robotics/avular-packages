package types

type Metadata struct {
	Name        string   `yaml:"name"`
	Version     string   `yaml:"version"`
	Owners      []string `yaml:"owners"`
	Description string   `yaml:"description,omitempty"`
}

// SpecDefaults provides project-level defaults that the CLI and
// application layer use when a value is not explicitly provided via
// flags or environment variables.  Embedding defaults in the product
// spec eliminates the need for a separate config file or repetitive
// CLI flags.
type SpecDefaults struct {
	TargetUbuntu string   `yaml:"target_ubuntu,omitempty"`
	Workspace    []string `yaml:"workspace,omitempty"`
	RepoIndex    string   `yaml:"repo_index,omitempty"`
	Output       string   `yaml:"output,omitempty"`
}

type ComposeRef struct {
	Name    string `yaml:"name"`
	Version string `yaml:"version"`
	Source  string `yaml:"source"`
	Path    string `yaml:"path"`

	// Profile holds an inline profile definition when Source is "inline".
	// This allows simple products to embed their packaging policy
	// directly without a separate profile file.
	Profile *InlineProfile `yaml:"profile,omitempty"`
}

// InlineProfile is a subset of Spec fields that can be embedded
// directly in a ComposeRef.  It carries the same semantics as a
// standalone profile spec but omits fields that are only meaningful
// at the product level (compose, publish, kind, metadata).
type InlineProfile struct {
	Inputs      Inputs                `yaml:"inputs"`
	Packaging   Packaging             `yaml:"packaging"`
	Resolutions []ResolutionDirective `yaml:"resolutions,omitempty"`
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
	Defaults    SpecDefaults          `yaml:"defaults,omitempty"`
	Compose     []ComposeRef          `yaml:"compose"`
	Inputs      Inputs                `yaml:"inputs"`
	Packaging   Packaging             `yaml:"packaging"`
	Resolutions []ResolutionDirective `yaml:"resolutions"`
	Publish     Publish               `yaml:"publish"`

	// Schema holds an optional inline schema mapping.  When present,
	// these mappings are loaded before any schema_files references,
	// giving file-based schemas higher precedence (they override
	// inline entries on a per-key basis).
	Schema *SchemaFile `yaml:"schema,omitempty"`
}
