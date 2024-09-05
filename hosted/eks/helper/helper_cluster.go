package helper

import (
	"fmt"
	"os"
	"strconv"

	"github.com/rancher/shepherd/extensions/clusters/eks"

	"github.com/rancher/hosted-providers-e2e/hosted/helpers"

	"github.com/epinio/epinio/acceptance/helpers/proc"
	"github.com/pkg/errors"
	"github.com/rancher/shepherd/clients/rancher"
	management "github.com/rancher/shepherd/clients/rancher/generated/management/v3"
	"github.com/rancher/shepherd/extensions/clusters/kubernetesversions"
	"github.com/rancher/shepherd/pkg/config"
	namegen "github.com/rancher/shepherd/pkg/namegenerator"
	k8slabels "k8s.io/apimachinery/pkg/labels"
	"k8s.io/utils/pointer"
)

func GetTags() map[string]string {
	eksConfig := new(management.EKSClusterConfigSpec)
	config.LoadConfig(eks.EKSClusterConfigConfigurationFileKey, eksConfig)
	providerTags := helpers.GetCommonMetadataLabels()
	if clusterCleanup, _ := strconv.ParseBool(os.Getenv("DOWNSTREAM_CLUSTER_CLEANUP")); clusterCleanup == false {
		providerTags["janitor-ignore"] = "true"
	}

	if eksConfig.Tags != nil {
		for key, value := range *eksConfig.Tags {
			providerTags[key] = value
		}
	}
	return providerTags
}

// UpgradeClusterKubernetesVersion upgrades the k8s version to the value defined by upgradeToVersion.
func UpgradeClusterKubernetesVersion(cluster *management.Cluster, upgradeToVersion *string, client *rancher.Client) (*management.Cluster, error) {
	upgradedCluster := new(management.Cluster)
	upgradedCluster.Name = cluster.Name
	upgradedCluster.EKSConfig = cluster.EKSConfig
	upgradedCluster.EKSConfig.KubernetesVersion = upgradeToVersion

	cluster, err := client.Management.Cluster.Update(cluster, &upgradedCluster)
	if err != nil {
		return nil, err
	}
	return cluster, nil
}

// UpgradeNodeKubernetesVersion upgrades the k8s version of nodegroup to the value defined by upgradeToVersion.
func UpgradeNodeKubernetesVersion(cluster *management.Cluster, upgradeToVersion *string, client *rancher.Client) (*management.Cluster, error) {
	upgradedCluster := new(management.Cluster)
	upgradedCluster.Name = cluster.Name
	upgradedCluster.EKSConfig = cluster.EKSConfig
	for i := range upgradedCluster.EKSConfig.NodeGroups {
		upgradedCluster.EKSConfig.NodeGroups[i].Version = upgradeToVersion
	}

	cluster, err := client.Management.Cluster.Update(cluster, &upgradedCluster)
	if err != nil {
		return nil, err
	}
	return cluster, nil
}

// DeleteEKSHostCluster deletes the EKS cluster
func DeleteEKSHostCluster(cluster *management.Cluster, client *rancher.Client) error {
	return client.Management.Cluster.Delete(cluster)
}

// AddNodeGroup adds a nodegroup to the list
func AddNodeGroup(cluster *management.Cluster, increaseBy int, client *rancher.Client) (*management.Cluster, error) {
	upgradedCluster := new(management.Cluster)
	upgradedCluster.Name = cluster.Name
	upgradedCluster.EKSConfig = cluster.EKSConfig
	nodeConfig := EksHostNodeConfig()

	for i := 1; i <= increaseBy; i++ {
		for _, ng := range nodeConfig {
			newNodeGroup := management.NodeGroup{
				NodegroupName: pointer.String(namegen.AppendRandomString("nodegroup")),
				DesiredSize:   ng.DesiredSize,
				DiskSize:      ng.DiskSize,
				InstanceType:  ng.InstanceType,
				MaxSize:       ng.MaxSize,
				MinSize:       ng.MinSize,
			}
			upgradedCluster.EKSConfig.NodeGroups = append(upgradedCluster.EKSConfig.NodeGroups, newNodeGroup)
		}
	}
	cluster, err := client.Management.Cluster.Update(cluster, &upgradedCluster)
	if err != nil {
		return nil, err
	}
	return cluster, nil
}

// DeleteNodeGroup deletes a nodegroup from the list
// TODO: Modify this method to delete a custom qty of DeleteNodeGroup, perhaps by adding an `decreaseBy int` arg
func DeleteNodeGroup(cluster *management.Cluster, client *rancher.Client) (*management.Cluster, error) {
	upgradedCluster := new(management.Cluster)
	upgradedCluster.Name = cluster.Name
	upgradedCluster.EKSConfig = cluster.EKSConfig

	upgradedCluster.EKSConfig.NodeGroups = cluster.EKSConfig.NodeGroups[:1]

	cluster, err := client.Management.Cluster.Update(cluster, &upgradedCluster)
	if err != nil {
		return nil, err
	}
	return cluster, nil
}

