package agentlogic

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"

	orb "github.com/paulmach/orb"
	"github.com/paulmach/orb/geo"
	"github.com/paulmach/orb/geojson"
	"github.com/paulmach/orb/maptile"
	"github.com/paulmach/orb/maptile/tilecover"
	"github.com/paulmach/orb/planar"
)

// Mission is the main datatype
type Mission struct {
	Description   string
	MissionType   MissionType
	AreaLink      string
	MetaNeeded    MetaNeeded
	Goal          Goal
	Geometry      orb.Geometry
	SwarmGeometry orb.Geometry
}

// MetaNeeded is the needed metadata
type MetaNeeded struct {
	MovementAxis   int
	SwarmSW        []string
	OnboardHW      []string
	DataCollection string
}

// Goal is the beginning and end
type Goal struct {
	Do      string
	End     string
	Endgame string
	Reached bool
}

// MissionType is a String
type MissionType string

// The types of mission abailable
const (
	Find MissionType = "find"
	// Surveil             = "surveil"
	// Measure             = "measure"
	// Other               = "other"
)

// ReplanMission will give you a new plan split betweena all agents
func ReplanMission(m Mission, agentholders map[string]AgentHolder, zoom maptile.Zoom) (map[string]Mission, error) {
	newMissions := make(map[string]Mission)
	unassignedMissions := []Mission{}
	unassignedAgents := []AgentHolder{}

	v, ok := m.Geometry.(orb.Polygon)
	if ok == false {
		log.Fatal("Could not cast geometry to polygon")
	}

	// Convert the polygon to tiles
	tiles := tilecover.Geometry(v, zoom)
	sortedTileSlice := []maptile.Tile{}

	for key := range tiles {
		sortedTileSlice = append(sortedTileSlice, key)
	}

	// Calculate number of tiles pr. agent
	// declaring them to make code easier to read
	numTiles := len(sortedTileSlice)
	agentCount := len(agentholders)
	eachAgent := numTiles / agentCount

	// Sort by x,y
	sortedTileSlice = sortTiles(sortedTileSlice)
	tmpTiles := []maptile.Tile{}
	i := 0
	for _, ah := range agentholders {
		unassignedAgents = append(unassignedAgents, ah)

		start := eachAgent * i
		stop := eachAgent*i + eachAgent

		if i == agentCount-1 {
			tmpTiles = sortedTileSlice[start:]
		} else {
			tmpTiles = sortedTileSlice[start:stop]
		}

		// Reduce to a single min and max point along x axis
		reducedSortedTileSlice := []maptile.Tile{}
		previousTile := maptile.Tile{}
		prepreviousTile := maptile.Tile{}
		for _, t := range tmpTiles {
			if t.X != previousTile.X {
				reducedSortedTileSlice = append(reducedSortedTileSlice, t)
			} else if t.X == previousTile.X && prepreviousTile.X == t.X && t.Y >= previousTile.Y {
				reducedSortedTileSlice[len(reducedSortedTileSlice)-1] = t
			} else {
				reducedSortedTileSlice = append(reducedSortedTileSlice, t)
			}

			prepreviousTile = previousTile
			previousTile = t
		}

		// Make a path with all the top and bottom tiles.
		envelope := []maptile.Tile{}
		for i, t := range reducedSortedTileSlice {
			if i%2 == 0 && i < len(reducedSortedTileSlice)-1 {
				envelope = append(envelope, t)
			}
		}
		for i := len(reducedSortedTileSlice) - 1; i >= 0; i-- {
			if i%2 == 1 && i < len(reducedSortedTileSlice)-1 {
				envelope = append(envelope, reducedSortedTileSlice[i])
			}
		}

		envelope = append(envelope, reducedSortedTileSlice[0])

		// Make a set of points, with the centre of the tiles
		var points []orb.Point
		for _, t := range envelope {
			points = append(points, t.Bound().Center())

		}

		agentMission := m
		agentMission.Geometry = orb.Polygon{points}
		unassignedMissions = append(unassignedMissions, agentMission)
		i++
	}

	for _, ah := range unassignedAgents {
		location := orb.Point{ah.State.Position.X, ah.State.Position.Y}
		smallestDistance := 0.0
		smallestDistanceIndex := 0

		for mi, m := range unassignedMissions {
			centroid, _ := planar.CentroidArea(m.Geometry)
			distance := geo.Distance(location, centroid)

			if smallestDistance < distance {
				smallestDistance = distance
				smallestDistanceIndex = mi
			}
		}

		newMissions[ah.Agent.UUID] = unassignedMissions[smallestDistanceIndex]

		unassignedMissions = append(unassignedMissions[:smallestDistanceIndex],
			unassignedMissions[smallestDistanceIndex+1:]...)

	}

	if len(newMissions) > 0 {
		return newMissions, nil
	}
	return nil, fmt.Errorf("Doesnt work")
}

// MissionArea will say something intelligent about the mission
func (m *Mission) MissionArea() (centre orb.Point, area float64) {
	centroid, area := planar.CentroidArea(m.Geometry)
	return centroid, area
}

