package cmd

import (
	"fmt"

	"github.com/gokul1063/songer/internal"
	"github.com/spf13/cobra"
)

// testCmd represents the test command
var testCmd = &cobra.Command{
	Use:   "test",
	Short: "A brief description of your command",
	Long: `A longer description that spans multiple lines and likely contains examples
and usage of using your command. For example:

Cobra is a CLI library for Go that empowers applications.
This application is a tool to generate the needed files
to quickly create a Cobra application.`,
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Println("test called")
		result1 := internal.ProcessSongName("test1")
		result2 := internal.ProcessSongName("test2 test2 test2")
		fmt.Println(result1)
		fmt.Println(result2)

		result3 := internal.TisFileExist(result1)
		fmt.Println(result3)

	},
}

func init() {
	rootCmd.AddCommand(testCmd)

}
