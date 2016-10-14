package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"path"
	"path/filepath"
	"strings"

	docker "github.com/fsouza/go-dockerclient"
	"github.com/open-lambda/open-lambda/worker/sandbox"
	"github.com/open-lambda/open-lambda/worker/server"
)

type Admin struct {
	client *docker.Client
	fns    map[string]AdminFn
}

type CmdArgs struct {
	flags   *flag.FlagSet
	cluster *string
}

func NewCmdArgs() *CmdArgs {
	args := CmdArgs{}
	args.flags = flag.NewFlagSet("flag", flag.ExitOnError)
	args.cluster = args.flags.String("cluster", "", "give a cluster directory")
	return &args
}

func (args *CmdArgs) Parse(require_cluster bool) {
	args.flags.Parse(os.Args[2:])

	if *args.cluster != "" {
		abscluster, err := filepath.Abs(*args.cluster)
		*args.cluster = abscluster
		if err != nil {
			log.Fatal("failed to get abs cluster dir: ", err)
		}
	} else if require_cluster {
		log.Fatal("please specify a cluster directory")
	}
}

func (args *CmdArgs) LogPath(name string) string {
	return path.Join(*args.cluster, "logs", name)
}

type AdminFn func() error

func NewAdmin() *Admin {
	admin := Admin{fns: map[string]AdminFn{}}
	if client, err := docker.NewClientFromEnv(); err != nil {
		log.Fatal("failed to get docker client: ", err)
	} else {
		admin.client = client
	}

	admin.fns["help"] = admin.help
	admin.fns["new-cluster"] = admin.new_cluster
	admin.fns["status"] = admin.status
	admin.fns["rethinkdb"] = admin.rethinkdb
	admin.fns["worker"] = admin.worker
	admin.fns["workers"] = admin.workers
	admin.fns["kill"] = admin.kill
	return &admin
}

func (admin *Admin) command(cmd string) {
	fn := admin.fns[cmd]
	if fn == nil {
		admin.help()
		return
	}
	if err := fn(); err != nil {
		log.Fatalf("Failed to run %v, %v\n", cmd, err)
	}
}

func (admin *Admin) help() error {
	fmt.Printf("Run %v <command> <args>\n", os.Args[0])
	fmt.Printf("\n")
	fmt.Printf("Commands:\n")
	for command, _ := range admin.fns {
		fmt.Printf("  %v\n", command)
	}
	return nil
}

func (admin *Admin) new_cluster() error {
	args := NewCmdArgs()
	args.Parse(true)

	if err := os.Mkdir(*args.cluster, 0700); err != nil {
		return err
	}

	if err := os.Mkdir(path.Join(*args.cluster, "logs"), 0700); err != nil {
		return err
	}

	fmt.Printf("%s\n", *args.cluster)
	return nil
}

func (admin *Admin) status() error {
	flags := flag.NewFlagSet("flag", flag.ExitOnError)
	cluster_rel := flags.String("cluster", "", "give a cluster directory")
	flags.Parse(os.Args[2:])

	containers1, err := admin.client.ListContainers(docker.ListContainersOptions{})
	if err != nil {
		return err
	}

	if *cluster_rel == "" {
		other := 0
		node_counts := map[string]int{}

		for _, containers2 := range containers1 {
			label := containers2.Labels[sandbox.DOCKER_LABEL_CLUSTER]
			if label != "" {
				node_counts[label] += 1
			} else {
				other += 1
			}
		}

		fmt.Printf("%d container(s) without OpenLambda labels\n\n", other)
		fmt.Printf("%d cluster(s):\n", len(node_counts))
		for cluster_name, count := range node_counts {
			fmt.Printf("  <%s> (%d nodes)\n", cluster_name, count)
		}
		fmt.Printf("\n")
		fmt.Printf("For info about a specific cluster, use -cluster=<cluster-name>\n")
	} else {
		cluster, err := filepath.Abs(*cluster_rel)
		if err != nil {
			return err
		}
		fmt.Printf("Nodes in %s cluster:\n", cluster)
		for _, containers2 := range containers1 {
			if containers2.Labels[sandbox.DOCKER_LABEL_CLUSTER] == cluster {
				name := containers2.Names[0]
				oltype := containers2.Labels[sandbox.DOCKER_LABEL_TYPE]
				fmt.Printf("  <%s> (%s)\n", name, oltype)
			}
		}
	}

	return nil
}

