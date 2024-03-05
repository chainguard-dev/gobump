package types

type Package struct {
	OldName string
	Name    string
	Version string
	Replace bool
	Require bool
	Index   int
}

type Config struct {
	Modroot    string
	GoVersion  string
	ShowDiff   bool
	Tidy       bool
	TidyCompat string
}
