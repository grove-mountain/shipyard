package cmd

import (
	"fmt"
	"os"

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
	Args:               cobra.MinimumNArgs(3),
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

		switch args[0] {
		case string(config.TypeContainer):
			container := args[1]

			if args[2] != "--" {
				l.Error("No command specified, expected a seperator -- followed by a command")
				return
			}

			// find the container id
			ids, err := dt.FindContainerIDs(container, config.TypeContainer)
			if err != nil {
				l.Error("Unable to find container for ", ids[0])
			}

			in, stdout, _ := term.StdStreams()
			err = dt.CreateShell(ids[0], args[3:], in, stdout, stdout)
			if err != nil {
				fmt.Println("Error", err)
			}
		case string(config.TypeK8sCluster):
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
