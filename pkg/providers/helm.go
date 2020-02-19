package providers

import (
	"sync"
	"time"

	hclog "github.com/hashicorp/go-hclog"
	"github.com/shipyard-run/shipyard/pkg/clients"
	"github.com/shipyard-run/shipyard/pkg/config"
	"github.com/shipyard-run/shipyard/pkg/utils"
	"golang.org/x/xerrors"
)

var helmLock sync.Mutex

func init() {
	helmLock = sync.Mutex{}
}

type Helm struct {
	config     *config.Helm
	kubeClient clients.Kubernetes
	helmClient clients.Helm
	log        hclog.Logger
}

func NewHelm(c *config.Helm, kc clients.Kubernetes, hc clients.Helm, l hclog.Logger) *Helm {
	return &Helm{c, kc, hc, l}
}

func (h *Helm) Create() error {
	h.log.Info("Creating Helm chart", "ref", h.config.Name)

	// get the target cluster
	kcPath, err := h.getKubeConfigPath()
	if err != nil {
		return err
	}

	// obtain a lock so that other runs can not execute helm at the same time
	helmLock.Lock()

	// set the KubeConfig for the kubernetes client
	// this is used by the healthchecks
	err = h.kubeClient.SetConfig(kcPath)
	if err != nil {
		helmLock.Unlock()
		return xerrors.Errorf("unable to create Kubernetes client: %w", err)
	}

	err = h.helmClient.Create(kcPath, h.config.Name, h.config.Chart, h.config.Values)
	if err != nil {
		helmLock.Unlock()
		return err
	}

	// release the lock
	helmLock.Unlock()

	// we can now health check the install
	if h.config.HealthCheck != nil && len(h.config.HealthCheck.Pods) > 0 {
		to, err := time.ParseDuration(h.config.HealthCheck.Timeout)
		if err != nil {
			return xerrors.Errorf("unable to parse healthcheck duration: %w", err)
		}

		err = h.kubeClient.HealthCheckPods(h.config.HealthCheck.Pods, to)
		if err != nil {
			return xerrors.Errorf("healthcheck failed after helm chart setup: %w", err)
		}
	}

	return nil
}

func (h *Helm) Destroy() error {
	h.log.Info("Destroy Helm chart", "ref", h.config.Name)
	kcPath, err := h.getKubeConfigPath()
	if err != nil {
		return err
	}

	// get the target cluster
	h.helmClient.Destroy(kcPath, h.config.Name)
	return nil
}

func (h *Helm) getKubeConfigPath() (string, error) {
	target, err := h.config.FindDependentResource(h.config.Cluster)
	if err != nil {
		return "", xerrors.Errorf("Unable to find cluster: %w", err)
	}

	_, destPath, _ := utils.CreateKubeConfigPath(target.Info().Name)
	return destPath, nil
}

func (h *Helm) Lookup() ([]string, error) {
	return []string{}, nil
}
