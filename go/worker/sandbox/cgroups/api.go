package cgroups

type Cgroup interface {
	Name() string
	GetMemUsageMB() int
	GetMemLimitMB() int
	SetMemLimitMB(mb int)
	Pause() error
	Unpause() error
	Release()
	AddPid(pid string) error
	GetPIDs() ([]string, error)
	KillAllProcs()
	DebugString() string

	// TODO: find a way to rip this out.  Higher layers should not
	// directly be accessing cgroup file entries.
	CgroupProcsPath() string
}
