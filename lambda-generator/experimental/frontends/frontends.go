package frontends

type FrontEnd interface {
	// returns the human-readable name of this frontend
	FrontEndName() string

	// Adds a template lambda at <location>
	AddLambda(location string)
}
