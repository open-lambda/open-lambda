package main

import (
	"flag"
	"fmt"
	docker "github.com/fsouza/go-dockerclient"
	"log"
	"os"
)

type Admin struct {
	client *docker.Client
	fns    map[string]AdminFn
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
	admin.fns["status"] = admin.status
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

func (admin *Admin) status() error {
	flags := flag.NewFlagSet("flag", flag.ExitOnError)
	cluster := flags.String("cluster", "", "give a cluster name")
	flags.Parse(os.Args[2:])

	containers1, err := admin.client.ListContainers(docker.ListContainersOptions{})
	if err != nil {
		return err
	}

	if *cluster == "" {
		node_counts := map[string]int{}

		for _, containers2 := range containers1 {
			for label, value := range containers2.Labels {
				if label == "ol.cluster" {
					node_counts[value] += 1
				}
			}
		}

		fmt.Printf("Clusters:\n")
		for cluster_name, count := range node_counts {
			fmt.Printf("  <%s> (%d nodes)\n", cluster_name, count)
		}
		fmt.Printf("\n")
		fmt.Printf("For info about a specific cluster, use -cluster=<cluster-name>\n")
	} else {
		fmt.Printf("Nodes in %s cluster:\n", *cluster)
		for _, containers2 := range containers1 {
			if containers2.Labels["ol.cluster"] == *cluster {
				name := containers2.Names[0]
				oltype := containers2.Labels["ol.type"]
				fmt.Printf("  <%s> (%s)\n", name, oltype)
			}
		}
	}

	return nil
}

func (admin *Admin) kill() error {
	client := admin.client
	flags := flag.NewFlagSet("flag", flag.ExitOnError)
	cluster := flags.String("cluster", "", "give a cluster name")
	flags.Parse(os.Args[2:])

	if *cluster == "" {
		fmt.Printf("Please specify a cluster\n")
		return nil
	}

	containers1, err := client.ListContainers(docker.ListContainersOptions{})
	if err != nil {
		return err
	}

	for _, containers2 := range containers1 {
		if containers2.Labels["ol.cluster"] == *cluster {
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
