package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

var rootCmd = &cobra.Command{
	Short: "Create an estimate of the cloud carbon footprint we have",
	Long:  `Calculate our carbon footprint based on AWS usage reports.`,
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Println("Here is Run.")
	},
}

func init() {
	rootCmd.AddCommand(analyseCmd)
}

func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