func (admin *Admin) rethinkdb() error {
	client := admin.client
	flags := flag.NewFlagSet("flag", flag.ExitOnError)
	cluster_rel := flags.String("cluster", "", "give a cluster directory")
	count := flags.Int("count", 1, "specify number of nodes to start")
	flags.Parse(os.Args[2:])

	if *cluster_rel == "" {
		fmt.Printf("Please specify a cluster\n")
		return nil
	}

	cluster, err := filepath.Abs(*cluster_rel)
	if err != nil {
		return err
	}

	labels := map[string]string{}
	labels[sandbox.DOCKER_LABEL_CLUSTER] = cluster
	labels[sandbox.DOCKER_LABEL_TYPE] = "db"

	var first_container *docker.Container

	for i := 0; i < *count; i++ {
		cmd := []string{"rethinkdb", "--bind", "all"}
		if first_container != nil {
			ip := first_container.NetworkSettings.IPAddress
			cmd = append(cmd, "--join", fmt.Sprintf("%s:%d", ip, 29015))
		}

		fmt.Printf("Starting shard: %s\n", strings.Join(cmd, " "))

		// create and start container
		container, err := client.CreateContainer(
			docker.CreateContainerOptions{
				Config: &docker.Config{
					Cmd:    cmd,
					Image:  "rethinkdb",
					Labels: labels,
				},
			},
		)
		if err != nil {
			return err
		}
		if err := client.StartContainer(container.ID, container.HostConfig); err != nil {
			return err
		}

		// get network assignments
		container, err = client.InspectContainer(container.ID)
		if err != nil {
			return err
		}

		if i == 0 {
			first_container = container
		}
	}

	return nil
}

func (admin *Admin) worker() error {
	flags := flag.NewFlagSet("flag", flag.ExitOnError)
	config := flags.String("config", "", "give a json config file")
	flags.Parse(os.Args[2:])

	if *config == "" {
		fmt.Printf("Please specify a json config file\n")
		return nil
	}

	server.Main(*config)

	return nil
}

func (admin *Admin) workers() error {
	args := NewCmdArgs()
	config := args.flags.String("config", "", "give a json config file")
	count := args.flags.Int("count", 1, "specify number of workers to start")
	args.Parse(true)

	if *config == "" {
		fmt.Printf("Please specify a json config file\n")
		return nil
	}

	for i := 0; i < *count; i++ {
		logpath := args.LogPath(fmt.Sprintf("worker-%d.out", i))
		f, err := os.Create(logpath)
		if err != nil {
			return err
		}
		attr := os.ProcAttr{
			Files: []*os.File{nil, f, f},
		}
		cmd := []string{
			os.Args[0],
			"worker",
			"-config=" + *config,
		}
		proc, err := os.StartProcess(os.Args[0], cmd, &attr)
		if err != nil {
			return err
		}
		fmt.Printf("Started worker [pid %d], log at %s\n", proc.Pid, logpath)
	}

	return nil
}

func (admin *Admin) kill() error {
	client := admin.client
	flags := flag.NewFlagSet("flag", flag.ExitOnError)
	cluster_rel := flags.String("cluster", "", "give a cluster directory")
	flags.Parse(os.Args[2:])

	if *cluster_rel == "" {
		fmt.Printf("Please specify a cluster\n")
		return nil
	}
	cluster, err := filepath.Abs(*cluster_rel)
	if err != nil {
		return err
	}

	containers1, err := client.ListContainers(docker.ListContainersOptions{})
	if err != nil {
		return err
	}

	for _, containers2 := range containers1 {
		if containers2.Labels[sandbox.DOCKER_LABEL_CLUSTER] == cluster {
			cid := containers2.ID
			container, err := client.InspectContainer(cid)
			if err != nil {
				return err
			}
			if container.State.Paused {
				fmt.Printf("Unpause container %v\n", cid)
				if err := client.UnpauseContainer(cid); err != nil {
					return err
				}
			}

			fmt.Printf("Kill container %v\n", cid)
			opts := docker.KillContainerOptions{ID: cid}
			if err := client.KillContainer(opts); err != nil {
				return err
			}
		}
	}

	return nil
}

func main() {
	admin := NewAdmin()
	if len(os.Args) < 2 {
		admin.help()
		os.Exit(1)
	}
	admin.command(os.Args[1])
}
