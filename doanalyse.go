package main

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

func doAnalyzeCmd(cmd *cobra.Command, args []string) {
	if len(args) == 0 {
		fmt.Println("No tar file path argument given")
		os.Exit(0)
	}
	tarPath := args[0]
	if _, err := os.Stat(tarPath); os.IsNotExist(err) {
		fmt.Println("Please provide a tar file path like:  \"/analyse/analyse.tar\"")
		os.Exit(0)
	}
	fmt.Println("near Run call")

	Run(Options{
		Ci:     false,
		Path:   tarPath,
		Source: SourceDockerArchive,
	})

}
