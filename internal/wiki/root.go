package wiki

import "github.com/spf13/cobra"

var rootCmd = &cobra.Command{
	Use:   "wiki",
	Short: "Personal knowledge base CLI",
}

func init() {
	rootCmd.AddCommand(articleCmd)
	rootCmd.AddCommand(bookmarkGroupCmd)
	rootCmd.AddCommand(searchCmd)
	rootCmd.AddCommand(pushCmd)
}

func Execute() error {
	return rootCmd.Execute()
}
