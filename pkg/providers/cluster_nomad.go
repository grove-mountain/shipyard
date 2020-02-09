package providers

import (
	"fmt"
	"math/rand"
	"time"

	"github.com/shipyard-run/shipyard/pkg/config"
)

const nomadBaseImage = "shipyardrun/nomad"

// TODO tidy code, add tests like k3s cluster
func (c *Cluster) createNomad() error {
	c.log.Info("Creating Cluster", "ref", c.config.Name)

	// check the cluster does not already exist
	ids, _ := c.client.FindContainerIDs(c.config.Name, c.config.NetworkRef.Name)
	if len(ids) > 0 {
		return ErrorClusterExists
	}

	// set the image
	image := fmt.Sprintf("%s:%s", nomadBaseImage, c.config.Version)

	// create the volume for the cluster
	volID, err := c.client.CreateVolume(c.config.Name)
	if err != nil {
		return err
	}

	// create the server
	// since the server is just a container create the container config and provider
	cc := config.Container{}
	cc.Name = fmt.Sprintf("server.%s", c.config.Name)
	cc.Image = config.Image{Name: image}
	cc.NetworkRef = c.config.NetworkRef
	cc.Privileged = true // nomad must run Privileged as Docker needs to manipulate ip tables and stuff

	// set the volume mount for the images
	cc.Volumes = []config.Volume{
		config.Volume{
			Source:      volID,
			Destination: "/images",
			Type:        "volume",
		},
	}

	// set the environment variables for the K3S_KUBECONFIG_OUTPUT and K3S_CLUSTER_SECRET
	cc.Environment = c.config.Environment

	// set the API server port to a random number 64000 - 65000
	apiPort := rand.Intn(1000) + 64000

	// expose the API server port
	cc.Ports = []config.Port{
		config.Port{
			Local:    4646,
			Host:     apiPort,
			Protocol: "tcp",
		},
	}

	_, err = c.client.CreateContainer(cc)
	if err != nil {
		return err
	}

	// import the images to the servers docker instance
	// importing images means that Nomad does not need to pull from a remote docker hub
	if c.config.Images != nil && len(c.config.Images) > 0 {
		//return c.ImportLocalDockerImages(c.config.Images)
	}

	// wait for nomad to start
	err = c.httpClient.HealthCheckHTTP(fmt.Sprintf("http://localhost:%d/v1/status/leader", apiPort), 60*time.Second)
	if err != nil {
		return err
	}

	// ensure all client nodes are up
	err = c.httpClient.HealthCheckNomad(fmt.Sprintf("http://localhost:%d", apiPort), c.config.Nodes, 60*time.Second)
	if err != nil {
		return err
	}

	return nil
}

func (c *Cluster) destroyNomad() error {
	return c.destroyK3s()
}
