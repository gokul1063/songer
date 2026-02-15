/*
Copyright © 2026 NAME HERE <EMAIL ADDRESS>
*/
package cmd

import (
	"fmt"

	"github.com/gokul1063/songer/services"
	"github.com/spf13/cobra"
)

// playCmd represents the play command
var playCmd = &cobra.Command{
	Use:   "play",
	Short: "Plays song based on the availabily ",
	Long: `This command uses mpv as server 
	if the songs exists locally it uses that 
	else it download the songs form ytdlp and plays it 
	
	`,

	Run: func(cmd *cobra.Command, args []string) {
		fmt.Println("play called")
		fmt.Println(args[0])
		services.PlaySong(args[0])

	},
}

func init() {
	rootCmd.AddCommand(playCmd)
}
