package state

type HandlerState int

const (
	Unitialized HandlerState = iota
	Running
	Paused
)

func (h HandlerState) String() string {
	switch h {
	case Unitialized:
		return "unitialized"
	case Running:
		return "running"
	case Paused:
		return "paused"
	default:
		panic("Unknown state!")
	}
}
