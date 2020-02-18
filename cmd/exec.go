package cmd

import (
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/docker/docker/pkg/term"
	"github.com/shipyard-run/shipyard/pkg/clients"
	"github.com/shipyard-run/shipyard/pkg/config"
	"github.com/shipyard-run/shipyard/pkg/utils"
	"github.com/spf13/cobra"
)

var cluster string

var execCmd = &cobra.Command{
	Use:                "exec",
	Short:              "Execute a command in a Resource",
	Long:               `Execute a command in a Resource or start a Tools resource and execute`,
	Args:               cobra.MinimumNArgs(1),
	DisableFlagParsing: true,
	Run: func(cmd *cobra.Command, args []string) {
		l := createLogger()
		cd, _ := clients.NewDocker()
		dt := clients.NewDockerTasks(cd, l)

		// find a list of resources in the current stack
		sc := config.New()
		err := sc.FromJSON(utils.StatePath())
		if err != nil {
			l.Error("No resources are running, start a stack with 'shipyard run [blueprint]'")
			return
		}

		parts := strings.Split(args[0], ".")
		if len(parts) < 2 {
			l.Error("No target specified for resource")
			return
		}

		switch parts[0] {
		case string(config.TypeContainer):
			container := parts[1]

			command := []string{"sh"}
			if len(args) > 1 {
				// TODO turn this into a function
				if args[1] != "--" {
					l.Error("No command specified, expected a seperator -- followed by a command")
					return
				}

				command = args[2:]
			}

			// find the container id
			ids, err := dt.FindContainerIDs(container, config.TypeContainer)
			if err != nil {
				l.Error("Unable to find container", "container", container)
				return
			}

			in, stdout, _ := term.StdStreams()
			err = dt.CreateShell(ids[0], command, in, stdout, stdout)
			if err != nil {
				l.Error("Could not execute command", "container", ids[0], "error", err)
				return
			}
		case string(config.TypeK8sCluster):
			// shipyard exec k8s_cluster.k3s <pod> -- <command>
			clusterName := parts[1]

			// check if the given cluster exists
			cluster, err := sc.FindResource(fmt.Sprintf("%s.%s", config.TypeK8sCluster, clusterName))
			if err != nil {
				l.Error("Unable to find cluster", "cluster", clusterName, "error", err)
				return
			}

			// get the pod to execute the command in
			if len(args) < 2 {
				l.Error("No target specified", "cluster", clusterName)
			}
			pod := args[1]

			if len(args) < 3 {
				l.Error("No command specified, expected a seperator -- followed by a command")
				return
			}

			// kubectl exec -ti pod <container>
			command := append([]string{"kubectl", "exec", "-ti", pod}, args[2:]...)

			// start a tools container
			i := config.Image{Name: "shipyardrun/tools:latest"}
			err = dt.PullImage(i, false)
			if err != nil {
				l.Error("Could pull tools image", "error", err)
				return
			}

			c := config.NewContainer(fmt.Sprintf("exec-%d", time.Now().Nanosecond()))
			sc.AddResource(c)
			c.Image = i
			c.Command = []string{"tail", "-f", "/dev/null"}

			c.Networks = cluster.(*config.K8sCluster).Networks

			wd, err := os.Getwd()
			if err != nil {
				l.Error("Could not get working directory", "error", err)
				return
			}

			c.Volumes = []config.Volume{
				config.Volume{
					Source:      wd,
					Destination: "/files",
				},
				config.Volume{
					Source:      utils.ShipyardHome(),
					Destination: "/root/.shipyard",
				},
			}

			c.Environment = []config.KV{
				config.KV{
					Key:   "KUBECONFIG",
					Value: fmt.Sprintf("/root/.shipyard/config/%s/kubeconfig-docker.yaml", clusterName),
				},
			}

			tools, err := dt.CreateContainer(c)
			if err != nil {
				l.Error("Could not create tools container", "error", err)
				return
			}
			defer dt.RemoveContainer(tools)

			in, stdout, _ := term.StdStreams()
			err = dt.CreateShell(tools, command, in, stdout, stdout)
			if err != nil {
				l.Error("Could not execute command", "cluster", clusterName, "error", err)
				return
			}
		case string(config.TypeNomadCluster):
		default:
			l.Error("Unknown resource type")
			os.Exit(1)
		}
	},
}

func init() {
	execCmd.PersistentFlags().StringVarP(&cluster, "cluster", "c", "default", "the cluster to attach to")
}
