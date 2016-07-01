package sandbox

type SandboxManager interface {
	Create(name string) (Sandbox, error)
	Pull(name string) error
}
