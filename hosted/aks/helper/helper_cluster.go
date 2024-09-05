package helper

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"

	"github.com/rancher/shepherd/extensions/clusters/aks"

	"github.com/rancher/hosted-providers-e2e/hosted/helpers"

	"github.com/Masterminds/semver/v3"
	"github.com/rancher/shepherd/clients/rancher"
	management "github.com/rancher/shepherd/clients/rancher/generated/management/v3"
	"github.com/rancher/shepherd/extensions/clusters/kubernetesversions"
	"github.com/rancher/shepherd/pkg/config"
	namegen "github.com/rancher/shepherd/pkg/namegenerator"
	"k8s.io/utils/pointer"

	"github.com/epinio/epinio/acceptance/helpers/proc"
	"github.com/pkg/errors"
)

var (
	subscriptionID = os.Getenv("AKS_SUBSCRIPTION_ID")
)

func GetTags() map[string]string {
	aksConfig := new(management.AKSClusterConfigSpec)
	config.LoadConfig(aks.AKSClusterConfigConfigurationFileKey, aksConfig)
	tags := helpers.GetCommonMetadataLabels()
	for key, value := range aksConfig.Tags {
		tags[key] = value
	}
	return tags
}

// UpgradeClusterKubernetesVersion upgrades the k8s version to the value defined by upgradeToVersion.
func UpgradeClusterKubernetesVersion(cluster *management.Cluster, upgradeToVersion *string, client *rancher.Client) (*management.Cluster, error) {
	upgradedCluster := new(management.Cluster)
	upgradedCluster.Name = cluster.Name
	upgradedCluster.AKSConfig = cluster.AKSConfig
	upgradedCluster.AKSConfig.KubernetesVersion = upgradeToVersion

	cluster, err := client.Management.Cluster.Update(cluster, &upgradedCluster)
	if err != nil {
		return nil, err
	}
	return cluster, nil
}

// UpgradeNodeKubernetesVersion upgrades the k8s version of nodepool to the value defined by upgradeToVersion.
func UpgradeNodeKubernetesVersion(cluster *management.Cluster, upgradeToVersion *string, client *rancher.Client) (*management.Cluster, error) {
	upgradedCluster := new(management.Cluster)
	upgradedCluster.Name = cluster.Name
	upgradedCluster.AKSConfig = cluster.AKSConfig
	for i := range upgradedCluster.AKSConfig.NodePools {
		upgradedCluster.AKSConfig.NodePools[i].OrchestratorVersion = upgradeToVersion
	}

	cluster, err := client.Management.Cluster.Update(cluster, &upgradedCluster)
	if err != nil {
		return nil, err
	}
	return cluster, nil
}

// DeleteAKSHostCluster deletes the AKS cluster
func DeleteAKSHostCluster(cluster *management.Cluster, client *rancher.Client) error {
	return client.Management.Cluster.Delete(cluster)
}

