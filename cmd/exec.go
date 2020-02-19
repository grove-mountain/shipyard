package cmd

import (
	"fmt"

	"github.com/c-bata/go-prompt"
	"github.com/docker/docker/pkg/term"
	"github.com/shipyard-run/shipyard/pkg/clients"
	"github.com/shipyard-run/shipyard/pkg/config"
	"github.com/spf13/cobra"
)

var cluster string

func promptContainer(cfg *config.Config) func(t prompt.Document) []prompt.Suggest {
	containers, _ := cfg.FindResourceByType(config.TypeContainer)

	return func(t prompt.Document) []prompt.Suggest {
		s := []prompt.Suggest{}
		for _, container := range containers {
			s = append(s, prompt.Suggest{
				Text: container.Info().Name,
			})
		}
		return s
	}
}

var execCmd = &cobra.Command{
	Use:   "exec",
	Short: "Execute a command in a Resource",
	Long:  `Execute a command in a Resource or start a Tools resource and execute`,
	Args:  cobra.NoArgs,
	Run: func(cmd *cobra.Command, args []string) {
		l := createLogger()
		cd, _ := clients.NewDocker()
		dt := clients.NewDockerTasks(cd, l)

		// find a list of resources in the current stack
		// sc := config.New()
		// err := sc.FromJSON(utils.StatePath())
		// if err != nil {
		// 	fmt.Println("No resources are running, start a stack with 'shipyard run [blueprint]'")
		// 	return
		// }

		// get the continers and clsuters
		// k8s, _ := sc.FindResourceByType(config.TypeK8sCluster)
		// nomad, _ := sc.FindResourceByType(config.TypeNomadCluster)
		// containers, _ := sc.FindResourceByType(config.TypeContainer)

		// fmt.Println("Which resource type would you like to use?")
		// fmt.Println("")

		// if len(k8s) > 0 {
		// 	fmt.Println("[1] Kubernetes Clusters")
		// }
		// if len(nomad) > 0 {
		// 	fmt.Println("[2] Nomad Clusters")
		// }
		// if len(containers) > 0 {
		// 	fmt.Println("[3] Containers")
		// }

		// // get input
		// reader := bufio.NewReader(os.Stdin)
		// char, _, _ := reader.ReadRune()

		// switch char {
		// case '1':
		// 	fmt.Println("Which pod would you like to use?")
		// case '2':
		// 	fmt.Println("Which job would you like to use?")
		// case '3':
		// 	fmt.Println("")
		// 	fmt.Println("Which container would you like to use?")
		// 	fmt.Println("")
		// 	for i, c := range containers {
		// 		fmt.Printf("[%d] %s\n", i+1, c.Info().Name)
		// 	}

		// 	fmt.Println("")
		// 	reader := bufio.NewReader(os.Stdin)
		// 	char, _, _ := reader.ReadRune()
		// 	i, _ := strconv.Atoi(string(char))
		// 	i = i - 1

		// 	// execute the command
		// 	fmt.Println("")
		// 	fmt.Printf("Starting interactive shell in container %s\n", containers[i].Info().Name)

		// find the container id
		ids, err := dt.FindContainerIDs("consul", config.TypeContainer)
		if err != nil {
			fmt.Println("Unable to find Docker container for ", ids[0])
		}

		in, stdout, _ := term.StdStreams()
		err = dt.CreateShell(ids[0], []string{"sh"}, in, stdout, stdout)
		if err != nil {
			fmt.Println("Error", err)
		}
		// default:
		// 	fmt.Println("Invalid option")
		// 	os.Exit(1)
		// }

	},
}

func init() {
	execCmd.PersistentFlags().StringVarP(&cluster, "cluster", "c", "default", "the cluster to attach to")
}
