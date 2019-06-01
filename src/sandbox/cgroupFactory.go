package sandbox

import (
	"container/list"
	"fmt"
	"os"
	"path"
	"sync"
)

type CgroupFactory struct {
	mutex    sync.Mutex
	poolSize int
	freeCg   *list.List
}

func NewCgroupFactory(prefix string, poolSize int) (*CgroupFactory, error) {
	var freeCg *list.List
	freeCg = nil

	if poolSize != 0 {
		// init cgroup pool
		freeCg = list.New()
		for i := 1; i <= poolSize; i++ {
			cgId := fmt.Sprintf("%s-%d", prefix, i)
			if err := CreateCg(cgId); err != nil {
				return nil, err
			} else {
				freeCg.PushBack(cgId)
			}
		}
	}

	return &CgroupFactory{poolSize: poolSize, freeCg: freeCg}, nil
}

func (cgf *CgroupFactory) GetCg(id string) (string, error) {
	var err error
	var cgId string
	err = nil

	if cgf.freeCg != nil {
		cgf.mutex.Lock()
		if cgEle := cgf.freeCg.Front(); cgEle != nil {
			cgId = cgEle.Value.(string)
			cgf.freeCg.Remove(cgEle)
		}
		cgf.mutex.Unlock()
	}

	if cgId == "" {
		err = CreateCg(id)
		cgId = id
	}

	return cgId, err
}

func (cgf *CgroupFactory) PutCg(id, cgId string) error {
	if id == cgId {
		if err := DestroyCg(cgId); err != nil {
			return err
		}
	} else {
		cgf.mutex.Lock()
		cgf.freeCg.PushBack(cgId)
		cgf.mutex.Unlock()
	}

	return nil
}

func CreateCg(cgId string) error {
	for _, cgroup := range CGroupList {
		cgroupPath := path.Join("/sys/fs/cgroup/", cgroup, OLCGroupName, cgId)
		if err := os.MkdirAll(cgroupPath, 0700); err != nil {
			return err
		}
	}

	return nil
}

func DestroyCg(cgId string) error {
	for _, cg := range CGroupList {
		cgPath := path.Join("/sys/fs/cgroup/", cg, OLCGroupName, cgId)
		if err := os.Remove(cgPath); err != nil {
			return err
		}
	}

	return nil
}
