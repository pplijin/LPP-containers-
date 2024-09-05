package helper

import (
	"fmt"

	"github.com/Masterminds/semver/v3"
	"github.com/rancher/rancher/tests/framework/clients/rancher"
	management "github.com/rancher/rancher/tests/framework/clients/rancher/generated/management/v3"
	"github.com/rancher/rancher/tests/framework/extensions/clusters/kubernetesversions"
	"github.com/rancher/rancher/tests/framework/pkg/config"
	namegen "github.com/rancher/rancher/tests/framework/pkg/namegenerator"
	"k8s.io/utils/pointer"

	"github.com/epinio/epinio/acceptance/helpers/proc"
	"github.com/pkg/errors"
)

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

	fmt.Println("Creating AKS resource group ...")
	out, err := proc.RunW("az", "group", "create", "--location", location, "--resource-group", clusterName)
	if err != nil {
		return errors.Wrap(err, "Failed to create cluster: "+out)
	}

	fmt.Println("Creating AKS cluster ...")
	out, err = proc.RunW("az", "aks", "create", "--resource-group", clusterName, "--generate-ssh-keys", "--kubernetes-version", k8sVersion, "--enable-managed-identity", "--name", clusterName, "--node-count", nodes)
	if err != nil {
		return errors.Wrap(err, "Failed to create cluster: "+out)
	}

	fmt.Println("Created AKS cluster: ", clusterName)

	return nil
}

// Complete cleanup steps for Azure AKS
func DeleteAKSClusteronAzure(clusterName string) error {

	fmt.Println("Deleting AKS resource group which will delete cluster too ...")
	out, err := proc.RunW("az", "group", "delete", "--name", clusterName, "--yes")
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
	Imported         bool                      `json:"imported" yaml:"imported"`
	NodePools        []*management.AKSNodePool `json:"nodePools" yaml:"nodePools"`
}
