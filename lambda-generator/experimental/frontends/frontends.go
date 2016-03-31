package frontends

type FrontEnd interface {

	// Adds a template lambda at <location>
	// Must be provided by frontend
	AddLambda(location string)

	// Builds lambda at <path>
	// Must be provided by frontend
	BuildLambda(path string)

	// Returns the ID for a given lambda path
	// Id's are used to identify an individual lambda throughout the system
	// Docker image tags are an example use of Id's
	// Implemented in base.go, override only if needed
	GetId(path string) (string, error)
}
