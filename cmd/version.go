package cmd

import (
	"encoding/json"
	"fmt"

	"github.com/spf13/cobra"
)

var versionJSON bool

var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "Print version, commit, and build date",
	RunE: func(cmd *cobra.Command, args []string) error {
		if versionJSON {
			out, err := json.Marshal(map[string]string{
				"version": buildVersion,
				"commit":  buildCommit,
				"date":    buildDate,
			})
			if err != nil {
				return err
			}
			fmt.Println(string(out))
			return nil
		}
		fmt.Printf("dev-cli %s\ncommit: %s\nbuilt:  %s\n", buildVersion, buildCommit, buildDate)
		return nil
	},
}

func init() {
	versionCmd.Flags().BoolVar(&versionJSON, "json", false, "Output as JSON")
	rootCmd.AddCommand(versionCmd)
}