// ListSingleVariantAKSAvailableVersions returns a list of single variants of minor versions
// For e.g 1.27.5, 1.26.6, 1.25.8
func ListSingleVariantAKSAvailableVersions(client *rancher.Client, cloudCredentialID, region string) (availableVersions []string, err error) {
	availableVersions, err = kubernetesversions.ListAKSAllVersions(client, cloudCredentialID, region)
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

// GetK8sVersionVariantAKS returns a variant of a given minor K8s version
func GetK8sVersionVariantAKS(minorVersion string, client *rancher.Client, cloudCredentialID, region string) (version string, err error) {
	versions, err := ListSingleVariantAKSAvailableVersions(client, cloudCredentialID, region)
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

// AddNodePool adds a nodepool to the list
func AddNodePool(cluster *management.Cluster, increaseBy int, client *rancher.Client) (*management.Cluster, error) {
	upgradedCluster := new(management.Cluster)
	upgradedCluster.Name = cluster.Name
	upgradedCluster.AKSConfig = cluster.AKSConfig
	nodeConfig := AksHostNodeConfig()

	for i := 1; i <= increaseBy; i++ {
		for _, np := range nodeConfig {
			newNodepool := management.AKSNodePool{
				Count:             pointer.Int64(1),
				VMSize:            np.VMSize,
				Mode:              np.Mode,
				EnableAutoScaling: np.EnableAutoScaling,
				Name:              pointer.String(namegen.RandStringLower(5)),
			}
			upgradedCluster.AKSConfig.NodePools = append(upgradedCluster.AKSConfig.NodePools, newNodepool)
		}
	}
	cluster, err := client.Management.Cluster.Update(cluster, &upgradedCluster)
	if err != nil {
		return nil, err
	}
	return cluster, nil
}

// DeleteNodePool deletes a nodepool from the list
// TODO: Modify this method to delete a custom qty of DeleteNodePool, perhaps by adding an `decreaseBy int` arg
func DeleteNodePool(cluster *management.Cluster, client *rancher.Client) (*management.Cluster, error) {
	upgradedCluster := new(management.Cluster)
	upgradedCluster.Name = cluster.Name
	upgradedCluster.AKSConfig = cluster.AKSConfig
	upgradedCluster.AKSConfig.NodePools = cluster.AKSConfig.NodePools[:1]

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
	upgradedCluster.AKSConfig = cluster.AKSConfig
	for i := range upgradedCluster.AKSConfig.NodePools {
		upgradedCluster.AKSConfig.NodePools[i].Count = pointer.Int64(nodeCount)
	}

	cluster, err := client.Management.Cluster.Update(cluster, &upgradedCluster)
	if err != nil {
		return nil, err
	}
	return cluster, nil
}

// ListAKSAvailableVersions is a function to list and return only available AKS versions for a specific cluster.
func ListAKSAvailableVersions(client *rancher.Client, clusterID string) (availableVersions []string, err error) {
	// kubernetesversions.ListAKSAvailableVersions expects cluster.Version.GitVersion to be available, which it is not sometimes, so we fetch the cluster again to ensure it has all the available data
	cluster, err := client.Management.Cluster.ByID(clusterID)
	if err != nil {
		return nil, err
	}
	return kubernetesversions.ListAKSAvailableVersions(client, cluster)
}

// Create Azure AKS cluster using AZ CLI
func CreateAKSClusterOnAzure(location string, clusterName string, k8sVersion string, nodes string) error {
	tags := GetTags()
	formattedTags := convertMapToAKSString(tags)
	fmt.Println("Creating AKS resource group ...")
	rgargs := []string{"group", "create", "--location", location, "--resource-group", clusterName, "--subscription", subscriptionID}
	fmt.Printf("Running command: az %v\n", rgargs)

	out, err := proc.RunW("az", rgargs...)
	if err != nil {
		return errors.Wrap(err, "Failed to create cluster: "+out)
	}

	fmt.Println("Creating AKS cluster ...")
	args := []string{"aks", "create", "--resource-group", clusterName, "--generate-ssh-keys", "--kubernetes-version", k8sVersion, "--enable-managed-identity", "--name", clusterName, "--subscription", subscriptionID, "--node-count", nodes, "--tags", formattedTags}
	fmt.Printf("Running command: az %v\n", args)
	out, err = proc.RunW("az", args...)
	if err != nil {
		return errors.Wrap(err, "Failed to create cluster: "+out)
	}

	fmt.Println("Created AKS cluster: ", clusterName)

	return nil
}

// convertMapToAKSString converts the map of labels to a string format acceptable by azure CLI
// acceptable format: `--tags "owner=hostedproviders" "testname=sometest"`
func convertMapToAKSString(tags map[string]string) string {
	var convertedString string
	for key, value := range tags {
		convertedString += fmt.Sprintf("\"%s=%s\" ", key, value)
	}
	return convertedString
}

// Complete cleanup steps for Azure AKS
func DeleteAKSClusteronAzure(clusterName string) error {

	fmt.Println("Deleting AKS resource group which will delete cluster too ...")
	args := []string{"group", "delete", "--name", clusterName, "--yes", "--subscription", subscriptionID}
	fmt.Printf("Running command: az %v\n", args)

	out, err := proc.RunW("az", args...)
	if err != nil {
		return errors.Wrap(err, "Failed to delete resource group: "+out)
	}

	fmt.Println("Deleted AKS resource group: ", clusterName)

	return nil
}

func ImportAKSHostedCluster(client *rancher.Client, displayName, cloudCredentialID string, enableClusterAlerting, enableClusterMonitoring, enableNetworkPolicy, windowsPreferedCluster bool, labels map[string]string) (*management.Cluster, error) {
	aksHostCluster := AksHostClusterConfig(displayName, cloudCredentialID)
	cluster := &management.Cluster{
		DockerRootDir:           "/var/lib/docker",
		AKSConfig:               aksHostCluster,
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

func AksHostClusterConfig(displayName, cloudCredentialID string) *management.AKSClusterConfigSpec {
	var aksClusterConfig ImportClusterConfig
	config.LoadConfig("aksClusterConfig", &aksClusterConfig)

	return &management.AKSClusterConfigSpec{
		AzureCredentialSecret: cloudCredentialID,
		ClusterName:           displayName,
		Imported:              aksClusterConfig.Imported,
		ResourceLocation:      aksClusterConfig.ResourceLocation,
		ResourceGroup:         aksClusterConfig.ResourceGroup,
	}
}

func AksHostNodeConfig() []management.AKSNodePool {
	var nodeConfig management.AKSClusterConfigSpec
	config.LoadConfig("aksClusterConfig", &nodeConfig)

	return nodeConfig.NodePools
}

type ImportClusterConfig struct {
	ResourceGroup    string                    `json:"resourceGroup" yaml:"resourceGroup"`
	ResourceLocation string                    `json:"resourceLocation" yaml:"resourceLocation"`
	Tags             map[string]string         `json:"tags,omitempty" yaml:"tags,omitempty"`
	Imported         bool                      `json:"imported" yaml:"imported"`
	NodePools        []*management.AKSNodePool `json:"nodePools" yaml:"nodePools"`
}

// DefaultAKS returns the default AKS version used by Rancher
func DefaultAKS(client *rancher.Client, cloudCredentialID, region string) (defaultEKS string, err error) {
	url := fmt.Sprintf("%s://%s/meta/aksVersions", "https", client.RancherConfig.Host)
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return "", err
	}
	req.Header.Add("Authorization", "Bearer "+client.RancherConfig.AdminToken)

	q := req.URL.Query()
	q.Add("cloudCredentialId", cloudCredentialID)
	q.Add("region", region)
	req.URL.RawQuery = q.Encode()

	resp, err := client.Management.APIBaseClient.Ops.Client.Do(req)
	if err != nil {
		return
	}
	bodyBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return
	}

	var versions []string
	if err = json.Unmarshal(bodyBytes, &versions); err != nil {
		return
	}

	var maxRange string
	maxRange, err = getUIK8sDefaultVersionRange(client)
	if err != nil {
		return
	}

	// Iterate in the reverse order to get the highest version
	// We obtain the value similar to UI; ref: https://github.com/rancher/ui/blob/master/lib/shared/addon/components/cluster-driver/driver-azureaks/component.js#L140
	for i := len(versions) - 1; i >= 0; i-- {
		if strings.Contains(versions[i], maxRange) {
			defaultEKS = versions[i]
			return
		}
	}

	return
}

// getUIK8sDefaultVersionRange returns the default Maj.Min version supported by the UI
func getUIK8sDefaultVersionRange(client *rancher.Client) (value string, err error) {
	url := fmt.Sprintf("%s://%s/v3/settings/ui-k8s-default-version-range", "https", client.RancherConfig.Host)
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return
	}
	req.Header.Add("Authorization", "Bearer "+client.RancherConfig.AdminToken)

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

	value = mapResponse["value"].(string)
	value = strings.TrimPrefix(value, "<=v")
	value = strings.TrimSuffix(value, ".x")
	return

}

// GetK8sVersion returns the k8s version to be used by the test;
// this value can either be a version derived from envvar DOWNSTREAM_K8S_MINOR_VERSION or the default UI value returned by DefaultAKS.
func GetK8sVersion(client *rancher.Client, cloudCredentialID, region string) (string, error) {
	if k8sMinorVersion := helpers.DownstreamK8sMinorVersion; k8sMinorVersion != "" {
		return GetK8sVersionVariantAKS(k8sMinorVersion, client, cloudCredentialID, region)
	}
	return DefaultAKS(client, cloudCredentialID, region)
}
