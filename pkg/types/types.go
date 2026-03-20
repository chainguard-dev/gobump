package types //nolint:revive

// Package represents a Go module package to be updated or replaced.
type Package struct {
	OldName string `json:"oldName,omitempty" yaml:"oldName,omitempty"`
	Name    string `json:"name,omitempty" yaml:"name,omitempty"`
	Version string `json:"version,omitempty" yaml:"version,omitempty"`
	Replace bool   `json:"replace,omitempty" yaml:"replace,omitempty"`
	Require bool   `json:"require,omitempty" yaml:"require,omitempty"`
	Index   int    `json:"index,omitempty" yaml:"index,omitempty"`
	// Force allows downgrading a package to a version older than the current one.
	// By default, downgrade attempts are skipped with a warning.
	Force bool `json:"force,omitempty" yaml:"force,omitempty"`
}

// Config contains configuration options for the update process.
type Config struct {
	Modroot         string
	GoVersion       string
	ShowDiff        bool
	Tidy            bool
	TidyCompat      string
	TidySkipInitial bool
	ForceWork       bool
}

// PackageList is used to marshal from yaml/json file to get the list of packages.
type PackageList struct {
	Packages []Package `json:"packages" yaml:"packages"`
}
