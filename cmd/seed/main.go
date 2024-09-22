package main

import "github.com/idea456/commuter-api/pkg/utils"

func main() {
	g := utils.NewSeedGenerator()
	g.GenerateStops()
	g.GenerateEdges()
	g.GenerateProperties()
	g.GenerateNearestStops(600)

}
