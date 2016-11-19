package main

import (
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"path"
	"path/filepath"
	"sort"
	"strings"

	docker "github.com/fsouza/go-dockerclient"
	"github.com/open-lambda/open-lambda/worker/config"
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

func (args *CmdArgs) PidPath(name string) string {
	return path.Join(*args.cluster, "logs", name+".pid")
}

func (args *CmdArgs) ConfigPath(name string) string {
	return path.Join(*args.cluster, "config", name+".json")
}

func (args *CmdArgs) TemplatePath() string {
	return args.ConfigPath("template")
}

func (args *CmdArgs) RegistryPath() string {
	return path.Join(*args.cluster, "registry")
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

func (admin *Admin) cluster_nodes(cluster string) (map[string]([]string), error) {
	client := admin.client
	nodes := map[string]([]string){}

	containers, err := client.ListContainers(docker.ListContainersOptions{})
	if err != nil {
		return nil, err
	}

	for _, container := range containers {
		if container.Labels[sandbox.DOCKER_LABEL_CLUSTER] == cluster {
			cid := container.ID
			type_label := container.Labels[sandbox.DOCKER_LABEL_TYPE]
			nodes[type_label] = append(nodes[type_label], cid)
		}
	}

	return nodes, nil
}

func (admin *Admin) help() error {
	fmt.Printf("Run %v <command> <args>\n", os.Args[0])
	fmt.Printf("\n")
	fmt.Printf("Commands:\n")
	cmds := make([]string, 0, len(admin.fns))
	for cmd := range admin.fns {
		cmds = append(cmds, cmd)
	}
	sort.Strings(cmds)

	for _, cmd := range cmds {
		fmt.Printf("  %v\n", cmd)
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

	if err := os.Mkdir(args.RegistryPath(), 0700); err != nil {
		return err
	}

	// config dir and template
	if err := os.Mkdir(path.Join(*args.cluster, "config"), 0700); err != nil {
		return err
	}
	c := &config.Config{
		Worker_port:  "?",
		Cluster_name: *args.cluster,
		Registry:     "local",
		Reg_dir:      args.RegistryPath(),
	}
	if err := c.Defaults(); err != nil {
		return err
	}
	if err := c.Save(args.TemplatePath()); err != nil {
		return err
	}

	fmt.Printf("Cluster Directory: %s\n\n", *args.cluster)
	fmt.Printf("Worker Defaults: \n%s\n\n", c.DumpStr())
	fmt.Printf("You may now start a cluster using the \"workers\" command\n")

	return nil
}

func (admin *Admin) status() error {
	args := NewCmdArgs()
	args.Parse(false)

	containers1, err := admin.client.ListContainers(docker.ListContainersOptions{})
	if err != nil {
		return err
	}

	if *args.cluster == "" {
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
		fmt.Printf("Nodes in %s cluster:\n", *args.cluster)
		for _, containers2 := range containers1 {
			if containers2.Labels[sandbox.DOCKER_LABEL_CLUSTER] == *args.cluster {
				name := containers2.Names[0]
				oltype := containers2.Labels[sandbox.DOCKER_LABEL_TYPE]
				fmt.Printf("  <%s> (%s)\n", name, oltype)
			}
		}
	}

	return nil
}

func (admin *Admin) rethinkdb() error {
	args := NewCmdArgs()
	count := args.flags.Int("count", 1, "specify number of nodes to start")
	args.Parse(true)

	client := admin.client
	labels := map[string]string{}
	labels[sandbox.DOCKER_LABEL_CLUSTER] = *args.cluster
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
	conf := flags.String("config", "", "give a json config file")
	flags.Parse(os.Args[2:])

	if *conf == "" {
		fmt.Printf("Please specify a json config file\n")
		return nil
	}

	server.Main(*conf)
	return nil
}

func (admin *Admin) workers() error {
	args := NewCmdArgs()
	foreach := args.flags.Bool("foreach", false, "start one worker per db instance")
	portbase := args.flags.Int("port", 8080, "port range [port, port+n) will be used for workers")
	n := args.flags.Int("n", 1, "specify number of workers to start")
	args.Parse(true)

	worker_confs := []*config.Config{}
	if *foreach {
		nodes, err := admin.cluster_nodes(*args.cluster)
		if err != nil {
			return err
		}

		// start one worker per db shard
		for _, cid := range nodes["db"] {
			container, err := admin.client.InspectContainer(cid)
			if err != nil {
				return err
			}

			fmt.Printf("DB node: %v\n", container.NetworkSettings.IPAddress)

			c, err := config.ParseConfig(args.TemplatePath())
			if err != nil {
				return err
			}
			worker_confs = append(worker_confs, c)
		}
	} else {
		for i := 0; i < *n; i++ {
			c, err := config.ParseConfig(args.TemplatePath())
			if err != nil {
				return err
			}
			worker_confs = append(worker_confs, c)
		}
	}

	for i, conf := range worker_confs {
		conf_path := args.ConfigPath(fmt.Sprintf("worker-%d", i))
		conf.Worker_port = fmt.Sprintf("%d", *portbase+i)
		if err := conf.Save(conf_path); err != nil {
			return err
		}

		// stdout+stderr both go to log
		log_path := args.LogPath(fmt.Sprintf("worker-%d.out", i))
		f, err := os.Create(log_path)
		if err != nil {
			return err
		}
		attr := os.ProcAttr{
			Files: []*os.File{nil, f, f},
		}
		cmd := []string{
			os.Args[0],
			"worker",
			"-config=" + conf_path,
		}
		proc, err := os.StartProcess(os.Args[0], cmd, &attr)
		if err != nil {
			return err
		}

		pidpath := args.PidPath(fmt.Sprintf("worker-%d", i))
		if err := ioutil.WriteFile(pidpath, []byte(fmt.Sprintf("%d", proc.Pid)), 0644); err != nil {
			return err
		}

		fmt.Printf("Started worker: pid %d, port %s, log at %s\n", proc.Pid, conf.Worker_port, log_path)
	}

	return nil
}

func (admin *Admin) kill() error {
	args := NewCmdArgs()
	args.Parse(true)

	client := admin.client
	containers1, err := client.ListContainers(docker.ListContainersOptions{})
	if err != nil {
		return err
	}

	for _, containers2 := range containers1 {
		if containers2.Labels[sandbox.DOCKER_LABEL_CLUSTER] == *args.cluster {
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
