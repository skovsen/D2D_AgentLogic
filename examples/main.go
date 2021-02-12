package main

import (
	"fmt"
	"io/ioutil"
	"log"

	"github.com/paulmach/orb/geojson"
	logic "github.com/skovsen/D2D_AgentLogic"
)

func main() {

	// Create a simple mission with a description and whatever parameters we think of
	// in this case pi and eulers number
	mission := logic.Mission{
		Description: "Kill all Humans",
		MissionType: "Search and Destroy",
		AreaLink:    "http://kill.all.humans",
		MetaNeeded: logic.MetaNeeded{
			MovementAxis: 3,
		},
		Goal: logic.Goal{
			Do:  "Starting thing",
			End: "Jobs done",
		},
	}

	// Set the bounding box of the mission using a Polygon
	// Examples are both with relative coordinates and geospacial coordinates.
	// Check out the wiki for examples: https://en.wikipedia.org/wiki/GeoJSON

	// mission.Geometry = orb.Polygon{
	// 	{
	// 		{30, 10},
	// 		{40, 40},
	// 		{50, 50},
	// 		{20, 40},
	// 		{30, 10},
	// 	},
	// }

	// mission.Geometry = orb.Polygon{
	// 	{
	// 		{0, 100},
	// 		{100, 100},
	// 		{100, 0},
	// 		{0, 0},
	// 		{0, 100},
	// 	},
	// }

	// Load geojson into geometry, complex example
	mission.LoadFeatures("./testdata/aarhus.geojson")
	// fmt.Println(mission)

	// Print bounds and mission centre
	p, a := mission.MissionArea()
	fmt.Printf("Total Mission Area: %v %v \n", p, a)

	// Create three agents, and the agent array
	agent1 := logic.AgentHolder{
		Agent: logic.Agent{UUID: "ALL"},
	}
	agent2 := logic.AgentHolder{
		Agent: logic.Agent{UUID: "YOUR"},
	}
	agent3 := logic.AgentHolder{
		Agent: logic.Agent{UUID: "BASE"},
	}

	// agent2 := logic.Agent{UUID: "YOUR"}
	// agent3 := logic.Agent{UUID: "BASE"}
	// agents := []logic.Agent{agent1, agent2, agent3}
	agents := map[string]logic.AgentHolder{"ALL": agent1, "YOUR": agent2, "BASE": agent3}

	// Plan the mission with the available agents
	agentMap, err := logic.ReplanMission(mission, agents, 18)
	if err != nil {
		log.Fatal(err)
	}

	// Save a path and mission area to file
	// for each drone. These files can be opened in Q-GIS
	for aID, m := range agentMap {
		var ah = logic.AgentHolder{}
		for id, agent := range agents {
			if id == aID {
				ah = agent
				break
			}
		}

		fc := geojson.NewFeatureCollection()
		fc.Append(geojson.NewFeature(m.Geometry))
		rawJSON, _ := fc.MarshalJSON()
		_ = ioutil.WriteFile(fmt.Sprintf("agentarea-%v.json", ah.Agent.UUID), rawJSON, 0644)

		path, err := m.GeneratePath(ah, 18)
		if err != nil {
			log.Fatal(err)
		}

		fc = geojson.NewFeatureCollection()
		fc.Append(geojson.NewFeature(path))
		rawJSON, _ = fc.MarshalJSON()
		_ = ioutil.WriteFile(fmt.Sprintf("agentpath-%v.json", ah.Agent.UUID), rawJSON, 0644)
	}
}
