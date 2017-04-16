package main

import (
	"bufio"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"path"
	"path/filepath"
	"strconv"
	"strings"

	docker "github.com/fsouza/go-dockerclient"
	dutil "github.com/open-lambda/open-lambda/worker/dockerutil"

	"github.com/open-lambda/open-lambda/registry"
	"github.com/open-lambda/open-lambda/worker/config"
	pip "github.com/open-lambda/open-lambda/worker/pip-manager"
	"github.com/open-lambda/open-lambda/worker/server"
	"github.com/urfave/cli"
)

var client *docker.Client

// TODO: notes about setup process
// TODO: notes about creating a directory in local
// TODO: docker registry setup

// Parse parses the cluster name. If required is true but
// the cluster name is empty, program will exit with an error.
func parseCluster(cluster string, required bool) string {
	if cluster != "" {
		if abscluster, err := filepath.Abs(cluster); err != nil {
			log.Fatal("failed to get abs cluster dir: ", err)
		} else {
			return abscluster
		}
	} else if required {
		log.Fatal("please specify a cluster directory")
	}
	return cluster
}

// logPath gets the logging directory of the cluster
func logPath(cluster string, name string) string {
	return path.Join(cluster, "logs", name)
}

// workerPath gets the worker directory of the cluster
func workerPath(cluster string, name string) string {
	return path.Join(cluster, "workers", name)
}

// pidPath gets the path of the pid file of a process in the container
func pidPath(cluster string, name string) string {
	return path.Join(cluster, "logs", name+".pid")
}

// configPath gets the path of a JSON config file in the cluster
func configPath(cluster string, name string) string {
	return path.Join(cluster, "config", name+".json")
}

// BasePath gets location for storing base handler files (e.g., Ubuntu
// install files) for cgroup mode
func basePath(cluster string) string {
	return path.Join(cluster, "base")
}

// templatePath gets the config template directory of the cluster
func templatePath(cluster string) string {
	return configPath(cluster, "template")
}

// registryPath gets the registry directory of the cluster
func registryPath(cluster string) string {
	return path.Join(cluster, "registry")
}

// clusterNodes finds all docker containers belongs to a cluster and returns
// a mapping from the type of the container to its container ID.
func clusterNodes(cluster string) (map[string]([]string), error) {
	nodes := map[string]([]string){}

	containers, err := client.ListContainers(docker.ListContainersOptions{})
	if err != nil {
		return nil, err
	}

	for _, container := range containers {
		if container.Labels[dutil.DOCKER_LABEL_CLUSTER] == cluster {
			cid := container.ID
			type_label := container.Labels[dutil.DOCKER_LABEL_TYPE]
			nodes[type_label] = append(nodes[type_label], cid)
		}
	}

	return nodes, nil
}

// newCluster corresponds to the "new" command of the admin tool.
func newCluster(ctx *cli.Context) error {
	cluster := parseCluster(ctx.String("cluster"), true)

	if err := os.Mkdir(cluster, 0700); err != nil {
		return err
	}

	if err := os.Mkdir(path.Join(cluster, "logs"), 0700); err != nil {
		return err
	}

	if err := os.Mkdir(path.Join(cluster, "workers"), 0700); err != nil {
		return err
	}

	if err := os.Mkdir(registryPath(cluster), 0700); err != nil {
		return err
	}

	if err := os.Mkdir(basePath(cluster), 0700); err != nil {
		return err
	}

	// config dir and template
	if err := os.Mkdir(path.Join(cluster, "config"), 0700); err != nil {
		return err
	}
	c := &config.Config{
		Worker_port:    "?",
		Cluster_name:   cluster,
		Registry:       "local",
		Sandbox:        "docker",
		Reg_dir:        registryPath(cluster),
		Worker_dir:     workerPath(cluster, "default"),
		Sandbox_config: map[string]interface{}{"processes": 10},
	}
	if err := c.Defaults(); err != nil {
		return err
	}
	if err := c.Save(templatePath(cluster)); err != nil {
		return err
	}

	fmt.Printf("Cluster Directory: %s\n\n", cluster)
	fmt.Printf("Worker Defaults: \n%s\n\n", c.DumpStr())
	fmt.Printf("You may now start a cluster using the \"workers\" command\n")

	return nil
}

