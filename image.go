package main

type Image struct {
	Trees  []*FileTree
	Layers []*Layer
}

func (img *Image) Analyze() (*AnalysisResult, error) {
	efficiency, inefficiencies := Efficiency(img.Trees)
	var sizeBytes, userSizeBytes uint64

	for i, v := range img.Layers {
		sizeBytes += v.Size
		if i != 0 {
			userSizeBytes += v.Size
		}
	}

	var wastedBytes uint64
	for _, file := range inefficiencies {
		wastedBytes += uint64(file.CumulativeSize)
	}

	return &AnalysisResult{
		Layers:            img.Layers,
		RefTrees:          img.Trees,
		Efficiency:        efficiency,
		UserSizeByes:      userSizeBytes,
		SizeBytes:         sizeBytes,
		WastedBytes:       wastedBytes,
		WastedUserPercent: float64(wastedBytes) / float64(userSizeBytes),
		Inefficiencies:    inefficiencies,
	}, nil
}
