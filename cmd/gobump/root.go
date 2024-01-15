package cmd

import (
	"fmt"
	"log"
	"os"
	"strings"

	"github.com/chainguard-dev/gobump/pkg/types"
	"github.com/chainguard-dev/gobump/pkg/update"
	"github.com/spf13/cobra"
	"sigs.k8s.io/release-utils/version"
)

type rootCLIFlags struct {
	packages  string
	modroot   string
	replaces  string
	goVersion string
	tidy      bool
}

var rootFlags rootCLIFlags

// rootCmd represents the base command when called without any subcommands
var rootCmd = &cobra.Command{
	Use:   "gobump",
	Short: "gobump cli",
	Args:  cobra.NoArgs,
	// Uncomment the following line if your bare application
	// has an action associated with it:
	Run: func(cmd *cobra.Command, args []string) {
		if rootFlags.packages == "" {
			log.Println("Usage: gobump -packages=<package@version>,...")
			os.Exit(1)
		}
		packages := strings.Split(rootFlags.packages, " ")
		pkgVersions := map[string]*types.Package{}
		for _, pkg := range packages {
			parts := strings.Split(pkg, "@")
			if len(parts) != 2 {
				fmt.Println("Usage: gobump -packages=<package@version>,...")
				os.Exit(1)
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
					fmt.Println("Usage: gobump -replaces=<oldpackage=newpackage@version>,...")
					os.Exit(1)
				}
				// extract the new package name and version
				rep := strings.Split(strings.TrimPrefix(replace, fmt.Sprintf("%s=", parts[0])), "@")
				if len(rep) != 2 {
					fmt.Println("Usage: gobump -replaces=<oldpackage=newpackage@version>,...")
					os.Exit(1)
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

		if _, err := update.DoUpdate(pkgVersions, rootFlags.modroot, rootFlags.tidy, rootFlags.goVersion); err != nil {
			fmt.Println("failed running update: ", err)
			os.Exit(1)
		}
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
	flagSet.StringVar(&rootFlags.goVersion, "go-version", "", "set the go-version for go-mod-tidy")
}
