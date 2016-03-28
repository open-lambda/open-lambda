package frontends

type FrontEnd interface {
	// returns the human-readable name of this frontend
	FrontEndName() string

	// Adds a template lambda at <location>
	AddLambda(location string)

	// Builds lambda at <path>
	BuildLambda(path string)

	// Returns the ID for a given lambda path
	// Id's are used to identify an individual lambda throughout the system
	// Docker image tags are an example use of Id's
	GetId(path string) (string, error)
}
