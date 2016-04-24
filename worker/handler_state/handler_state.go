package handler_state

type HandlerState int

const (
	Unitialized HandlerState = iota
	Stopped
	Running
	Paused
)
