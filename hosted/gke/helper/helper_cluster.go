package helper

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"

	"github.com/rancher/shepherd/extensions/clusters/gke"
	k8slabels "k8s.io/apimachinery/pkg/labels"

	"github.com/rancher/hosted-providers-e2e/hosted/helpers"

	"github.com/Masterminds/semver/v3"
	"github.com/epinio/epinio/acceptance/helpers/proc"
	"github.com/pkg/errors"
	"github.com/rancher/shepherd/clients/rancher"
	management "github.com/rancher/shepherd/clients/rancher/generated/management/v3"
	"github.com/rancher/shepherd/extensions/clusters/kubernetesversions"
	"github.com/rancher/shepherd/pkg/config"
	namegen "github.com/rancher/shepherd/pkg/namegenerator"
	"k8s.io/utils/pointer"
)

// GetLabels fetches labels from config file, appends the common labels/tags to it and returns the map
func GetLabels() map[string]string {
	gkeConfig := new(management.GKEClusterConfigSpec)
	config.LoadConfig(gke.GKEClusterConfigConfigurationFileKey, gkeConfig)
	labels := helpers.GetCommonMetadataLabels()
	if gkeConfig.Labels != nil {
		for key, value := range *gkeConfig.Labels {
			labels[key] = value
		}
	}
	return labels
}

// UpgradeKubernetesVersion upgrades the k8s version to the value defined by upgradeToVersion; if upgradeNodePool is true, it also upgrades nodepools' k8s version
func UpgradeKubernetesVersion(cluster *management.Cluster, upgradeToVersion *string, client *rancher.Client, upgradeNodePool bool) (*management.Cluster, error) {
	upgradedCluster := new(management.Cluster)
	upgradedCluster.Name = cluster.Name
	upgradedCluster.GKEConfig = cluster.GKEConfig
	upgradedCluster.GKEConfig.KubernetesVersion = upgradeToVersion
	if upgradeNodePool {
		for i := range upgradedCluster.GKEConfig.NodePools {
			upgradedCluster.GKEConfig.NodePools[i].Version = upgradeToVersion
		}
	}

	cluster, err := client.Management.Cluster.Update(cluster, &upgradedCluster)
	if err != nil {
		return nil, err
	}
	return cluster, nil
}

// DeleteGKEHostCluster deletes the GKE cluster
func DeleteGKEHostCluster(cluster *management.Cluster, client *rancher.Client) error {
	return client.Management.Cluster.Delete(cluster)
}

// AddNodePool adds a nodepool to the list
func AddNodePool(cluster *management.Cluster, increaseBy int, client *rancher.Client) (*management.Cluster, error) {
	upgradedCluster := new(management.Cluster)
	upgradedCluster.Name = cluster.Name
	upgradedCluster.GKEConfig = cluster.GKEConfig
	nodeConfig := GkeHostNodeConfig()

	for i := 1; i <= increaseBy; i++ {
		for _, np := range nodeConfig {
			newNodepool := management.GKENodePoolConfig{
				InitialNodeCount:  np.InitialNodeCount,
				Version:           cluster.GKEConfig.KubernetesVersion,
				Config:            np.Config,
				Autoscaling:       np.Autoscaling,
				Management:        np.Management,
				MaxPodsConstraint: np.MaxPodsConstraint,
				Name:              pointer.String(namegen.RandStringLower(5)),
			}
			upgradedCluster.GKEConfig.NodePools = append(upgradedCluster.GKEConfig.NodePools, newNodepool)
		}
	}
	cluster, err := client.Management.Cluster.Update(cluster, &upgradedCluster)
	if err != nil {
		return nil, err
	}
	return cluster, nil
}

// DeleteNodePool deletes a nodepool from the list
// TODO: Modify this method to delete a custom qty of nodepool, perhaps by adding an `decreaseBy int` arg
func DeleteNodePool(cluster *management.Cluster, client *rancher.Client) (*management.Cluster, error) {
	upgradedCluster := new(management.Cluster)
	upgradedCluster.Name = cluster.Name
	upgradedCluster.GKEConfig = cluster.GKEConfig

	upgradedCluster.GKEConfig.NodePools = cluster.GKEConfig.NodePools[1:]

	cluster, err := client.Management.Cluster.Update(cluster, &upgradedCluster)
	if err != nil {
		return nil, err
	}
	return cluster, nil
}

