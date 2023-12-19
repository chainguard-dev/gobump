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
	packages string
	modroot  string
	replaces string
	tidy     bool
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
		packages := strings.Split(rootFlags.packages, ",")
		pkgVersions := []*types.Package{}
		for _, pkg := range packages {
			parts := strings.Split(pkg, "@")
			if len(parts) != 2 {
				fmt.Println("Usage: gobump -packages=<package@version>,...")
				os.Exit(1)
			}
			pkgVersions = append(pkgVersions, &types.Package{
				Name:    parts[0],
				Version: parts[1],
			})
		}

		var replaces []string
		if len(rootFlags.replaces) != 0 {
			replaces = strings.Split(rootFlags.replaces, " ")
		}

		if _, err := update.DoUpdate(pkgVersions, replaces, rootFlags.modroot); err != nil {
			fmt.Println("Error running update: ", err)
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
}