// GenerateEnvelope generates a new polygon based on
// Zoom is the granularity of the path
// TODO: Maybe use information in agent to automatically guess zoom, or something else
func (m *Mission) GenerateEnvelope(a Agent, zoom maptile.Zoom) (orb.Geometry, error) {
	// Check that the geometry is a Polygon
	// this only works for polygons at the moment
	v, ok := m.Geometry.(orb.Polygon)
	if ok == false {
		log.Fatal("Could not cast geometry to polygon")
	}

	// Convert the polygon to tiles
	tiles := tilecover.Geometry(v, zoom)
	sortedTileSlice := []maptile.Tile{}

	for key := range tiles {
		sortedTileSlice = append(sortedTileSlice, key)
	}

	// Sort by x,y
	sortedTileSlice = sortTiles(sortedTileSlice)[len(sortedTileSlice)/3*2 : len(sortedTileSlice)/3*3]

	// Reduce to a single min and max point along x axis
	reducedSortedTileSlice := []maptile.Tile{}
	previousTile := maptile.Tile{}
	prepreviousTile := maptile.Tile{}
	for _, t := range sortedTileSlice {
		if t.X != previousTile.X {
			reducedSortedTileSlice = append(reducedSortedTileSlice, t)
		} else if t.X == previousTile.X && prepreviousTile.X == t.X && t.Y >= previousTile.Y {
			reducedSortedTileSlice[len(reducedSortedTileSlice)-1] = t
		} else {
			reducedSortedTileSlice = append(reducedSortedTileSlice, t)
		}

		prepreviousTile = previousTile
		previousTile = t
	}

	// Make a path with all the top and bottom tiles.
	envelope := []maptile.Tile{}
	for i, t := range reducedSortedTileSlice {
		if i%2 == 0 && i < len(reducedSortedTileSlice)-1 {
			envelope = append(envelope, t)
		}
	}
	for i := len(reducedSortedTileSlice) - 1; i >= 0; i-- {
		if i%2 == 1 && i < len(reducedSortedTileSlice)-1 {
			envelope = append(envelope, reducedSortedTileSlice[i])
		}
	}

	// Make a set of points, with the centre of the tiles
	var points []orb.Point
	for _, t := range envelope {
		points = append(points, t.Bound().Center())
	}

	return orb.Polygon{points}, nil
}

// GeneratePath to follow based on mission
// Zoom is the granularity of the path
// TODO: Maybe use information in agent to automatically guess zoom, or something else
func (m *Mission) GeneratePath(ah AgentHolder, zoom maptile.Zoom) (orb.Geometry, error) {
	// Check that the geometry is a Polygon
	// this only works for polygons at the moment
	v, ok := m.Geometry.(orb.Polygon)
	if ok == false {
		log.Fatal("Could not cast geometry to polygon")
	}

	// Convert the polygon to tiles
	tiles := tilecover.Geometry(v, zoom)
	sortedTileSlice := []maptile.Tile{}

	for key := range tiles {
		sortedTileSlice = append(sortedTileSlice, key)
	}

	// Sort by x,y
	sortedTileSlice = sortTiles(sortedTileSlice)

	// Reduce to a single min and max point along x axis
	reducedSortedTileSlice := []maptile.Tile{}
	previousTile := maptile.Tile{}
	prepreviousTile := maptile.Tile{}
	for _, t := range sortedTileSlice {
		if t.X != previousTile.X {
			reducedSortedTileSlice = append(reducedSortedTileSlice, t)
		} else if t.X == previousTile.X && prepreviousTile.X == t.X && t.Y >= previousTile.Y {
			reducedSortedTileSlice[len(reducedSortedTileSlice)-1] = t
		} else {
			reducedSortedTileSlice = append(reducedSortedTileSlice, t)
		}

		prepreviousTile = previousTile
		previousTile = t
	}

	// Swap every other coordinate set
	for i := range reducedSortedTileSlice {
		if i%3 == 1 && i < len(reducedSortedTileSlice)-1 {
			reducedSortedTileSlice[i+1], reducedSortedTileSlice[i] = reducedSortedTileSlice[i], reducedSortedTileSlice[i+1]
		}
	}

	// Get the centre of all boxes at this detail level.
	// This should be fine for a first algorithm
	var points []orb.Point
	for _, t := range reducedSortedTileSlice {
		points = append(points, t.Bound().Center())
	}

	agentLocation := orb.Point{ah.State.Position.X, ah.State.Position.Y}
	distanceToStart := geo.Distance(points[0], agentLocation)
	distanceToEnd := geo.Distance(points[len(points)-1], agentLocation)

	if distanceToEnd < distanceToStart {
		points = reversePoints(points)
	}

	//return orb.MultiLineString{points}, nil
	return orb.MultiLineString{points}, nil
}

func reversePoints(input []orb.Point) []orb.Point {
	if len(input) == 0 {
		return input
	}
	return append(reversePoints(input[1:]), input[0])
}

