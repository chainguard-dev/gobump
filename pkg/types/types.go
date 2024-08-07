package types

type Package struct {
	OldName string `json:"oldName,omitempty" yaml:"oldName,omitempty"`
	Name    string `json:"name,omitempty" yaml:"name,omitempty"`
	Version string `json:"version,omitempty" yaml:"version,omitempty"`
	Replace bool   `json:"replace,omitempty" yaml:"replace,omitempty"`
	Require bool   `json:"require,omitempty" yaml:"require,omitempty"`
	Index   int    `json:"index,omitempty" yaml:"index,omitempty"`
}

type Config struct {
	Modroot         string
	GoVersion       string
	ShowDiff        bool
	Tidy            bool
	TidyCompat      string
	TidySkipInitial bool
}

// Used to marshal from yaml/json file to get the list of packages
type PackageList struct {
	Packages []Package `json:"packages" yaml:"packages"`
}
