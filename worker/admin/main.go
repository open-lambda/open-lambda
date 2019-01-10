package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"path"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"
	"time"

	docker "github.com/fsouza/go-dockerclient"
	minio "github.com/minio/minio-go"
	dutil "github.com/open-lambda/open-lambda/worker/dockerutil"

	"github.com/open-lambda/open-lambda/worker/config"
	"github.com/open-lambda/open-lambda/worker/server"
	"github.com/urfave/cli"
)

var client *docker.Client

const OLCONF = "/.ol.conf"

// TODO: notes about setup process
// TODO: notes about creating a directory in local

// Parse parses the cluster name. If required is true but
// the cluster name is empty, program will exit with an error.
func parseCluster(cluster string, required bool) string {
	if cluster == "" {
		buf, err := ioutil.ReadFile(OLCONF)
		if err != nil {
			log.Fatalf("no cluster directory specified and failed to read %s", OLCONF)
		}

		cluster = strings.TrimSpace(string(buf))
	}

	abscluster, err := filepath.Abs(cluster)
	if err != nil {
		log.Fatal("failed to get abs cluster dir: ", err)
	}

	return abscluster
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
// install files) for sock mode
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

// packagesPath gets the packages directory of the cluster
func packagesPath(cluster string) string {
	return path.Join(cluster, "packages")
}

// cachePath gets the import-cache directory of the cluster
func cachePath(cluster string) string {
	return path.Join(cluster, "import-cache")
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

	if err := os.Mkdir(packagesPath(cluster), 0700); err != nil {
		return err
	}

	if err := os.Mkdir(cachePath(cluster), 0700); err != nil {
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
		Registry_dir:   registryPath(cluster),
		Pkgs_dir:       packagesPath(cluster),
		Worker_dir:     workerPath(cluster, "default"),
		Sandbox_config: map[string]interface{}{"processes": 10},
	}
	if err := c.Defaults(); err != nil {
		return err
	}
	if err := c.Save(templatePath(cluster)); err != nil {
		return err
	}

	dump_sock_image(ctx)

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

		return nil
	}

	var pingErr bool

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
				pingErr = true
				continue
			}
			defer response.Body.Close()
			body, err := ioutil.ReadAll(response.Body)
			if err != nil {
				fmt.Printf("  Failed to read body from GET to %s\n", url)
				pingErr = true
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
	fmt.Printf("\n")

	if pingErr {
		return fmt.Errorf("At least one worker failed the status check")
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
			if err := p.Signal(syscall.SIGINT); err != nil {
				fmt.Printf("%s\n", err.Error())
				fmt.Printf("Failed to kill process with PID %d.  May require manual cleanup.\n", pid)
			}
		}
	}

	return nil
}

func dump_sock_image(ctx *cli.Context) (err error) {
	cluster := parseCluster(ctx.String("cluster"), true)

	// create a base directory to run sock handlers
	err = dutil.DumpDockerImage(client, "lambda", basePath(cluster))
	if err != nil {
		return err
	} else if err = write_dns(basePath(cluster)); err != nil {
		return err
	}

	// configure template to use sock containers
	c, err := config.ParseConfig(templatePath(cluster))
	if err != nil {
		return err
	}
	c.SOCK_base_path = basePath(cluster)
	if err := c.Save(templatePath(cluster)); err != nil {
		return err
	}

	return nil
}

// need this because Docker containers don't have a dns server in /etc/resolv.conf
func write_dns(rootDir string) error {
	dnsPath := filepath.Join(rootDir, "etc/resolv.conf")
	return ioutil.WriteFile(dnsPath, []byte("nameserver 8.8.8.8\n"), 0644)
}

