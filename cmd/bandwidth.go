package cmd

import (
	bd "bandwidth/elasticollect"
	"github.com/spf13/cobra"
)

// serverCmd represents the server command
var bdCmd = &cobra.Command{
	Use:   "start",
	Short: "start calculating domain's bandwidth",
	Long:  `start calculating domain's bandwidth`,
	Run: func(cmd *cobra.Command, args []string) {
		mergeViperServer()
		bd.Run()
	},
}

func init() {
	rootCmd.AddCommand(bdCmd)
}