// status corresponds to the "status" command of the admin tool.
func status(ctx *cli.Context) error {
	cluster := parseCluster(ctx.String("cluster"), false)

	if cluster == "" {
		containers1, err := client.ListContainers(docker.ListContainersOptions{})
		if err != nil {
			return err
		}

		other := 0
		node_counts := map[string]int{}

		for _, containers2 := range containers1 {
			label := containers2.Labels[dutil.DOCKER_LABEL_CLUSTER]
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
		logs, err := ioutil.ReadDir(path.Join(cluster, "logs"))
		if err != nil {
			return err
		}
		fmt.Printf("Worker Pings:\n")
		for _, fi := range logs {
			if strings.HasSuffix(fi.Name(), ".pid") {
				name := fi.Name()[:len(fi.Name())-4]
				c, err := config.ParseConfig(configPath(cluster, name))
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
		nodes, err := clusterNodes(cluster)
		if err != nil {
			return err
		}

		for typ, cids := range nodes {
			fmt.Printf("  %s containers:\n", typ)
			for _, cid := range cids {
				container, err := client.InspectContainer(cid)
				if err != nil {
					return err
				}
				fmt.Printf("    %s [%s] => %s\n", container.Name, container.Config.Image, container.State.StateString())
			}
		}
	}

	return nil
}

// rethinkdb corresponds to the "rethinkdb" command of the admin tool.
func rethinkdb(ctx *cli.Context) error {
	cluster := parseCluster(ctx.String("cluster"), true)
	count := ctx.Int("num-nodes")

	labels := map[string]string{}
	labels[dutil.DOCKER_LABEL_CLUSTER] = cluster
	labels[dutil.DOCKER_LABEL_TYPE] = "db"

	image := "rethinkdb"

	// pull if not local
	_, err := client.InspectImage(image)
	if err == docker.ErrNoSuchImage {
		fmt.Printf("Pulling RethinkDB image...\n")
		err := client.PullImage(
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

	for i := 0; i < count; i++ {
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

// worker_exec corresponds to the "worker-exec" command of the admin tool.
func worker_exec(ctx *cli.Context) error {
	conf := ctx.String("config")

	if conf == "" {
		fmt.Printf("Please specify a json config file\n")
		return nil
	}

	server.Main(conf)
	return nil
}

// workers corresponds to the "workers" command of the admin tool.
//
// The JSON config in the cluster template directory will be populated for each
// worker, and their pid will be written to the log directory. worker_exec will
// be called to run the worker processes.
func workers(ctx *cli.Context) error {
	cluster := parseCluster(ctx.String("cluster"), true)
	foreach := ctx.Bool("foreach")
	portbase := ctx.Int("port")
	n := ctx.Int("num-workers")

	worker_confs := []*config.Config{}
	if foreach {
		nodes, err := clusterNodes(cluster)
		if err != nil {
			return err
		}

		// start one worker per db shard
		for _, cid := range nodes["db"] {
			container, err := client.InspectContainer(cid)
			if err != nil {
				return err
			}

			fmt.Printf("DB node: %v\n", container.NetworkSettings.IPAddress)

			c, err := config.ParseConfig(templatePath(cluster))
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
		for i := 0; i < n; i++ {
			c, err := config.ParseConfig(templatePath(cluster))
			if err != nil {
				return err
			}
			worker_confs = append(worker_confs, c)
		}
	}

	for i, conf := range worker_confs {
		conf_path := configPath(cluster, fmt.Sprintf("worker-%d", i))
		conf.Worker_port = fmt.Sprintf("%d", portbase+i)
		conf.Worker_dir = workerPath(cluster, fmt.Sprintf("worker-%d", i))
		if err := os.Mkdir(conf.Worker_dir, 0700); err != nil {
			return err
		}
		if err := conf.Save(conf_path); err != nil {
			return err
		}

		// stdout+stderr both go to log
		log_path := logPath(cluster, fmt.Sprintf("worker-%d.out", i))
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

		pidpath := pidPath(cluster, fmt.Sprintf("worker-%d", i))
		if err := ioutil.WriteFile(pidpath, []byte(fmt.Sprintf("%d", proc.Pid)), 0644); err != nil {
			return err
		}

		fmt.Printf("Started worker: pid %d, port %s, log at %s\n", proc.Pid, conf.Worker_port, log_path)
	}

	return nil
}

// nginx corresponds to the "nginx" command of the admin tool.
func nginx(ctx *cli.Context) error {
	cluster := parseCluster(ctx.String("cluster"), true)
	portbase := ctx.Int("port")
	n := ctx.Int("num-nodes")

	image := "nginx"
	labels := map[string]string{}
	labels[dutil.DOCKER_LABEL_CLUSTER] = cluster
	labels[dutil.DOCKER_LABEL_TYPE] = "balancer"

	// pull if not local
	_, err := client.InspectImage(image)
	if err == docker.ErrNoSuchImage {
		fmt.Printf("Pulling nginx image...\n")
		err := client.PullImage(
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

	logs, err := ioutil.ReadDir(path.Join(cluster, "logs"))
	if err != nil {
		return err
	}
	num_workers := 0
	for _, fi := range logs {
		if strings.HasSuffix(fi.Name(), ".pid") {
			name := fi.Name()[:len(fi.Name())-4]
			c, err := config.ParseConfig(configPath(cluster, name))
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
	for i := 0; i < n; i++ {
		port := portbase + i
		path := path.Join(cluster, "config", fmt.Sprintf("nginx-%d.conf", i))
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

// kill corresponds to the "kill" command of the admin tool.
func kill(ctx *cli.Context) error {
	cluster := parseCluster(ctx.String("cluster"), true)

	nodes, err := clusterNodes(cluster)
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
	logs, err := ioutil.ReadDir(path.Join(cluster, "logs"))
	if err != nil {
		return err
	}
	for _, fi := range logs {
		if strings.HasSuffix(fi.Name(), ".pid") {
			data, err := ioutil.ReadFile(logPath(cluster, fi.Name()))
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

// manage cgroups directly
func cgroup_sandbox(ctx *cli.Context) error {
	cluster := parseCluster(ctx.String("cluster"), true)

	// create a base directory to run cgroup containers
	baseDir := path.Join(basePath(cluster), "lambda")
	err := dutil.DumpDockerImage(client, "lambda", baseDir)
	if err != nil {
		return err
	}

	// configure template to use cgroup containers
	c, err := config.ParseConfig(templatePath(cluster))
	if err != nil {
		return err
	}
	c.Sandbox = "cgroup"
	c.Cgroup_base = baseDir
	if err := c.Save(templatePath(cluster)); err != nil {
		return err
	}

	return nil
}

// olstore_exec corresponds to the "olstore-exec" command of the admin tool.
func olstore_exec(ctx *cli.Context) error {
	port := ctx.Int("port")
	ips := ctx.String("ips")

	pushs := registry.InitPushServer(port, strings.Split(ips, ","))
	pushs.Run()
	return fmt.Errorf("Push Server Crashed\n")
}

// olstore corresponds to the "olstore" commanf of the admin tool. It starts an
// olstore that listens to a given port and connects with all rethink db
// instances of the cluster. It calls olstore_exec to starts the olstore.
func olstore(ctx *cli.Context) error {
	cluster := parseCluster(ctx.String("cluster"), true)
	port := ctx.Int("port")

	// get rethinkdb addrs
	nodes, err := clusterNodes(cluster)
	if err != nil {
		return err
	}

	ips := []string{}
	for _, cid := range nodes["db"] {
		container, err := client.InspectContainer(cid)
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
	log_path := logPath(cluster, fmt.Sprintf("olstore.out"))
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
		fmt.Sprintf("-port=%v", port),
	}
	proc, err := os.StartProcess(os.Args[0], cmd, &attr)
	if err != nil {
		return err
	}

	pidpath := pidPath(cluster, "olstore")
	if err := ioutil.WriteFile(pidpath, []byte(fmt.Sprintf("%d", proc.Pid)), 0644); err != nil {
		return err
	}

	fmt.Printf("Started olstore: pid %d, port %v, log at %s\n", proc.Pid, port, log_path)
	return nil
}

// uploads corresponds to the "upload" command of the admin tool.
func upload(ctx *cli.Context) error {
	server := ctx.String("server")
	name := ctx.String("name")
	fname := ctx.String("file")
	registry.Push(server, name, fname)
	return nil
}

// installs requirements to an unpack-only pip mirror
func install(ctx *cli.Context) error {
	index := ctx.String("index")
	target := ctx.String("target")
	reqsFile := ctx.String("reqs")
	reqArgs := ctx.Args()

	reqs := []string{}

	reqSet := map[string]bool{}
	if reqsFile != "" {
		if file, err := os.Open(reqsFile); err != nil {
			return err
		} else {
			defer file.Close()
			scanner := bufio.NewScanner(file)
			for scanner.Scan() {
				normalized := strings.ToLower(scanner.Text())
				if _, ok := reqSet[normalized]; !ok {
					reqSet[normalized] = true
					reqs = append(reqs, normalized)
				}
			}
		}
	}

	for _, req := range reqArgs {
		normalized := strings.ToLower(req)
		if _, ok := reqSet[normalized]; !ok {
			reqSet[normalized] = true
			reqs = append(reqs, normalized)
		}
	}

	if m, err := pip.NewUnpackMirrorServer(index, target); err != nil {
		log.Fatal("fail to create unpack mirror server: ", err)
	} else if remains, err := m.Prepare(reqs); err != nil {
		log.Fatal("fail to prepare unpack mirror: ", err)
	} else {
		remainsLog := filepath.Join(target, "remains.log")
		if err := ioutil.WriteFile(remainsLog, []byte(strings.Join(remains, "\n")), 0644); err != nil {
			log.Fatal("fail to write logs: ", err)
		}
		if len(remains) == 0 {
			log.Printf("All packages installed successfully\n")
		} else {
			log.Printf("Fail to installed %d out of %d packages. List written to %s\n", len(remains), len(reqs), remainsLog)
		}
	}
	return nil
}

// main runs the admin tool
func main() {
	if c, err := docker.NewClientFromEnv(); err != nil {
		log.Fatal("failed to get docker client: ", err)
	} else {
		client = c
	}

	cli.CommandHelpTemplate = `NAME:
   {{.HelpName}} - {{if .Description}}{{.Description}}{{else}}{{.Usage}}{{end}}
USAGE:
   {{if .UsageText}}{{.UsageText}}{{else}}{{.HelpName}} command{{if .VisibleFlags}} [command options]{{end}} {{if .ArgsUsage}}{{.ArgsUsage}}{{else}}[arguments...]{{end}}{{end}}
COMMANDS:{{range .VisibleCategories}}{{if .Name}}
   {{.Name}}:{{end}}{{range .VisibleCommands}}
     {{join .Names ", "}}{{"\t"}}{{.Usage}}{{end}}
{{end}}{{if .VisibleFlags}}
OPTIONS:
   {{range .VisibleFlags}}{{.}}
   {{end}}{{end}}
`
	app := cli.NewApp()
	app.Usage = "Admin tool for Open-Lambda"
	app.UsageText = "admin COMMAND [ARG...]"
	app.ArgsUsage = "ArgsUsage"
	app.EnableBashCompletion = true
	app.HideVersion = true
	clusterFlag := cli.StringFlag{
		Name:  "cluster",
		Usage: "The `NAME` of the cluster directory",
	}
	app.Commands = []cli.Command{
		cli.Command{
			Name:        "new",
			Usage:       "Create a cluster",
			UsageText:   "admin new --cluseter=NAME",
			Description: "A cluster directory of the given name will be created with internal structure initialized.",
			Flags:       []cli.Flag{clusterFlag},
			Action:      newCluster,
		},
		cli.Command{
			Name:        "status",
			Usage:       "Print status of one or all clusters",
			UsageText:   "admin status [--cluster=NAME]",
			Description: "If no cluster name is specified, number of containers of each cluster is printed; otherwise the connection information for all containers in the given cluster will be displayed.",
			Flags:       []cli.Flag{clusterFlag},
			Action:      status,
		},
		cli.Command{
			Name:        "workers",
			Usage:       "Start one or more worker servers",
			UsageText:   "admin workers --cluster=NAME [--foreach] [-p|--port=PORT] [-n|--num-workers=NUM]",
			Description: "Start one or more workers in cluster using the same config template.",
			Flags: []cli.Flag{
				clusterFlag,
				cli.BoolFlag{
					Name:  "foreach",
					Usage: "Start one worker per db instance",
				},
				cli.IntFlag{
					Name:  "port, p",
					Usage: "Port range [`PORT`, `PORT`+n) will be used for workers",
					Value: 8080,
				},
				cli.IntFlag{
					Name:  "num-workers, n",
					Usage: "To start `NUM` workers",
					Value: 1,
				},
			},
			Action: workers,
		},
		cli.Command{
			Name:        "worker-exec",
			Usage:       "Start one worker with config",
			UsageText:   "admin worker-exec -c|--config=FILE",
			Description: "Start a worker with a JSON config file.",
			Flags: []cli.Flag{
				cli.StringFlag{
					Name:  "config, c",
					Usage: "Load worker configuration from `FILE`",
				},
			},
			Action: worker_exec,
		},
		cli.Command{
			Name:        "rethinkdb",
			Usage:       "Start one or more rethinkdb nodes",
			UsageText:   "admin rethinkdb --cluster=NAME [-n|--num-nodes=NUM]",
			Description: "A cluster of rethinkdb intances will be started with default ip and port (172.17.0.2:28015).",
			Flags: []cli.Flag{
				clusterFlag,
				cli.IntFlag{
					Name:  "num-nodes, n",
					Usage: "To start `NUM` rethinkdb nodes",
					Value: 1,
				},
			},
			Action: rethinkdb,
		},
		cli.Command{
			Name:        "nginx",
			Usage:       "Start one or more Nginx containers",
			UsageText:   "admin nginx --cluster=NAME [-p|--port=PORT] [-n|--num-nodes=NUM]",
			Description: "Start one or more Nginx nodes in cluster. Run this command after starting some workers.",
			Flags: []cli.Flag{
				clusterFlag,
				cli.IntFlag{
					Name:  "port, p",
					Usage: "Port range [`PORT`, `PORT`+n) will be used for containers",
					Value: 9080,
				},
				cli.IntFlag{
					Name:  "num-nodes, n",
					Usage: "To start `NUM` Nginx nodes",
					Value: 1,
				},
			},
			Action: nginx,
		},
		cli.Command{
			Name:        "olstore",
			Usage:       "Start one olstore containers in a cluster",
			UsageText:   "admin olstore --cluster=NAME [-p|--port=PORT]",
			Description: "Start one olstore that connectes with all databases in the cluster.",
			Flags: []cli.Flag{
				clusterFlag,
				cli.IntFlag{
					Name:  "port, p",
					Usage: "Push/pull lambdas at `PORT`",
					Value: 7080,
				},
			},
			Action: olstore,
		},
		cli.Command{
			Name:        "olstore-exec",
			Usage:       "Start one olstore",
			UsageText:   "admin olstore-exec [-p|-port=PORT] [--ips=ADDRS]",
			Description: "Start one olstore registry for storing lambda code.",
			Flags: []cli.Flag{
				cli.IntFlag{
					Name:  "port, p",
					Usage: "Push/pull lambdas at `PORT`",
					Value: 7080,
				},
				cli.StringFlag{
					Name:  "ips",
					Usage: "comma-separated rethinkdb `ADDRS`",
				},
			},
			Action: olstore_exec,
		},
		cli.Command{
			Name:        "upload",
			Usage:       "Upload a file to registry",
			UsageText:   "admin upload --cluster=NAME [--server=ADDR] [--handler=NAME] [--file=PATH]",
			Description: "Upload a file to registry. The file must be a compressed file.",
			Flags: []cli.Flag{
				clusterFlag,
				cli.StringFlag{
					Name:  "server",
					Usage: "`ADDR` of olstore that receives the file",
				},
				cli.StringFlag{
					Name:  "handler",
					Usage: "`NAME` of the handler",
				},
				cli.StringFlag{
					Name:  "file",
					Usage: "`PATH` to the file",
				},
			},
			Action: upload,
		},
		cli.Command{
			Name:        "cgroup-sandbox",
			Usage:       "Use cgroups directly",
			UsageText:   "admin cgroup-sandbox --cluster=NAME",
			Description: "Creates a root file system in the cluster directory and configures OpenLambda to use cgroup containers",
			Flags: []cli.Flag{
				clusterFlag,
			},
			Action: cgroup_sandbox,
		},
		cli.Command{
			Name:      "kill",
			Usage:     "Kill containers and processes in a cluster",
			UsageText: "admin kill --cluster=NAME",
			Flags:     []cli.Flag{clusterFlag},
			Action:    kill,
		},
		cli.Command{
			Name:      "install",
			Usage:     "Install packages to an unpack-only Pip mirror",
			UsageText: "admin install [-i|--index=INDEX] [-t|--target=TARGET] [-r|--reqs=FILE] [req...]",
			Flags: []cli.Flag{
				cli.StringFlag{
					Name:  "index, i",
					Usage: "`INDEX` of Pip mirror",
				},
				cli.StringFlag{
					Name:  "target, t",
					Usage: "`TARGET` directory to install the unpack mirror",
					Value: "/tmp/.open_lambda/pip",
				},
				cli.StringFlag{
					Name:  "reqs, r",
					Usage: "`FILE` of requirements to install",
				},
			},
			ArgsUsage: "Requirements to install",
			Action:    install,
		},
	}
	app.Run(os.Args)
}
