package agentlogic

// An Agent is a thing - anything that can move
type Agent struct {
	UUID               string
	Nick               string
	URI                string
	Position           Vector
	Key                string
	Battery            int
	MovementDimensions int
	Hardware           []string
	Software           []string
}

type AgentHolder struct {
	Agent     Agent
	State     State
	LastSeen  int64
	AgentType AgentType
}

type State struct {
	ID           string
	Mission      Mission
	Battery      int
	Position     Vector
	MissionIndex int
}

// Point - struct holding X Y Z values
type Vector struct {
	X, Y, Z float64
}

type AgentType int

const (
	ControllerAgent = 0
	ContextAgent    = 1
)
