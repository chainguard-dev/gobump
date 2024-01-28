package cmd

import (
	"fmt"
	"strings"

	"github.com/chainguard-dev/gobump/pkg/types"
	"github.com/chainguard-dev/gobump/pkg/update"
	"github.com/spf13/cobra"
	"sigs.k8s.io/release-utils/version"
)

type rootCLIFlags struct {
	packages   string
	modroot    string
	replaces   string
	goVersion  string
	tidy       bool
	showDiff   bool
	tidyCompat string
}

var rootFlags rootCLIFlags

// rootCmd represents the base command when called without any subcommands
var rootCmd = &cobra.Command{
	Use:   "gobump",
	Short: "gobump cli",
	Args:  cobra.NoArgs,
	// Uncomment the following line if your bare application
	// has an action associated with it:
	RunE: func(cmd *cobra.Command, args []string) error {
		if rootFlags.packages == "" && rootFlags.replaces == "" {
			return fmt.Errorf("Error: No packages or replaces provided. Usage: gobump --packages=\"<package1@version> <package2@version> ...\" --replaces=\"<package3=package4@version> ...\"")
		}
		packages := strings.Split(rootFlags.packages, " ")
		pkgVersions := map[string]*types.Package{}
		for _, pkg := range packages {
			parts := strings.Split(pkg, "@")
			if len(parts) != 2 {
				return fmt.Errorf("Error: Invalid package format. Each package should be in the format <package@version>. Usage: gobump --packages=\"<package1@version> <package2@version> ...\"")
			}
			pkgVersions[parts[0]] = &types.Package{
				Name:    parts[0],
				Version: parts[1],
			}
		}

		var replaces []string
		if len(rootFlags.replaces) != 0 {
			replaces = strings.Split(rootFlags.replaces, " ")
			for _, replace := range replaces {
				parts := strings.Split(replace, "=")
				if len(parts) != 2 {
					return fmt.Errorf("Error: Invalid replace format. Each replace should be in the format <oldpackage=newpackage@version>. Usage: gobump -replaces=\"<oldpackage=newpackage@version> ...\"")
				}
				// extract the new package name and version
				rep := strings.Split(strings.TrimPrefix(replace, fmt.Sprintf("%s=", parts[0])), "@")
				if len(rep) != 2 {
					return fmt.Errorf("Error: Invalid replace format. Each replace should be in the format <oldpackage=newpackage@version>. Usage: gobump -replaces=\"<oldpackage=newpackage@version> ...\"")
				}
				// Merge/Add the packages to replace reusing the initial list of packages
				pkgVersions[rep[0]] = &types.Package{
					OldName: parts[0],
					Name:    rep[0],
					Version: rep[1],
					Replace: true,
				}
			}
		}

		if _, err := update.DoUpdate(pkgVersions, &types.Config{Modroot: rootFlags.modroot, Tidy: rootFlags.tidy, GoVersion: rootFlags.goVersion, ShowDiff: rootFlags.showDiff, TidyCompat: rootFlags.tidyCompat}); err != nil {
			return fmt.Errorf("Failed to running update. Error: %v", err)
		}
		return nil
	},
}

func RootCmd() *cobra.Command {
	return rootCmd
}

func init() {
	rootCmd.AddCommand(version.WithFont("starwars"))

	rootCmd.DisableAutoGenTag = true

	flagSet := rootCmd.Flags()
	flagSet.StringVar(&rootFlags.packages, "packages", "", "A space-separated list of packages to update")
	flagSet.StringVar(&rootFlags.modroot, "modroot", "", "path to the go.mod root")
	flagSet.StringVar(&rootFlags.replaces, "replaces", "", "A space-separated list of packages to replace")
	flagSet.BoolVar(&rootFlags.tidy, "tidy", false, "Run 'go mod tidy' command")
	flagSet.BoolVar(&rootFlags.showDiff, "show-diff", false, "Show the difference between the original and 'go.mod' files")
	flagSet.StringVar(&rootFlags.goVersion, "go-version", "", "set the go-version for go-mod-tidy")
	flagSet.StringVar(&rootFlags.tidyCompat, "compat", "", "set the go version for which the tidied go.mod and go.sum files should be compatible")
}
