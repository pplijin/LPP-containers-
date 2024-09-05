package helper

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/rancher-sandbox/ele-testhelpers/tools"
	"github.com/rancher/shepherd/extensions/clusters"
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

// CreateAKSHostedCluster creates the AKS cluster on Rancher
func CreateAKSHostedCluster(client *rancher.Client, displayName, cloudCredentialID, k8sVersion, location string) (*management.Cluster, error) {
	var aksClusterConfig aks.ClusterConfig
	config.LoadConfig(aks.AKSClusterConfigConfigurationFileKey, &aksClusterConfig)

	aksClusterConfig.ResourceGroup = displayName
	dnsPrefix := displayName + "-dns"
	aksClusterConfig.DNSPrefix = &dnsPrefix
	aksClusterConfig.ResourceLocation = location
	aksClusterConfig.Tags = helpers.GetCommonMetadataLabels()
	aksClusterConfig.KubernetesVersion = &k8sVersion

	return aks.CreateAKSHostedCluster(client, displayName, cloudCredentialID, aksClusterConfig, false, false, false, false, nil)
}

// ImportAKSHostedCluster imports an AKS cluster to Rancher
func ImportAKSHostedCluster(client *rancher.Client, clusterName, cloudCredentialID, location string, tags map[string]string) (*management.Cluster, error) {
	cluster := &management.Cluster{
		DockerRootDir: "/var/lib/docker",
		AKSConfig: &management.AKSClusterConfigSpec{
			AzureCredentialSecret: cloudCredentialID,
			ClusterName:           clusterName,
			Imported:              true,
			ResourceLocation:      location,
			ResourceGroup:         clusterName,
			Tags:                  tags,
		},
		Name: clusterName,
	}

	clusterResp, err := client.Management.Cluster.Create(cluster)
	Expect(err).To(BeNil())

	return clusterResp, err
}

// DeleteAKSHostCluster deletes the AKS cluster
func DeleteAKSHostCluster(cluster *management.Cluster, client *rancher.Client) error {
	return client.Management.Cluster.Delete(cluster)
}

// UpgradeClusterKubernetesVersion upgrades the k8s version to the value defined by upgradeToVersion;
// if checkClusterConfig is set to true, it will validate that the cluster control plane has been upgrade successfully
func UpgradeClusterKubernetesVersion(cluster *management.Cluster, upgradeToVersion string, client *rancher.Client, checkClusterConfig bool) (*management.Cluster, error) {
	upgradedCluster := cluster
	currentVersion := *cluster.AKSConfig.KubernetesVersion
	upgradedCluster.AKSConfig.KubernetesVersion = &upgradeToVersion

	var err error
	cluster, err = client.Management.Cluster.Update(cluster, &upgradedCluster)
	Expect(err).To(BeNil())

	if checkClusterConfig {
		// Check if the desired config is set correctly
		Expect(*cluster.AKSConfig.KubernetesVersion).To(Equal(upgradeToVersion))
		// ensure nodepool version is still the same when config is applied
		for _, np := range cluster.AKSConfig.NodePools {
			Expect(*np.OrchestratorVersion).To(Equal(currentVersion))
		}

		// Check if the desired config has been applied in Rancher
		Eventually(func() string {
			ginkgo.GinkgoLogr.Info("Waiting for k8s upgrade to appear in AKSStatus.UpstreamSpec ...")
			cluster, err = client.Management.Cluster.ByID(cluster.ID)
			Expect(err).To(BeNil())
			return *cluster.AKSStatus.UpstreamSpec.KubernetesVersion
		}, tools.SetTimeout(10*time.Minute), 5*time.Second).Should(Equal(upgradeToVersion))
		// ensure nodepool version is same in Rancher
		for _, np := range cluster.AKSStatus.UpstreamSpec.NodePools {
			Expect(*np.OrchestratorVersion).To(Equal(currentVersion))
		}

	}
	return cluster, nil
}