// ScaleNodePool modifies the number of initialNodeCount of all the nodepools as defined by nodeCount
func ScaleNodePool(cluster *management.Cluster, client *rancher.Client, nodeCount int64) (*management.Cluster, error) {
	upgradedCluster := new(management.Cluster)
	upgradedCluster.Name = cluster.Name
	upgradedCluster.GKEConfig = cluster.GKEConfig
	for i := range upgradedCluster.GKEConfig.NodePools {
		upgradedCluster.GKEConfig.NodePools[i].InitialNodeCount = pointer.Int64(nodeCount)
	}

	cluster, err := client.Management.Cluster.Update(cluster, &upgradedCluster)
	if err != nil {
		return nil, err
	}
	return cluster, nil
}

// ListGKEAvailableVersions is a function to list and return only available GKE versions for a specific cluster.
func ListGKEAvailableVersions(client *rancher.Client, clusterID string) (availableVersions []string, err error) {
	// kubernetesversions.ListGKEAvailableVersions expects cluster.Version.GitVersion to be available, which it is not sometimes, so we fetch the cluster again to ensure it has all the available data
	cluster, err := client.Management.Cluster.ByID(clusterID)
	if err != nil {
		return nil, err
	}
	return kubernetesversions.ListGKEAvailableVersions(client, cluster)
}

// ListSingleVariantGKEAvailableVersions returns a list of single variants of minor versions
// For e.g 1.27.5-gke.1700, 1.26.6-gke.2100, 1.25.8-gke.200
func ListSingleVariantGKEAvailableVersions(client *rancher.Client, projectID, cloudCredentialID, zone, region string) (availableVersions []string, err error) {
	availableVersions, err = kubernetesversions.ListGKEAllVersions(client, projectID, cloudCredentialID, zone, region)
	if err != nil {
		return nil, err
	}
	var singleVersionList []string
	var oldMinor uint64
	for _, version := range availableVersions {
		semVersion := semver.MustParse(version)
		if currentMinor := semVersion.Minor(); oldMinor != currentMinor {
			singleVersionList = append(singleVersionList, version)
			oldMinor = currentMinor
		}
	}
	return singleVersionList, nil
}

// GetK8sVersionVariantGKE returns a variant of a given minor K8s version
func GetK8sVersionVariantGKE(minorVersion string, client *rancher.Client, projectID, cloudCredentialID, zone, region string) (string, error) {
	versions, err := ListSingleVariantGKEAvailableVersions(client, projectID, cloudCredentialID, zone, region)
	if err != nil {
		return "", err
	}

	for _, version := range versions {
		if strings.Contains(version, minorVersion) {
			return version, nil
		}
	}
	return "", fmt.Errorf("version %s not found", minorVersion)
}

// Create Google GKE cluster using gcloud CLI
func CreateGKEClusterOnGCloud(zone string, clusterName string, project string, k8sVersion string) error {

	labels := GetLabels()
	labelsAsString := k8slabels.SelectorFromSet(labels).String()

	// creating GKE using gcloud changes the kubeconfig to use GKE; this can be problematic for test cases that need to use local cluster;
	// this workaround helps to keep the original kubeconfig
	// TODO: move this to a common function once AKS and EKS is implemented
	currentKubeconfig := os.Getenv("KUBECONFIG")
	defer os.Setenv("KUBECONFIG", currentKubeconfig)

	tmpKubeConfig, err := os.CreateTemp("", clusterName)
	if err != nil {
		return err
	}
	defer os.Remove(tmpKubeConfig.Name()) // clean up
	os.Setenv("KUBECONFIG", tmpKubeConfig.Name())

	fmt.Println("Creating GKE cluster ...")
	args := []string{"container", "clusters", "create", clusterName, "--project", project, "--zone", zone, "--cluster-version", k8sVersion, "--labels", labelsAsString, "--network", "default", "--release-channel", "None", "--machine-type", "n2-standard-2", "--disk-size", "100", "--num-nodes", "1", "--no-enable-cloud-logging", "--no-enable-cloud-monitoring", "--no-enable-master-authorized-networks"}
	fmt.Printf("Running command: gcloud %v\n", args)
	out, err := proc.RunW("gcloud", args...)
	if err != nil {
		return errors.Wrap(err, "Failed to create cluster: "+out)
	}

	fmt.Println("Created GKE cluster: ", clusterName)

	return nil
}

