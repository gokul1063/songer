/*
Copyright © 2026 NAME HERE <EMAIL ADDRESS>
*/
package cmd

import (
	"fmt"

	"github.com/gokul1063/songer/internal"
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

		if offline {
			fmt.Println("offline tag called ")
			path := internal.SearchSong(args[0])
			fmt.Printf("The path returned : %s ", path)
			services.PlaySongTest1(path)
			return
		}

		if len(args) == 0 {
			services.DisplaySong()
			return
		}

		fmt.Println("play called")
		fmt.Println(args[0])
		services.PlaySong(args[0])

	},
}

var offline bool

func init() {
	rootCmd.AddCommand(playCmd)
	playCmd.Flags().BoolVarP(&offline, "offline", "o", false, "Play offline song test1")

}
