package state

type HandlerState int

const (
	Unitialized HandlerState = iota
	Stopped                  // TODO(tyler): split into new and stopped?
	Running
	Paused
)

func (h HandlerState) String() string {
	switch h {
	case Unitialized:
		return "unitialized"
	case Stopped:
		return "stoppend"
	case Running:
		return "running"
	case Paused:
		return "paused"
	default:
		panic("Unknown state!")
	}
}
