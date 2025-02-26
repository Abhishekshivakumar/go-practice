package main

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

var rootCmd = &cobra.Command{
	Use:   "dive analyse.tar",
	Short: "Docker Image Visualizer & Explorer",
	Long: `This tool provides a way to discover and explore the contents of a docker image. Additionally the tool estimates
the amount of wasted space and identifies the offending files from the image.`,
	Args: cobra.MaximumNArgs(1),
	Run:  doAnalyzeCmd,
}

func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Println("Error: while running cmd")
		os.Exit(0)
	}

}
