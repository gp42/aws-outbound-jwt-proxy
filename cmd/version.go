package cmd

import (
	"fmt"

	"github.com/gp42/aws-outbound-jwt-proxy/internal/version"
	"github.com/spf13/cobra"
)

var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "Print the build version and exit",
	Args:  cobra.NoArgs,
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Fprintln(cmd.OutOrStdout(), version.String())
	},
}

func init() {
	rootCmd.AddCommand(versionCmd)
	rootCmd.Version = version.String()
	rootCmd.SetVersionTemplate("{{.Version}}\n")
}
