package handler_state

type HandlerState int

const (
	Unitialized HandlerState = iota
	Stopped                  // TODO(tyler): split into new and stopped?
	Running
	Paused
)
