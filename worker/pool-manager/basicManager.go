package pmanager

import (
	"bufio"
	"errors"
	"fmt"
	"os"
	"time"

	docker "github.com/fsouza/go-dockerclient"
	dutil "github.com/open-lambda/open-lambda/worker/dockerutil"
	sb "github.com/open-lambda/open-lambda/worker/sandbox"

	"github.com/open-lambda/open-lambda/worker/config"
	"github.com/open-lambda/open-lambda/worker/pool-manager/policy"
)

type BasicManager struct {
	servers []policy.ForkServer
	poolDir string
	cid     string
	matcher policy.CacheMatcher
	evictor policy.CacheEvictor
}

func NewBasicManager(opts *config.Config) (bm *BasicManager, err error) {
	poolDir := opts.Pool_dir
	numServers := opts.Num_forkservers

	cid, err := initPoolContainer(poolDir, opts.Cluster_name, numServers)
	if err != nil {
		return nil, err
	}

	pidFile, err := os.Open(fmt.Sprintf("%s/fspids", poolDir))
	if err != nil {
		return nil, err
	}
	defer pidFile.Close()

	scnr := bufio.NewScanner(pidFile)

	servers := make([]policy.ForkServer, numServers, numServers)
	for k := 0; k < numServers; k++ {
		sockPath := fmt.Sprintf("%s/fs%d/fs.sock", poolDir, k)

		start := time.Now()
		// wait up to 5s for server to initialize
		for os.IsNotExist(err) {
			_, err = os.Stat(sockPath)
			if time.Since(start).Seconds() > 5 {
				return nil, errors.New("forkservers failed to initialize")
			}
		}

		if !scnr.Scan() {
			return nil, errors.New("too few lines in fspid file")
		}

		fspid := scnr.Text()

		if err := scnr.Err(); err != nil {
			return nil, err
		}

		servers[k] = policy.ForkServer{
			Pid:      fspid,
			SockPath: sockPath,
			Packages: []string{},
		}
	}

	bm = &BasicManager{
		servers: servers,
		cid:     cid,
		matcher: policy.NewRandomMatcher(servers),
		evictor: policy.NewRandomEvictor(servers),
	}

	return bm, nil
}

func (bm *BasicManager) ForkEnter(sandbox sb.ContainerSandbox) (err error) {
	fs, _ := bm.matcher.Match([]string{})

	// signal interpreter to forkenter into sandbox's namespace
	pid, err := sendFds(fs.SockPath, sandbox.NSPid())
	if err != nil {
		return err
	}

	// change cgroup of spawned lambda server
	if err = sandbox.CGroupEnter(pid); err != nil {
		return err
	}

	return nil
}

func initPoolContainer(poolDir, clusterName string, numServers int) (cid string, err error) {
	client, err := docker.NewClientFromEnv()
	if err != nil {
		return "", err
	}

	if err = os.MkdirAll(poolDir, os.ModeDir); err != nil {
		return "", err
	}

	labels := map[string]string{
		dutil.DOCKER_LABEL_CLUSTER: clusterName,
		dutil.DOCKER_LABEL_TYPE:    dutil.POOL,
	}

	volumes := []string{
		fmt.Sprintf("%s:%s", poolDir, "/host"),
	}

	caps := []string{"SYS_ADMIN"}

	cmd := []string{"python", "/initservers.py", fmt.Sprintf("%d", numServers)}

	container, err := client.CreateContainer(
		docker.CreateContainerOptions{
			Config: &docker.Config{
				Image:  dutil.POOL_IMAGE,
				Labels: labels,
				Cmd:    cmd,
			},
			HostConfig: &docker.HostConfig{
				Binds:   volumes,
				PidMode: "host",
				CapAdd:  caps,
			},
		},
	)

	if err := client.StartContainer(container.ID, nil); err != nil {
		return "", err
	}

	return container.ID, nil
}