// UpgradeNodeKubernetesVersion upgrades the k8s version of nodepool to the value defined by upgradeToVersion;
// if wait is set to true, it will wait until the cluster finishes upgrading;
// if checkClusterConfig is set to true, it will validate that nodepool has been upgraded successfully
func UpgradeNodeKubernetesVersion(cluster *management.Cluster, upgradeToVersion string, client *rancher.Client, wait, checkClusterConfig bool) (*management.Cluster, error) {
	upgradedCluster := cluster
	for i := range upgradedCluster.AKSConfig.NodePools {
		upgradedCluster.AKSConfig.NodePools[i].OrchestratorVersion = &upgradeToVersion
	}
	var err error
	cluster, err = client.Management.Cluster.Update(cluster, &upgradedCluster)
	Expect(err).To(BeNil())

	if checkClusterConfig {
		// Check if the desired config is set correctly
		for _, np := range cluster.AKSConfig.NodePools {
			Expect(*np.OrchestratorVersion).To(Equal(upgradeToVersion))
		}
	}

	if wait {
		err = clusters.WaitClusterToBeUpgraded(client, cluster.ID)
		Expect(err).To(BeNil())
	}

	if checkClusterConfig {
		// Check if the desired config has been applied in Rancher
		Eventually(func() bool {
			ginkgo.GinkgoLogr.Info("waiting for the nodepool upgrade to appear in AKSStatus.UpstreamSpec ...")
			cluster, err = client.Management.Cluster.ByID(cluster.ID)
			Expect(err).To(BeNil())
			for _, np := range cluster.AKSStatus.UpstreamSpec.NodePools {
				if *np.OrchestratorVersion != upgradeToVersion {
					return false
				}
			}
			return true
		}, tools.SetTimeout(12*time.Minute), 10*time.Second).Should(BeTrue())
	}
	return cluster, nil
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
	return helpers.FilterUIUnsupportedVersions(singleVersionList, client), nil
}

