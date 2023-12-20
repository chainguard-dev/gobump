package types

type Package struct {
	OldName string
	Name    string
	Version string
	Replace bool
	Require bool
}
