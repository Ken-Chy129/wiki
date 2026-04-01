package wiki

import "github.com/spf13/cobra"

var rootCmd = &cobra.Command{
	Use:   "wiki",
	Short: "Personal knowledge base CLI",
}

func init() {
	rootCmd.AddCommand(draftCmd)
	rootCmd.AddCommand(editCmd)
	rootCmd.AddCommand(bookmarkCmd)
	rootCmd.AddCommand(previewCmd)
	rootCmd.AddCommand(pushCmd)
	rootCmd.AddCommand(listCmd)
	rootCmd.AddCommand(searchCmd)
}

func Execute() error {
	return rootCmd.Execute()
}
