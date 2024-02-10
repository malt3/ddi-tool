package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

var rootCmd = &cobra.Command{
	Use:   "ddi-tool",
	Short: "ddi-tool is a swiss army knife for discoverable disk images",
	Long: `A simple CLI tool for extracting information from
				  and patching discoverable disk images.
				  (See the spec at https://uapi-group.org/specifications/specs/discoverable_disk_image/)`,
}

func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