// Starts a container running Minio, logs its AccessKey and SecretKey.
func registry(ctx *cli.Context) error {
	cluster := parseCluster(ctx.String("cluster"), true)

	access_key := ctx.String("access-key")
	secret_key := ctx.String("secret-key")
	port := ctx.Int("port")
	image := "minio/minio"

	_, err := client.InspectImage(image)
	if err == docker.ErrNoSuchImage {
		fmt.Printf("Pulling Minio image...\n")
		err := client.PullImage(
			docker.PullImageOptions{
				Repository: image,
				Tag:        "latest",
			},
			docker.AuthConfiguration{},
		)
		if err != nil {
			return err
		}
	} else if err != nil {
		return err
	}

	ports := map[docker.Port][]docker.PortBinding{"9000/tcp": []docker.PortBinding{docker.PortBinding{HostIP: "0.0.0.0", HostPort: fmt.Sprintf("%d", port)}}}
	cmd := []string{"server", "/data"}
	volumes := []string{"/mnt/data:/data", "/mnt/config:/root/.minio"}
	labels := map[string]string{
		dutil.DOCKER_LABEL_CLUSTER: cluster,
		dutil.DOCKER_LABEL_TYPE:    "registry",
	}
	env := []string{
		fmt.Sprintf("MINIO_ACCESS_KEY=%s", access_key),
		fmt.Sprintf("MINIO_SECRET_KEY=%s", secret_key),
	}

	// create and start container
	container, err := client.CreateContainer(
		docker.CreateContainerOptions{
			Config: &docker.Config{
				Cmd:    cmd,
				Env:    env,
				Image:  image,
				Labels: labels,
			},
			HostConfig: &docker.HostConfig{
				Binds:        volumes,
				PortBindings: ports,
			},
		},
	)
	if err != nil {
		return err
	}
	if err := client.StartContainer(container.ID, container.HostConfig); err != nil {
		return err
	}

	regClient, err := minio.New(fmt.Sprintf("localhost:%d", port), access_key, secret_key, false)
	if err != nil {
		return err
	}

	start := time.Now()
	var bucketErr error
	for {
		if time.Since(start) > 10*time.Second {
			return fmt.Errorf("failed to connect to bucket after 10s :: %v", bucketErr)
		}

		if exists, err := regClient.BucketExists(config.REGISTRY_BUCKET); err != nil {
			bucketErr = err
			continue
		} else if !exists {
			if err := regClient.MakeBucket(config.REGISTRY_BUCKET, "us-east-1"); err != nil {
				bucketErr = err
				continue
			}
		} else {
			break
		}
	}

	c, err := config.ParseConfig(templatePath(cluster))
	if err != nil {
		return err
	}
	c.Registry = "remote"
	c.Registry_access_key = access_key
	c.Registry_secret_key = secret_key
	if err := c.Save(templatePath(cluster)); err != nil {
		return err
	}

	return nil
}

// uploads corresponds to the "upload" command of the admin tool.
func upload(ctx *cli.Context) error {
	access_key := ctx.String("access-key")
	secret_key := ctx.String("secret-key")
	address := ctx.String("address")
	handler := ctx.String("handler")
	file := ctx.String("file")

	regClient, err := minio.New(address, access_key, secret_key, false)
	if err != nil {
		return err
	}

	opts := minio.PutObjectOptions{ContentType: "application/gzip", ContentEncoding: "binary"}
	if _, err := regClient.FPutObject(config.REGISTRY_BUCKET, handler, file, opts); err != nil {
		return err
	}

	return nil
}

// setconf sets a configuration option in the cluster's template
func setconf(ctx *cli.Context) error {
	cluster := parseCluster(ctx.String("cluster"), false)

	if len(ctx.Args()) != 1 {
		log.Fatal("Usage: admin setconf <json_options>")
	}

	if c, err := config.ParseConfig(templatePath(cluster)); err != nil {
		return err
	} else if err := json.Unmarshal([]byte(ctx.Args()[0]), c); err != nil {
		return fmt.Errorf("failed to set config options :: %v", err)
	} else if err := c.Save(templatePath(cluster)); err != nil {
		return err
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
			UsageText:   "admin new --cluster=NAME",
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
			Name:        "registry",
			Usage:       "Start the code registry.",
			UsageText:   "admin registry [-p|-port=PORT] [--access-key=KEY] [--secret-key=KEY]",
			Description: "Start the code reigstry.",
			Flags: []cli.Flag{
				clusterFlag,
				cli.StringFlag{
					Name:  "access-key",
					Usage: "Minio access key",
				},
				cli.StringFlag{
					Name:  "secret-key",
					Usage: "Minio secret key",
				},
				cli.IntFlag{
					Name:  "port, p",
					Usage: "Push/pull lambdas at `PORT`",
					Value: 9000,
				},
			},
			Action: registry,
		},
		cli.Command{
			Name:        "upload",
			Usage:       "Upload handler code to the registry",
			UsageText:   "admin upload --cluster=NAME --handler=NAME --file=PATH [--access-key=KEY] [--secret-key=KEY]",
			Description: "Upload a file to registry. The file must be a tarball.",
			Flags: []cli.Flag{
				clusterFlag,
				cli.StringFlag{
					Name:  "access-key",
					Usage: "Minio access key",
				},
				cli.StringFlag{
					Name:  "secret-key",
					Usage: "Minio secret key",
				},
				cli.StringFlag{
					Name:  "address",
					Usage: "Address+port of remote Minio server",
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
			Name:      "kill",
			Usage:     "Kill containers and processes in a cluster",
			UsageText: "admin kill --cluster=NAME",
			Flags:     []cli.Flag{clusterFlag},
			Action:    kill,
		},
		cli.Command{
			Name:      "setconf",
			Usage:     "Set a configuration option in the cluster's template.",
			UsageText: "admin setconf [--cluster=NAME] options (options is JSON string)",
			Flags:     []cli.Flag{clusterFlag},
			Action:    setconf,
		},
	}
	err := app.Run(os.Args)
	if err != nil {
		log.Fatal(err)
	}
}