func (m Mission) String() string {
	return fmt.Sprintf("D - %s  Geometry - %v SwarmGeometry - %v", m.Description, m.Geometry, m.SwarmGeometry)
}

// Sort tiles according to (x, y)
func sortTiles(t []maptile.Tile) []maptile.Tile {
	n := len(t)
	sorted := false
	for !sorted {
		swapped := false
		for i := 0; i < n-1; i++ {
			if t[i].Y > t[i+1].Y {
				t[i+1], t[i] = t[i], t[i+1]
				swapped = true
			}
			if t[i].X > t[i+1].X {
				t[i+1], t[i] = t[i], t[i+1]
				swapped = true
			}
		}
		if !swapped {
			sorted = true
		}
		n = n - 1
	}
	return t
}

// LoadFeatures loads stuff from file
func (m *Mission) LoadFeatures(path string) {
	data, err := ioutil.ReadFile(path)
	if err != nil {
		log.Fatalf("unable to read file: %v", err)
	}

	f, err := geojson.UnmarshalFeature(data)
	if err == nil {
		m.Geometry = f.Geometry
		return
	}

	fc, err := geojson.UnmarshalFeatureCollection(data)
	if err == nil {
		if len(fc.Features) != 1 {
			log.Fatalf("must have 1 feature: %v", len(fc.Features))
		}
		m.Geometry = fc.Features[0].Geometry
		return
	}

	g, err := geojson.UnmarshalGeometry(data)
	if err != nil {
		log.Fatalf("unable to unmarshal feature: %v", err)
	}

	m.Geometry = geojson.NewFeature(g.Geometry()).Geometry
}

// UnmarshalJSON will unmarshal the correct geometry from the json structure.
// This is being done by hand so we can hide the Geometry Unmarshalling
func (m *Mission) UnmarshalJSON(rawData []byte) error {
	var dat map[string]*json.RawMessage

	if err := json.Unmarshal(rawData, &dat); err != nil {
		panic(err)
	}

	err := json.Unmarshal(*dat["Description"], &m.Description)
	if err != nil {
		fmt.Println(err)
	}

	err = json.Unmarshal(*dat["MissionType"], &m.MissionType)
	if err != nil {
		fmt.Println(err)
	}

	err = json.Unmarshal(*dat["AreaLink"], &m.AreaLink)
	if err != nil {
		fmt.Println(err)
	}

	err = json.Unmarshal(*dat["MetaNeeded"], &m.MetaNeeded)
	if err != nil {
		fmt.Println(err)
	}

	err = json.Unmarshal(*dat["Goal"], &m.Goal)
	if err != nil {
		fmt.Println(err)
	}

	// if no geoemtry is  set, we ignore the below part
	if dat["Geometry"] == nil {
		return nil
	}
	// Generate the JSON we need,
	// based in the knowledte that this is a polygon
	geometryString := fmt.Sprintf("%s%s%s", "{\"type\": \"Polygon\",\"coordinates\":", string(*dat["Geometry"]), "}")
	f, err := geojson.UnmarshalFeature([]byte(geometryString))
	if err == nil {
		m.Geometry = f.Geometry
	}
	fc, err := geojson.UnmarshalFeatureCollection([]byte(geometryString))
	if err == nil {
		if len(fc.Features) != 1 {
			log.Fatalf("must have 1 feature: %v", len(fc.Features))
		}
		m.Geometry = fc.Features[0].Geometry
	}
	g, err := geojson.UnmarshalGeometry([]byte(geometryString))
	if err != nil {
		log.Fatalf("unable to unmarshal feature: %v", err)
	}
	m.Geometry = geojson.NewFeature(g.Geometry()).Geometry

	// err = json.Unmarshal(*dat["AgentArea"], &m.AgentArea)
	// if err != nil {
	// 	fmt.Println(err)
	// }

	// if no Swarmgeoemtry is  set, we ignore the below part
	if dat["SwarmGeometry"] == nil {
		return nil
	}
	// Generate the JSON we need,
	// based in the knowledte that this is a polygon
	swarmgeometryString := fmt.Sprintf("%s%s%s", "{\"type\": \"Polygon\",\"coordinates\":", string(*dat["SwarmGeometry"]), "}")
	fs, err := geojson.UnmarshalFeature([]byte(swarmgeometryString))
	if err == nil {
		m.SwarmGeometry = fs.Geometry
	}
	fcs, err := geojson.UnmarshalFeatureCollection([]byte(swarmgeometryString))
	if err == nil {
		if len(fc.Features) != 1 {
			log.Fatalf("must have 1 feature: %v", len(fc.Features))
		}
		m.SwarmGeometry = fcs.Features[0].Geometry
	}
	gs, err := geojson.UnmarshalGeometry([]byte(swarmgeometryString))
	if err != nil {
		log.Fatalf("unable to unmarshal feature: %v", err)
	}
	m.SwarmGeometry = geojson.NewFeature(gs.Geometry()).Geometry

	return nil
}
