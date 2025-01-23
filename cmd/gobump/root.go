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
	packages        string
	bumpFile        string
	modroot         string
	replaces        string
	goVersion       string
	tidy            bool
	skipInitialTidy bool
	showDiff        bool
	tidyCompat      string
}

var rootFlags rootCLIFlags

// rootCmd represents the base command when called without any subcommands.
var rootCmd = &cobra.Command{
	Use:          "gobump",
	Short:        "gobump cli",
	Args:         cobra.NoArgs,
	SilenceUsage: true,
	RunE: func(_ *cobra.Command, _ []string) error {
		if rootFlags.packages == "" && rootFlags.replaces == "" && rootFlags.bumpFile == "" {
			return fmt.Errorf("no packages or replaces provided. Use --packages or --replaces or --bump-file")
		}

		if rootFlags.packages != "" && rootFlags.bumpFile != "" {
			return fmt.Errorf("both --packages and --bump-file flags are provided. Use only one")
		}

		if rootFlags.replaces != "" && rootFlags.bumpFile != "" {
			return fmt.Errorf("both --replaces and --bump-file flags are provided. Use only one")
		}

		pkgVersions := map[string]*types.Package{}
		if rootFlags.bumpFile != "" {
			var err error
			pkgVersions, err = types.ParseFile(rootFlags.bumpFile)
			if err != nil {
				return fmt.Errorf("failed to parse bump file %q: %v", rootFlags.bumpFile, err)
			}
		} else {
			packages := strings.Fields(rootFlags.packages)
			for i, pkg := range packages {
				parts := strings.Split(pkg, "@")
				if len(parts) != 2 {
					return fmt.Errorf("invalid package format. Each package should be in the format <package@version>. Usage: gobump --packages=\"<package1@version> <package2@version> ...\"")
				}
				pkgVersions[parts[0]] = &types.Package{
					Name:    parts[0],
					Version: parts[1],
					Index:   i,
				}
			}

			if len(rootFlags.replaces) != 0 {
				replaces := strings.Fields(rootFlags.replaces)
				for i, replace := range replaces {
					parts := strings.Split(replace, "=")
					if len(parts) != 2 {
						return fmt.Errorf("invalid replace format. Each replace should be in the format <oldpackage=newpackage@version>. Usage: gobump -replaces=\"<oldpackage=newpackage@version> ...\"")
					}
					// extract the new package name and version
					rep := strings.Split(strings.TrimPrefix(replace, fmt.Sprintf("%s=", parts[0])), "@")
					if len(rep) != 2 {
						return fmt.Errorf("invalid replace format. Each replace should be in the format <oldpackage=newpackage@version>. Usage: gobump -replaces=\"<oldpackage=newpackage@version> ...\"")
					}
					// Merge/Add the packages to replace reusing the initial list of packages
					pkgVersions[rep[0]] = &types.Package{
						OldName: parts[0],
						Name:    rep[0],
						Version: rep[1],
						Replace: true,
						Index:   i,
					}
				}
			}
		}

		if _, err := update.DoUpdate(pkgVersions, &types.Config{Modroot: rootFlags.modroot, Tidy: rootFlags.tidy, GoVersion: rootFlags.goVersion, ShowDiff: rootFlags.showDiff, TidyCompat: rootFlags.tidyCompat, TidySkipInitial: rootFlags.skipInitialTidy}); err != nil {
			return fmt.Errorf("failed to run update. Error: %v", err)
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
	flagSet.StringVar(&rootFlags.bumpFile, "bump-file", "", "Filename containing the list of packages to update / replace")
	flagSet.StringVar(&rootFlags.modroot, "modroot", "", "path to the go.mod root")
	flagSet.StringVar(&rootFlags.replaces, "replaces", "", "A space-separated list of packages to replace")
	flagSet.BoolVar(&rootFlags.tidy, "tidy", false, "Run 'go mod tidy' command")
	flagSet.BoolVar(&rootFlags.skipInitialTidy, "skip-initial-tidy", false, "Skip running 'go mod tidy' command before updating the go.mod file")
	flagSet.BoolVar(&rootFlags.showDiff, "show-diff", false, "Show the difference between the original and 'go.mod' files")
	flagSet.StringVar(&rootFlags.goVersion, "go-version", "", "set the go-version for go-mod-tidy")
	flagSet.StringVar(&rootFlags.tidyCompat, "compat", "", "set the go version for which the tidied go.mod and go.sum files should be compatible")
}
