package helpers

import (
	"fmt"
	"os"
	"os/user"
	"strconv"
	"strings"
	"time"

	"github.com/rancher/shepherd/clients/rancher"
	v1 "github.com/rancher/shepherd/clients/rancher/v1"
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
	Provider          = os.Getenv("PROVIDER")
	testuser, _       = user.Current()
	clusterCleanup, _ = strconv.ParseBool(os.Getenv("DOWNSTREAM_CLUSTER_CLEANUP"))
	ClusterNamePrefix = func() string {
		if clusterCleanup {
			return fmt.Sprintf("%s-hp-ci", Provider)
		} else {
			return fmt.Sprintf("%s-%s-hp-ci", Provider, testuser.Username)
		}
	}()
	RancherVersion        = os.Getenv("RANCHER_VERSION")
	RancherUpgradeVersion = os.Getenv("RANCHER_UPGRADE_VERSION")
	Kubeconfig            = os.Getenv("KUBECONFIG")
	DownstreamKubeconfig  = func(clusterName string) string {
		return os.Getenv(fmt.Sprintf("%s_KUBECONFIG", clusterName))
	}
	K8sUpgradedMinorVersion   = os.Getenv("K8S_UPGRADE_MINOR_VERSION")
	DownstreamK8sMinorVersion = os.Getenv("DOWNSTREAM_K8S_MINOR_VERSION")
	IsImport                  = func() bool {
		if strings.Contains(os.Getenv("CATTLE_TEST_CONFIG"), "import") {
			return true
		}
		return false
	}()
)

type HelmChart struct {
	Name           string `json:"name"`
	Chart          string `json:"chart"`
	AppVersion     string `json:"app_version"`
	DerivedVersion string `json:"version"`
}

type Context struct {
	CloudCred          *v1.SteveAPIObject
	RancherAdminClient *rancher.Client
	StdUserClient      *rancher.Client
	Session            *session.Session
	ClusterCleanup     bool
	CloudCredID        string
}

type RancherVersionInfo struct {
	Version      string
	GitCommit    string
	RancherPrime string
	Devel        bool
}
