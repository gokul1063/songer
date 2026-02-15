/*
Copyright © 2026 NAME HERE <EMAIL ADDRESS>
*/
package cmd

import (
	"fmt"
	"strings"

	"github.com/gokul1063/songer/internal"
	"github.com/spf13/cobra"
)

var searchCmd = &cobra.Command{
	Use:   "search",
	Short: "Used to search a song",
	Long: `This first searches the song in the specific folder localy
	Then if not found it downloads that songs from ytdlp`,
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Println("search called")
		var res bool = internal.IsFileExist(strings.Join(args, " "))
		fmt.Println(res)
	},
}

func init() {
	rootCmd.AddCommand(searchCmd)

}
