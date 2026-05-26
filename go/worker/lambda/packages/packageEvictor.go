package packages

import (
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"sort"
	"sync/atomic"
	"time"

	"github.com/open-lambda/open-lambda/go/common"
)

type PackageEvictor struct {
	puller    *PackagePuller
	evictions int64
	evictChan chan struct{}
	evictDone chan struct{}
}

func NewPackageEvictor(puller *PackagePuller) *PackageEvictor {
	evictor := &PackageEvictor{
		puller:    puller,
		evictChan: make(chan struct{}, 1),
		evictDone: make(chan struct{}),
	}
	common.SetGauge("packages.evictions-total", 0)
	go evictor.evictionTask()
	return evictor
}

// NotifyEvictionCheck queues a background eviction pass.
func (pe *PackageEvictor) NotifyEvictionCheck() {
	select {
	case pe.evictChan <- struct{}{}:
	default:
	}
}

// Cleanup stops the background eviction worker.
func (pe *PackageEvictor) Cleanup() {
	close(pe.evictChan)
	<-pe.evictDone
}

func (pe *PackageEvictor) evictionTask() {
	defer close(pe.evictDone)
	for range pe.evictChan {
		pe.evictIfNeeded()
	}
}

func (pe *PackageEvictor) evictIfNeeded() {
	if common.Conf.Pkgs_max_size <= 0 {
		return
	}

	limitBytes := int64(common.Conf.Pkgs_max_size) * 1024 * 1024
	if pe.puller.TotalSize() <= limitBytes {
		return
	}

	pp := pe.puller
	pp.lifecycleMu.Lock()
	defer pp.lifecycleMu.Unlock()

	for atomic.LoadInt64(&pp.totalSize) > limitBytes {
		candidates := pe.evictablePackagesLocked()
		if len(candidates) == 0 {
			return
		}

		sort.Slice(candidates, func(i, j int) bool {
			if candidates[i].LastAccessed.Equal(candidates[j].LastAccessed) {
				return candidates[i].InstallTime.Before(candidates[j].InstallTime)
			}
			return candidates[i].LastAccessed.Before(candidates[j].LastAccessed)
		})

		evictedAny := false
		for _, candidate := range candidates {
			if atomic.LoadInt64(&pp.totalSize) <= limitBytes {
				return
			}
			if pe.evictPackageLocked(candidate, limitBytes) {
				evictedAny = true
			}
		}

		if !evictedAny {
			return
		}
	}
}

func (pe *PackageEvictor) evictablePackagesLocked() []*Package {
	pp := pe.puller
	candidates := []*Package{}
	pp.packages.Range(func(_, value any) bool {
		p := value.(*Package)
		if pe.canEvict(p) {
			candidates = append(candidates, p)
		}
		return true
	})
	return candidates
}

func (pe *PackageEvictor) canEvict(p *Package) bool {
	if atomic.LoadUint32(&p.installed) == 0 {
		return false
	}
	return atomic.LoadInt32(&p.funcRefs) == 0 && atomic.LoadInt32(&p.pkgRefs) == 0
}

func (pe *PackageEvictor) evictPackageLocked(p *Package, limitBytes int64) bool {
	pp := pe.puller
	p.installMutex.Lock()
	defer p.installMutex.Unlock()

	if !pe.canEvict(p) {
		return false
	}

	pkgPath := filepath.Join(common.Conf.Pkgs_dir, p.Name)
	if err := os.RemoveAll(pkgPath); err != nil {
		slog.Warn(fmt.Sprintf("failed to evict package %s: %v", p.Name, err))
		return false
	}

	freed := p.size
	p.size = 0
	p.InstallTime = time.Time{}
	p.LastAccessed = time.Time{}
	atomic.StoreUint32(&p.installed, 0)
	pp.packages.Delete(p.Name)

	if freed > 0 {
		atomic.AddInt64(&pp.totalSize, -freed)
	}
	common.SetGauge("packages.total-size-bytes", atomic.LoadInt64(&pp.totalSize))
	common.SetGauge("packages.evictions-total", atomic.AddInt64(&pe.evictions, 1))

	for _, depName := range uniquePackages(p.Meta.Deps) {
		tmp, ok := pp.packages.Load(depName)
		if !ok {
			continue
		}
		dep := tmp.(*Package)
		refs := atomic.AddInt32(&dep.pkgRefs, -1)
		if refs < 0 {
			panic(fmt.Sprintf("negative pkgRefs for %s", dep.Name))
		}
		if atomic.LoadInt64(&pp.totalSize) > limitBytes && pe.canEvict(dep) {
			pe.evictPackageLocked(dep, limitBytes)
		}
	}

	return true
}
