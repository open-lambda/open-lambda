package sandbox

import (
	"container/list"
	"fmt"
	"os/exec"
	"strings"
	"sync"
)

const gatewayIp string = "10.0.0.1"

var prefixMap = map[string]string{
	"sandbox": "10.0",
	"cache":   "10.1",
}

type NetnsFactory struct {
	mutex    sync.Mutex
	poolSize int
	freeIp   *list.List
	idMap    map[string]string
	ipMap    map[string]string
}

// prefix is the first two octets of the vguest ip address
func NewNetnsFactory(prefix string, poolSize int) *NetnsFactory {
	freeIp := list.New()

	// generate ip addrs
	octet3 := 1
	for poolSize > 0 {
		i := 1
		for ; i <= poolSize && i <= 255; i++ {
			ip := fmt.Sprintf("%s.%d.%d", prefixMap[prefix], octet3, i)
			freeIp.PushBack(ip)
		}
		poolSize -= (i - 1)
		octet3 += 1
	}

	return &NetnsFactory{
		poolSize: poolSize,
		freeIp:   freeIp,
		idMap:    make(map[string]string),
		ipMap:    make(map[string]string)}
}

func (nnf *NetnsFactory) CreateNetns(id string) error {
	prefix := strings.Split(id, "-")
	nsId := prefix[len(prefix)-1]

	vhost := "vh-" + nsId
	vguest := "vg-" + nsId

	nnf.mutex.Lock()
	ipEle := nnf.freeIp.Front()
	ip := ipEle.Value.(string)
	nnf.freeIp.Remove(ipEle)

	if _, ok := nnf.ipMap[nsId]; ok {
		nsId = prefix[len(prefix)-2]
	}
	nnf.idMap[id] = nsId
	nnf.ipMap[nsId] = ip
	nnf.mutex.Unlock()

	// create ns
	cmd := exec.Command("ip", "netns", "add", nsId)
	if err := cmd.Run(); err != nil {
		return err
	}

	// create veth pair
	cmd = exec.Command("ip", "link", "add", vhost, "type", "veth", "peer", "name", vguest)
	if err := cmd.Run(); err != nil {
		return err
	}

	// move vguest into ns
	cmd = exec.Command("ip", "link", "set", vguest, "netns", nsId)
	if err := cmd.Run(); err != nil {
		return err
	}

	// set vhost up
	cmd = exec.Command("ip", "link", "set", vhost, "up")
	if err := cmd.Run(); err != nil {
		return err
	}

	// add vhost to bridge
	cmd = exec.Command("ip", "link", "set", "dev", vhost, "master", "br1")
	if err := cmd.Run(); err != nil {
		return err
	}

	// set loopback device up
	cmd = exec.Command("ip", "netns", "exec", nsId, "ip", "link", "set", "lo", "up")
	if err := cmd.Run(); err != nil {
		return err
	}

	// add ip to vguest
	cmd = exec.Command("ip", "netns", "exec", nsId, "ip", "addr", "add", ip+"/8", "dev", vguest)
	if err := cmd.Run(); err != nil {
		return err
	}

	// bring guest up
	cmd = exec.Command("ip", "netns", "exec", nsId, "ip", "link", "set", vguest, "up")
	if err := cmd.Run(); err != nil {
		return err
	}

	// set default gateway for guest
	cmd = exec.Command("ip", "netns", "exec", nsId, "ip", "route", "add", "default", "via", gatewayIp)
	if err := cmd.Run(); err != nil {
		return err
	}

	return nil
}

func (nnf *NetnsFactory) DestroyNetns(id string) error {
	nnf.mutex.Lock()
	nsId := nnf.idMap[id]
	nnf.freeIp.PushBack(nnf.ipMap[nsId])
	delete(nnf.idMap, id)
	delete(nnf.ipMap, nsId)
	nnf.mutex.Unlock()

	vhost := "vh-" + nsId

	cmd := exec.Command("ip", "link", "delete", "dev", vhost)
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("NetnsFactory: Failed to remove vhost: %s, %v", vhost, err.Error())
	}

	cmd = exec.Command("ip", "netns", "del", nsId)
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("NetnsFactory: Failed to delete ns: %s, %v", nsId, err.Error())
	}

	return nil
}

func (nnf *NetnsFactory) GetNsId(id string) string {
	return nnf.idMap[id]
}