// Complete cleanup steps for Google GKE
func DeleteGKEClusterOnGCloud(zone, project, clusterName string) error {

	fmt.Println("Deleting GKE cluster ...")
	args := []string{"container", "clusters", "delete", clusterName, "--zone", zone, "--quiet", "--project", project}
	fmt.Printf("Running command: gcloud %v\n", args)
	out, err := proc.RunW("gcloud", args...)
	if err != nil {
		return errors.Wrap(err, "Failed to delete cluster: "+out)
	}

	fmt.Println("Deleted GKE cluster: ", clusterName)

	return nil
}

func ImportGKEHostedCluster(client *rancher.Client, displayName, cloudCredentialID string, enableClusterAlerting, enableClusterMonitoring, enableNetworkPolicy, windowsPreferedCluster bool, labels map[string]string) (*management.Cluster, error) {
	gkeHostCluster := GkeHostClusterConfig(displayName, cloudCredentialID)
	cluster := &management.Cluster{
		DockerRootDir:           "/var/lib/docker",
		GKEConfig:               gkeHostCluster,
		Name:                    displayName,
		EnableClusterAlerting:   enableClusterAlerting,
		EnableClusterMonitoring: enableClusterMonitoring,
		EnableNetworkPolicy:     &enableNetworkPolicy,
		Labels:                  labels,
		WindowsPreferedCluster:  windowsPreferedCluster,
	}

	clusterResp, err := client.Management.Cluster.Create(cluster)
	if err != nil {
		return nil, err
	}
	return clusterResp, err
}

func GkeHostClusterConfig(displayName, cloudCredentialID string) *management.GKEClusterConfigSpec {
	var gkeClusterConfig ImportClusterConfig
	config.LoadConfig("gkeClusterConfig", &gkeClusterConfig)

	return &management.GKEClusterConfigSpec{
		GoogleCredentialSecret: cloudCredentialID,
		ClusterName:            displayName,
		Imported:               gkeClusterConfig.Imported,
		Zone:                   gkeClusterConfig.Zone,
		ProjectID:              gkeClusterConfig.ProjectID,
	}
}

func GkeHostNodeConfig() []management.GKENodePoolConfig {
	var nodeConfig management.GKEClusterConfigSpec
	config.LoadConfig("gkeClusterConfig", &nodeConfig)

	return nodeConfig.NodePools
}

type ImportClusterConfig struct {
	ProjectID string                          `json:"projectID" yaml:"projectID"`
	Zone      string                          `json:"zone" yaml:"zone"`
	Imported  bool                            `json:"imported" yaml:"imported"`
	NodePools []*management.GKENodePoolConfig `json:"nodePools" yaml:"nodePools"`
	Labels    *map[string]string              `json:"labels,omitempty" yaml:"labels,omitempty"`
}

// DefaultGKE returns the default GKE version used by Rancher
func DefaultGKE(client *rancher.Client, projectID, cloudCredentialID, zone, region string) (defaultGKE string, err error) {
	url := fmt.Sprintf("%s://%s/meta/gkeVersions", "https", client.RancherConfig.Host)
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return "", err
	}
	req.Header.Add("Authorization", "Bearer "+client.RancherConfig.AdminToken)

	q := req.URL.Query()
	q.Add("cloudCredentialId", cloudCredentialID)

	if zone != "" {
		q.Add("zone", zone)
	} else if region != "" {
		q.Add("region", region)
	}

	q.Add("projectId", projectID)
	req.URL.RawQuery = q.Encode()

	resp, err := client.Management.APIBaseClient.Ops.Client.Do(req)
	if err != nil {
		return
	}
	bodyBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return
	}

	var mapResponse map[string]interface{}
	if err = json.Unmarshal(bodyBytes, &mapResponse); err != nil {
		return
	}

	defaultGKE = mapResponse["defaultClusterVersion"].(string)

	return
}

// GetK8sVersion returns the k8s version to be used by the test;
// this value can either be a version derived from envvar DOWNSTREAM_K8S_MINOR_VERSION or the default UI value returned by DefaultGKE.
func GetK8sVersion(client *rancher.Client, projectID, cloudCredentialID, zone, region string) (string, error) {
	if k8sMinorVersion := helpers.DownstreamK8sMinorVersion; k8sMinorVersion != "" {
		return GetK8sVersionVariantGKE(k8sMinorVersion, client, projectID, cloudCredentialID, zone, region)
	}
	return DefaultGKE(client, projectID, cloudCredentialID, zone, region)
}