// ScaleNodeGroup modifies the number of initialNodeCount of all the nodegroups as defined by nodeCount
func ScaleNodeGroup(cluster *management.Cluster, client *rancher.Client, nodeCount int64) (*management.Cluster, error) {
	upgradedCluster := new(management.Cluster)
	upgradedCluster.Name = cluster.Name
	upgradedCluster.EKSConfig = cluster.EKSConfig
	for i := range upgradedCluster.EKSConfig.NodeGroups {
		upgradedCluster.EKSConfig.NodeGroups[i].DesiredSize = pointer.Int64(nodeCount)
		upgradedCluster.EKSConfig.NodeGroups[i].MaxSize = pointer.Int64(nodeCount)
		upgradedCluster.EKSConfig.NodeGroups[i].MinSize = pointer.Int64(nodeCount)
	}

	cluster, err := client.Management.Cluster.Update(cluster, &upgradedCluster)
	if err != nil {
		return nil, err
	}
	return cluster, nil
}

// ListEKSAvailableVersions is a function to list and return only available EKS versions for a specific cluster.
func ListEKSAvailableVersions(client *rancher.Client, clusterID string) (availableVersions []string, err error) {
	// kubernetesversions.ListEKSAvailableVersions expects cluster.Version.GitVersion to be available, which it is not sometimes, so we fetch the cluster again to ensure it has all the available data
	cluster, err := client.Management.Cluster.ByID(clusterID)
	if err != nil {
		return nil, err
	}
	return kubernetesversions.ListEKSAvailableVersions(client, cluster)
}

// Create AWS EKS cluster using EKS CLI
func CreateEKSClusterOnAWS(eks_region string, clusterName string, k8sVersion string, nodes string) error {

	tags := GetTags()
	formattedTags := k8slabels.SelectorFromSet(tags).String()
	fmt.Println("Creating EKS cluster ...")
	args := []string{"create", "cluster", "--region=" + eks_region, "--name=" + clusterName, "--version=" + k8sVersion, "--nodegroup-name", "ranchernodes", "--nodes", nodes, "--managed", "--tags", formattedTags}
	fmt.Printf("Running command: eksctl %v\n", args)
	out, err := proc.RunW("eksctl", args...)
	if err != nil {
		return errors.Wrap(err, "Failed to create cluster: "+out)
	}
	fmt.Println("Created EKS cluster: ", clusterName)

	return nil
}

// Complete cleanup steps for Amazon EKS
func DeleteEKSClusterOnAWS(eks_region string, clusterName string) error {

	fmt.Println("Deleting EKS cluster ...")
	args := []string{"delete", "cluster", "--region=" + eks_region, "--name=" + clusterName, "--disable-nodegroup-eviction"}
	fmt.Printf("Running command: eksctl %v\n", args)
	out, err := proc.RunW("eksctl", args...)
	if err != nil {
		return errors.Wrap(err, "Failed to delete cluster: "+out)
	}

	fmt.Println("Deleted EKS cluster: ", clusterName)

	return nil
}

func ImportEKSHostedCluster(client *rancher.Client, displayName, cloudCredentialID string, enableClusterAlerting, enableClusterMonitoring, enableNetworkPolicy, windowsPreferedCluster bool, labels map[string]string) (*management.Cluster, error) {
	eksHostCluster := EksHostClusterConfig(displayName, cloudCredentialID)
	cluster := &management.Cluster{
		DockerRootDir:           "/var/lib/docker",
		EKSConfig:               eksHostCluster,
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

func EksHostClusterConfig(displayName, cloudCredentialID string) *management.EKSClusterConfigSpec {
	var eksClusterConfig ImportClusterConfig
	config.LoadConfig("eksClusterConfig", &eksClusterConfig)

	return &management.EKSClusterConfigSpec{
		AmazonCredentialSecret: cloudCredentialID,
		DisplayName:            displayName,
		Imported:               eksClusterConfig.Imported,
		Region:                 eksClusterConfig.Region,
	}
}

func EksHostNodeConfig() []management.NodeGroup {
	var nodeConfig management.EKSClusterConfigSpec
	config.LoadConfig("eksClusterConfig", &nodeConfig)

	return nodeConfig.NodeGroups
}

type ImportClusterConfig struct {
	Region     string                  `json:"region" yaml:"region"`
	Imported   bool                    `json:"imported" yaml:"imported"`
	NodeGroups []*management.NodeGroup `json:"nodeGroups" yaml:"nodeGroups"`
	Tags       *map[string]string      `json:"tags,omitempty" yaml:"tags,omitempty"`
}

// DefaultEKS returns a version less than the highest version or K8S_UPGRADE_MINOR_VERSION if it is set.
// Note: It does not return the default version used by UI which is the highest supported version; this is done to support k8s upgrade tests.
func DefaultEKS(client *rancher.Client) (defaultEKS string, err error) {
	var versions []string
	versions, err = kubernetesversions.ListEKSAllVersions(client)
	if err != nil {
		return
	}

	if upgradeVersion := helpers.K8sUpgradedMinorVersion; upgradeVersion != "" {
		for _, version := range versions {
			if helpers.VersionCompare(upgradeVersion, version) > 0 {
				return version, nil
			}
		}

	}
	return versions[1], nil
}

// GetK8sVersion returns the k8s version to be used by the test;
// this value can either be envvar DOWNSTREAM_K8S_MINOR_VERSION or the default UI value returned by DefaultEKS.
func GetK8sVersion(client *rancher.Client) (string, error) {
	if k8sVersion := helpers.DownstreamK8sMinorVersion; k8sVersion != "" {
		return k8sVersion, nil
	}
	return DefaultEKS(client)
}
