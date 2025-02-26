package main

import (
	"fmt"
)

type Options struct {
	Ci     bool
	Source ImageSource
	Path   string
}

func Run(options Options) {
	resolver, err := GetImageResolver(options.Source)
	if err != nil {
		fmt.Println("err-GetImageResolver", err)
	}
	img, err := resolver.Fetch(options.Path)

	if err != nil {
		fmt.Println("Errrr: resolver error")
	}
	analyse, err := img.Analyze()
	if err != nil {
		fmt.Println("Error: after analysis")
	}
	fmt.Println("Layers", analyse.Layers)
	fmt.Println("RefTrees", analyse.RefTrees)
	// for index, ref := range analyse.RefTrees {
	// 	fmt.Println(index, ref.String(false))
	// }
	fmt.Println("Efficiency", analyse.Efficiency)
	fmt.Println("SizeBytes", analyse.SizeBytes)
	fmt.Println("SizeBytes", analyse.SizeBytes)
	fmt.Println("WastedUserPercent", analyse.WastedUserPercent)
	fmt.Println("WastedBytes", analyse.WastedBytes)
	// fmt.Println("Inefficiencies", analyse.Inefficiencies)
	for index, node := range analyse.Inefficiencies {
		fmt.Println(index, node.minDiscoveredSize, node.CumulativeSize, node.Nodes, node.Path)
	}

}