// GetK8sVersionVariantAKS returns a variant of a given minor K8s version
func GetK8sVersionVariantAKS(minorVersion string, client *rancher.Client, cloudCredentialID, region string) (string, error) {
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

// AddNodePool adds a nodepool to the list; it uses the nodepool template defined in CATTLE_TEST_CONFIG file
// if wait is set to true, it will wait until the cluster finishes upgrading;
// if checkClusterConfig is set to true, it will validate that nodepool has been added successfully
func AddNodePool(cluster *management.Cluster, increaseBy int, client *rancher.Client, wait, checkClusterConfig bool) (*management.Cluster, error) {
	upgradedCluster := cluster
	currentNodePoolNumber := len(cluster.AKSConfig.NodePools)

	// We use management.AKSClusterConfigSpec instead of the usual aks.ClusterConfig to unmarshal the data without the need of a lot of post-processing.
	var aksConfig management.AKSClusterConfigSpec
	config.LoadConfig(aks.AKSClusterConfigConfigurationFileKey, &aksConfig)
	npTemplate := aksConfig.NodePools[0]

	updateNodePoolsList := cluster.AKSConfig.NodePools

	for i := 1; i <= increaseBy; i++ {
		newNodepool := management.AKSNodePool{
			Count:             pointer.Int64(1),
			VMSize:            npTemplate.VMSize,
			Mode:              npTemplate.Mode,
			EnableAutoScaling: npTemplate.EnableAutoScaling,
			Name:              pointer.String(namegen.RandStringLower(5)),
		}
		updateNodePoolsList = append(updateNodePoolsList, newNodepool)

	}
	upgradedCluster.AKSConfig.NodePools = updateNodePoolsList

	var err error
	cluster, err = client.Management.Cluster.Update(cluster, &upgradedCluster)
	Expect(err).To(BeNil())

	if checkClusterConfig {
		// Check if the desired config is set correctly
		Expect(len(cluster.AKSConfig.NodePools)).Should(BeNumerically("==", currentNodePoolNumber+increaseBy))
		for i, np := range cluster.AKSConfig.NodePools {
			Expect(np.Name).To(Equal(updateNodePoolsList[i].Name))
		}
	}

	if wait {
		err = clusters.WaitClusterToBeUpgraded(client, cluster.ID)
		Expect(err).To(BeNil())
	}
	if checkClusterConfig {
		// Check if the desired config has been applied in Rancher
		Eventually(func() int {
			ginkgo.GinkgoLogr.Info("Waiting for the total nodepool count to increase in AKSStatus.UpstreamSpec ...")
			cluster, err = client.Management.Cluster.ByID(cluster.ID)
			Expect(err).To(BeNil())
			return len(cluster.AKSStatus.UpstreamSpec.NodePools)
		}, tools.SetTimeout(12*time.Minute), 10*time.Second).Should(BeNumerically("==", currentNodePoolNumber+increaseBy))

		for i, np := range cluster.AKSStatus.UpstreamSpec.NodePools {
			Expect(np.Name).To(Equal(updateNodePoolsList[i].Name))
		}
	}
	return cluster, nil
}

// DeleteNodePool deletes a nodepool from the list; if wait is set to true, it will wait until the cluster finishes upgrading;
// if checkClusterConfig is set to true, it will validate that nodepool has been deleted successfully
// TODO: Modify this method to delete a custom qty of DeleteNodePool, perhaps by adding an `decreaseBy int` arg
func DeleteNodePool(cluster *management.Cluster, client *rancher.Client, wait, checkClusterConfig bool) (*management.Cluster, error) {
	currentNodePoolNumber := len(cluster.AKSConfig.NodePools)

	upgradedCluster := cluster
	updatedNodePoolsList := cluster.AKSConfig.NodePools[:1]
	upgradedCluster.AKSConfig.NodePools = updatedNodePoolsList

	var err error
	cluster, err = client.Management.Cluster.Update(cluster, &upgradedCluster)
	Expect(err).To(BeNil())

	if checkClusterConfig {
		// Check if the desired config is set correctly
		Expect(len(cluster.AKSConfig.NodePools)).Should(BeNumerically("==", currentNodePoolNumber-1))
		for i, np := range cluster.AKSConfig.NodePools {
			Expect(np.Name).To(Equal(updatedNodePoolsList[i].Name))
		}
	}
	if wait {
		err = clusters.WaitClusterToBeUpgraded(client, cluster.ID)
		Expect(err).To(BeNil())
	}
	if checkClusterConfig {

		// Check if the desired config has been applied in Rancher
		Eventually(func() int {
			ginkgo.GinkgoLogr.Info("Waiting for the total nodepool count to decrease in AKSStatus.UpstreamSpec ...")
			cluster, err = client.Management.Cluster.ByID(cluster.ID)
			Expect(err).To(BeNil())
			return len(cluster.AKSStatus.UpstreamSpec.NodePools)
		}, tools.SetTimeout(12*time.Minute), 10*time.Second).Should(BeNumerically("==", currentNodePoolNumber-1))
		for i, np := range cluster.AKSStatus.UpstreamSpec.NodePools {
			Expect(np.Name).To(Equal(updatedNodePoolsList[i].Name))
		}
	}
	return cluster, nil
}

// ScaleNodePool modifies the number of initialNodeCount of all the nodepools as defined by nodeCount;
// if wait is set to true, it will wait until the cluster finishes upgrading;
// if checkClusterConfig is set to true, it will validate that nodepool has been scaled successfully
func ScaleNodePool(cluster *management.Cluster, client *rancher.Client, nodeCount int64, wait, checkClusterConfig bool) (*management.Cluster, error) {
	upgradedCluster := cluster
	for i := range upgradedCluster.AKSConfig.NodePools {
		upgradedCluster.AKSConfig.NodePools[i].Count = pointer.Int64(nodeCount)
	}

	var err error
	cluster, err = client.Management.Cluster.Update(cluster, &upgradedCluster)
	Expect(err).To(BeNil())

	if checkClusterConfig {
		// Check if the desired config is set correctly
		for i := range cluster.AKSConfig.NodePools {
			Expect(*cluster.AKSConfig.NodePools[i].Count).To(BeNumerically("==", nodeCount))
		}
	}

	if wait {
		err = clusters.WaitClusterToBeUpgraded(client, cluster.ID)
		Expect(err).To(BeNil())
	}

	if checkClusterConfig {
		// check that the desired config is applied on Rancher
		Eventually(func() bool {
			ginkgo.GinkgoLogr.Info("Waiting for the node count change to appear in AKSStatus.UpstreamSpec ...")
			cluster, err = client.Management.Cluster.ByID(cluster.ID)
			Expect(err).To(BeNil())
			for i := range cluster.AKSStatus.UpstreamSpec.NodePools {
				if *cluster.AKSStatus.UpstreamSpec.NodePools[i].Count != nodeCount {
					return false
				}
			}
			return true
		}, tools.SetTimeout(12*time.Minute), 10*time.Second).Should(BeTrue())
	}

	return cluster, nil
}

// ListAKSAvailableVersions is a function to list and return only available AKS versions for a specific cluster.
func ListAKSAvailableVersions(client *rancher.Client, clusterID string) ([]string, error) {
	// kubernetesversions.ListAKSAvailableVersions expects cluster.Version.GitVersion to be available, which it is not sometimes, so we fetch the cluster again to ensure it has all the available data
	cluster, err := client.Management.Cluster.ByID(clusterID)
	if err != nil {
		return nil, err
	}
	allAvailableVersions, err := kubernetesversions.ListAKSAvailableVersions(client, cluster)
	if err != nil {
		return nil, err
	}
	return helpers.FilterUIUnsupportedVersions(allAvailableVersions, client), nil
}

// Create Azure AKS cluster using AZ CLI
func CreateAKSClusterOnAzure(location string, clusterName string, k8sVersion string, nodes string, tags map[string]string) error {
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

// defaultAKS returns the default AKS version used by Rancher; if forUpgrade is true, it returns the second-highest minor k8s version
func defaultAKS(client *rancher.Client, cloudCredentialID, region string, forUpgrade bool) (defaultAKS string, err error) {
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

	maxValue := helpers.HighestK8sMinorVersionSupportedByUI(client)

	// Iterate in the reverse order to get the highest version
	// We obtain the value similar to UI; ref: https://github.com/rancher/ui/blob/master/lib/shared/addon/components/cluster-driver/driver-azureaks/component.js#L140
	// For upgrade tests, it returns a variant of the second-highest minor version
	for i := len(versions) - 1; i >= 0; i-- {
		version := versions[i]
		// If UI maxValue not yet supported by operator
		if !strings.Contains(version, maxValue) {
			maxValue = versions[len(versions)-1]
		}

		if forUpgrade {
			if result := helpers.VersionCompare(version, maxValue); result == -1 {
				return version, nil
			}
		} else {
			if strings.Contains(version, maxValue) {
				return version, nil
			}
		}
	}

	return
}

// GetK8sVersion returns the k8s version to be used by the test;
// this value can either be a variant of envvar DOWNSTREAM_K8S_MINOR_VERSION or the default UI value returned by defaultAKS.
func GetK8sVersion(client *rancher.Client, cloudCredentialID, region string, forUpgrade bool) (string, error) {
	if k8sMinorVersion := helpers.DownstreamK8sMinorVersion; k8sMinorVersion != "" {
		return GetK8sVersionVariantAKS(k8sMinorVersion, client, cloudCredentialID, region)
	}
	return defaultAKS(client, cloudCredentialID, region, forUpgrade)
}
