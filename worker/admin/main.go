package main

import (
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"path"
	"path/filepath"
	"sort"
	"strconv"
	"strings"

	docker "github.com/fsouza/go-dockerclient"
	sbmanager "github.com/open-lambda/open-lambda/worker/sandbox-manager"

	"github.com/open-lambda/open-lambda/registry"
	"github.com/open-lambda/open-lambda/worker/config"
	"github.com/open-lambda/open-lambda/worker/server"
)

// TODO: notes about setup process
// TODO: notes about creating a directory in local
// TODO: docker registry setup

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

func (args *CmdArgs) WorkerPath(name string) string {
	return path.Join(*args.cluster, "workers", name)
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

type AdminFn struct {
	fn       func() error
	doc      string
	doc_long string
}

func NewAdmin() *Admin {
	admin := Admin{fns: map[string]AdminFn{}}
	if client, err := docker.NewClientFromEnv(); err != nil {
		log.Fatal("failed to get docker client: ", err)
	} else {
		admin.client = client
	}

	admin.fns["help"] = AdminFn{admin.help,
		"Print usage",
		strings.Join([]string{
			"admin help [command]",
			"List all commands or show details for one command.",
		}, "\n\n"),
	}
	admin.fns["new"] = AdminFn{admin.new_cluster,
		"Create a cluster",
		strings.Join([]string{
			"admin new -cluster=NAME",
			"A directory of the given name will be created with internal directory structure initialized.",
		}, "\n\n"),
	}
	admin.fns["status"] = AdminFn{admin.status,
		"Print status of one or all clusters",
		strings.Join([]string{
			"admin status [-cluster=NAME]",
			"If no cluster name is specified, number of containers of each cluster is printed; otherwise the connection information for all containers in the given cluster will be displayed.",
		}, "\n\n"),
	}
	admin.fns["rethinkdb"] = AdminFn{admin.rethinkdb,
		"Start one or more rethinkdb containers",
		strings.Join([]string{
			"admin rethinkdb -cluster=NAME [-n=NUM]",
			"NUM rethinkdb containers will be started in cluster NAME. By default, NUM=1.",
		}, "\n\n"),
	}
	admin.fns["worker-exec"] = AdminFn{admin.worker_exec,
		"Start one worker with config",
		strings.Join([]string{
			"admin worker-exec -config=FILE",
			"Start a worker with a JSON config file.",
		}, "\n\n"),
	}
	admin.fns["workers"] = AdminFn{admin.workers,
		"Start one or more workers",
		strings.Join([]string{
			"admin workers -cluster=NAME [-foreach] [-port=PORT] [-n=NUM]",
			"Start one or more workers in cluster NAME. If foreach is set, one worker per database node will be started. [PORT,PORT+NUM) will be the range of port numbers for the newly created workers. By default, PORT=8080 and NUM=1.",
		}, "\n\n"),
	}
	admin.fns["nginx"] = AdminFn{admin.nginx,
		"Start one or more Nginx containers",
		strings.Join([]string{
			"admin nginx -cluster=NAME [-port=PORT] [-n=NUM]",
			"Start one or more Nginx nodes in cluster NAME. [PORT,PORT+NUM) will be the range of port numbers for the newly created Nginx nodes. By default, PORT=9080 and NUM=1. Run this command after running some workers.",
		}, "\n\n"),
	}
	admin.fns["kill"] = AdminFn{admin.kill,
		"Kill containers and processes of a cluster",
		strings.Join([]string{
			"admin kill -cluster=NAME",
		}, "\n\n"),
	}
	admin.fns["olstore-exec"] = AdminFn{admin.olstore_exec,
		"Start one olstore",
		strings.Join([]string{
			"admin olstore-exec [-port=PORT] [-ips=ADDR1,ADDR2,...]",
			"Start one olstore registry for storing lambda code. ips is a comma-separated list of rethinkdb IP addresses. By default, olstore listens on port 7080.",
		}, "\n\n"),
	}
	admin.fns["olstore"] = AdminFn{admin.olstore,
		"Start one olstore containers in a cluster",
		strings.Join([]string{
			"admin olstore -cluster=NAME [-port=PORT]",
			"Starts an olstore that connected with all databases in the cluster NAME.",
		}, "\n\n"),
	}
	admin.fns["upload"] = AdminFn{admin.upload,
		"Upload a file to registry",
		strings.Join([]string{"admin upload [-server=ADDR] [-name=HANDLER] [-file=PATH]",
			"The file will be uploaded to the server at ADDDR, and it will be bound with the name HANDLER on the server.",
		}, "\n\n"),
	}
	return &admin
}

func (admin *Admin) command(cmd string) {
	fn, ok := admin.fns[cmd]
	if !ok {
		admin.help()
		return
	}
	if err := fn.fn(); err != nil {
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
		if container.Labels[sbmanager.DOCKER_LABEL_CLUSTER] == cluster {
			cid := container.ID
			type_label := container.Labels[sbmanager.DOCKER_LABEL_TYPE]
			nodes[type_label] = append(nodes[type_label], cid)
		}
	}

	return nodes, nil
}

func (admin *Admin) help() error {
	if len(os.Args) > 2 && os.Args[1] == "help" {
		if fn, ok := admin.fns[os.Args[2]]; ok {
			fmt.Printf("%s\n\n%s\n", fn.doc, fn.doc_long)
			return nil
		}
	}
	fmt.Printf("Run %v <command> <args>\n", os.Args[0])
	fmt.Printf("\n")
	fmt.Printf("Commands:\n")
	cmds := make([]string, 0, len(admin.fns))
	for cmd, fn := range admin.fns {
		cmds = append(cmds, fmt.Sprintf("%-15s%s", cmd, fn.doc))
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

	if err := os.Mkdir(path.Join(*args.cluster, "workers"), 0700); err != nil {
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
		Worker_port:    "?",
		Cluster_name:   *args.cluster,
		Registry:       "local",
		Reg_dir:        args.RegistryPath(),
		Worker_dir:     args.WorkerPath("default"),
		Sandbox_config: map[string]interface{}{"processes": 10},
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

	if *args.cluster == "" {
		containers1, err := admin.client.ListContainers(docker.ListContainersOptions{})
		if err != nil {
			return err
		}

		other := 0
		node_counts := map[string]int{}

		for _, containers2 := range containers1 {
			label := containers2.Labels[sbmanager.DOCKER_LABEL_CLUSTER]
			if label != "" {
				node_counts[label] += 1
			} else {
				other += 1
			}
		}

		fmt.Printf("%d container(s) without OpenLambda labels\n\n", other)
		for cluster_name, count := range node_counts {
			fmt.Printf("%d container(s) belonging to cluster <%s>\n", count, cluster_name)
		}
		fmt.Printf("\n")
		fmt.Printf("Other clusters with no containers may exist without being listed.\n")
		fmt.Printf("\n")
		fmt.Printf("For info about a specific cluster, use -cluster=<cluster-dir>\n")
	} else {
		// print worker connection info
		logs, err := ioutil.ReadDir(path.Join(*args.cluster, "logs"))
		if err != nil {
			return err
		}
		fmt.Printf("Worker Pings:\n")
		for _, fi := range logs {
			if strings.HasSuffix(fi.Name(), ".pid") {
				name := fi.Name()[:len(fi.Name())-4]
				c, err := config.ParseConfig(args.ConfigPath(name))
				if err != nil {
					return err
				}

				url := fmt.Sprintf("http://localhost:%s/status", c.Worker_port)
				response, err := http.Get(url)
				if err != nil {
					fmt.Printf("  Could not send GET to %s\n", url)
					continue
				}
				defer response.Body.Close()
				body, err := ioutil.ReadAll(response.Body)
				if err != nil {
					fmt.Printf("  Failed to read body from GET to %s\n", url)
					continue
				}
				fmt.Printf("  %s => %s [%s]\n", url, body, response.Status)
			}
		}
		fmt.Printf("\n")

		// print containers
		fmt.Printf("Cluster containers:\n")
		nodes, err := admin.cluster_nodes(*args.cluster)
		if err != nil {
			return err
		}

		for typ, cids := range nodes {
			fmt.Printf("  %s containers:\n", typ)
			for _, cid := range cids {
				container, err := admin.client.InspectContainer(cid)
				if err != nil {
					return err
				}
				fmt.Printf("    %s [%s] => %s\n", container.Name, container.Config.Image, container.State.StateString())
			}
		}
	}

	return nil
}

func (admin *Admin) rethinkdb() error {
	args := NewCmdArgs()
	count := args.flags.Int("n", 1, "specify number of nodes to start")
	args.Parse(true)

	client := admin.client
	labels := map[string]string{}
	labels[sbmanager.DOCKER_LABEL_CLUSTER] = *args.cluster
	labels[sbmanager.DOCKER_LABEL_TYPE] = "db"

	image := "rethinkdb"

	// pull if not local
	_, err := admin.client.InspectImage(image)
	if err == docker.ErrNoSuchImage {
		fmt.Printf("Pulling RethinkDB image...\n")
		err := admin.client.PullImage(
			docker.PullImageOptions{
				Repository: image,
				Tag:        "latest", // TODO: fixed version?
			},
			docker.AuthConfiguration{},
		)
		if err != nil {
			return err
		}
	} else if err != nil {
		return err
	}

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
					Image:  image,
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

func (admin *Admin) worker_exec() error {
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
	portbase := args.flags.Int("port", 8080, "port range [port, port+n) will be used for containers")
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
			sandbox_config := c.Sandbox_config.(map[string]interface{})
			sandbox_config["db"] = "rethinkdb"
			sandbox_config["rethinkdb.host"] = container.NetworkSettings.IPAddress
			sandbox_config["rethinkdb.port"] = 28015
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
		conf.Worker_dir = args.WorkerPath(fmt.Sprintf("worker-%d", i))
		if err := os.Mkdir(conf.Worker_dir, 0700); err != nil {
			return err
		}
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
			"worker-exec",
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

func (admin *Admin) nginx() error {
	args := NewCmdArgs()
	portbase := args.flags.Int("port", 9080, "port range [port, port+n) will be used for workers")
	n := args.flags.Int("n", 1, "specify number of workers to start")
	args.Parse(true)

	image := "nginx"
	client := admin.client
	labels := map[string]string{}
	labels[sbmanager.DOCKER_LABEL_CLUSTER] = *args.cluster
	labels[sbmanager.DOCKER_LABEL_TYPE] = "balancer"

	// pull if not local
	_, err := admin.client.InspectImage(image)
	if err == docker.ErrNoSuchImage {
		fmt.Printf("Pulling nginx image...\n")
		err := admin.client.PullImage(
			docker.PullImageOptions{
				Repository: image,
				Tag:        "latest", // TODO: fixed version?
			},
			docker.AuthConfiguration{},
		)
		if err != nil {
			return err
		}
	} else if err != nil {
		return err
	}

	// config template
	nginx_conf := strings.Join([]string{
		"http {\n",
		"	upstream workers {\n",
	}, "")

	logs, err := ioutil.ReadDir(path.Join(*args.cluster, "logs"))
	if err != nil {
		return err
	}
	num_workers := 0
	for _, fi := range logs {
		if strings.HasSuffix(fi.Name(), ".pid") {
			name := fi.Name()[:len(fi.Name())-4]
			c, err := config.ParseConfig(args.ConfigPath(name))
			if err != nil {
				return err
			}
			line := fmt.Sprintf("		server localhost:%s;\n", c.Worker_port)
			nginx_conf += line
			num_workers += 1
		}
	}
	if num_workers == 0 {
		log.Fatal("No upstream worker found")
	}
	nginx_conf += strings.Join([]string{
		"	}\n",
		"\n",
		"	server {\n",
		"		listen %d;\n",
		"		location / {\n",
		"			proxy_pass http://workers;\n",
		"		}\n",
		"	}\n",
		"}\n",
		"\n",
		"events {\n",
		"	worker_connections 1024;\n",
		"}\n",
	}, "")

	// start containers
	for i := 0; i < *n; i++ {
		port := *portbase + i
		path := path.Join(*args.cluster, "config", fmt.Sprintf("nginx-%d.conf", i))
		if err := ioutil.WriteFile(path, []byte(fmt.Sprintf(nginx_conf, port)), 0644); err != nil {
			return err
		}

		// create and start container
		container, err := client.CreateContainer(
			docker.CreateContainerOptions{
				Config: &docker.Config{
					Image:  image,
					Labels: labels,
				},
				HostConfig: &docker.HostConfig{
					Binds:       []string{fmt.Sprintf("%s:%s", path, "/etc/nginx/nginx.conf")},
					NetworkMode: "host",
				},
			},
		)
		if err != nil {
			return err
		}
		if err := client.StartContainer(container.ID, nil); err != nil {
			return err
		}

		fmt.Printf("nginx listening on localhost:%d\n", port)
	}

	return nil
}

func (admin *Admin) kill() error {
	args := NewCmdArgs()
	args.Parse(true)

	client := admin.client

	nodes, err := admin.cluster_nodes(*args.cluster)
	if err != nil {
		return err
	}

	// kill containers in cluster
	for typ, cids := range nodes {
		for _, cid := range cids {
			container, err := client.InspectContainer(cid)
			if err != nil {
				return err
			}

			if container.State.Paused {
				fmt.Printf("Unpause container %v (%s)\n", cid, typ)
				if err := client.UnpauseContainer(cid); err != nil {
					fmt.Printf("%s\n", err.Error())
					fmt.Printf("Failed to unpause container %v (%s).  May require manual cleanup.\n", cid, typ)
				}
			}

			fmt.Printf("Kill container %v (%s)\n", cid, typ)
			opts := docker.KillContainerOptions{ID: cid}
			if err := client.KillContainer(opts); err != nil {
				fmt.Printf("%s\n", err.Error())
				fmt.Printf("Failed to kill container %v (%s).  May require manual cleanup.\n", cid, typ)
			}
		}
	}

	// kill worker processes in cluster
	logs, err := ioutil.ReadDir(path.Join(*args.cluster, "logs"))
	if err != nil {
		return err
	}
	for _, fi := range logs {
		if strings.HasSuffix(fi.Name(), ".pid") {
			data, err := ioutil.ReadFile(args.LogPath(fi.Name()))
			if err != nil {
				return err
			}
			pidstr := string(data)
			pid, err := strconv.Atoi(pidstr)
			if err != nil {
				return err
			}
			fmt.Printf("Kill worker process with PID %d\n", pid)
			p, err := os.FindProcess(pid)
			if err != nil {
				fmt.Printf("%s\n", err.Error())
				fmt.Printf("Failed to find worker process with PID %d.  May require manual cleanup.\n", pid)
			}
			if err := p.Kill(); err != nil {
				fmt.Printf("%s\n", err.Error())
				fmt.Printf("Failed to kill process with PID %d.  May require manual cleanup.\n", pid)
			}
		}
	}

	return nil
}

func (admin *Admin) olstore_exec() error {
	flags := flag.NewFlagSet("flag", flag.ExitOnError)
	port := flags.Int("port", 7080, "port to push/pull lambdas")
	ips := flags.String("ips", "", "comma-separated rethinkdb addrs")
	flags.Parse(os.Args[2:])

	pushs := registry.InitPushServer(*port, strings.Split(*ips, ","))
	pushs.Run()
	return fmt.Errorf("Push Server Crashed\n")
}

func (admin *Admin) olstore() error {
	args := NewCmdArgs()
	port := args.flags.Int("port", 7080, "port to push/pull lambdas")
	args.Parse(true)

	// get rethinkdb addrs
	nodes, err := admin.cluster_nodes(*args.cluster)
	if err != nil {
		return err
	}

	ips := []string{}
	for _, cid := range nodes["db"] {
		container, err := admin.client.InspectContainer(cid)
		if err != nil {
			return err
		}

		ips = append(ips, container.NetworkSettings.IPAddress)
	}

	if len(ips) == 0 {
		fmt.Printf("No rethinkdb instances running this cluster\n")
		return nil
	}

	// stdout+stderr both go to log
	log_path := args.LogPath(fmt.Sprintf("olstore.out"))
	f, err := os.Create(log_path)
	if err != nil {
		return err
	}
	attr := os.ProcAttr{
		Files: []*os.File{nil, f, f},
	}
	cmd := []string{
		os.Args[0],
		"olstore-exec",
		"-ips=" + strings.Join(ips, ","),
		fmt.Sprintf("-port=%v", *port),
	}
	proc, err := os.StartProcess(os.Args[0], cmd, &attr)
	if err != nil {
		return err
	}

	pidpath := args.PidPath("olstore")
	if err := ioutil.WriteFile(pidpath, []byte(fmt.Sprintf("%d", proc.Pid)), 0644); err != nil {
		return err
	}

	fmt.Printf("Started olstore: pid %d, port %v, log at %s\n", proc.Pid, *port, log_path)
	return nil
}

func (admin *Admin) upload() error {
	args := NewCmdArgs()
	server := args.flags.String("server", "", "olstore addr")
	name := args.flags.String("name", "", "handler name")
	fname := args.flags.String("file", "", "file path")
	args.Parse(false)
	registry.Push(*server, *name, *fname)
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
