package helpers

import (
	"fmt"
	"os"
	"time"

	"github.com/rancher/shepherd/clients/rancher"
	"github.com/rancher/shepherd/extensions/cloudcredentials"
	"github.com/rancher/shepherd/pkg/session"
)

const (
	Timeout        = 30 * time.Minute
	CattleSystemNS = "cattle-system"
)

var (
	RancherPassword = os.Getenv("RANCHER_PASSWORD")
	RancherHostname = os.Getenv("RANCHER_HOSTNAME")
	RancherChannel  = func() string {
		if channel := os.Getenv("RANCHER_CHANNEL"); channel != "" {
			return channel
		} else {
			return "latest"
		}
	}()
	Provider                = os.Getenv("PROVIDER")
	ClusterNamePrefix       = fmt.Sprintf("%shostcluster-hp", Provider)
	RancherVersion          = os.Getenv("RANCHER_VERSION")
	RancherUpgradeVersion   = os.Getenv("RANCHER_UPGRADE_VERSION")
	Kubeconfig              = os.Getenv("KUBECONFIG")
	K8sUpgradedMinorVersion = os.Getenv("K8S_UPGRADE_MINOR_VERSION")
)

type HelmChart struct {
	Name           string `json:"name"`
	Chart          string `json:"chart"`
	AppVersion     string `json:"app_version"`
	DerivedVersion string `json:"version"`
}

type Context struct {
	CloudCred      *cloudcredentials.CloudCredential
	RancherClient  *rancher.Client
	Session        *session.Session
	ClusterCleanup bool
}

type RancherVersionInfo struct {
	Version      string
	GitCommit    string
	RancherPrime string
	Devel        bool
}
