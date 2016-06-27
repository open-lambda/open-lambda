package sandbox

type SandboxManager interface {
	Create(name string) Sandbox
}
