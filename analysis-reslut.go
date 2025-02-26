package main

type Analyzer interface {
	Analyze() (*AnalysisResult, error)
}

type AnalysisResult struct {
	Layers            []*Layer
	RefTrees          []*FileTree
	Efficiency        float64
	SizeBytes         uint64
	UserSizeByes      uint64  // this is all bytes except for the base image
	WastedUserPercent float64 // = wasted-bytes/user-size-bytes
	WastedBytes       uint64
	Inefficiencies    EfficiencySlice
}
