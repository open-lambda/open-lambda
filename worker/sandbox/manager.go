package sandbox

type SandboxManager interface {
	Create(name string) Sandbox
	Pull(name string) error
}
