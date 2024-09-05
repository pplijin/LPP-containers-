/*
Copyright © 2023 - 2024 SUSE LLC

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at
    http://www.apache.org/licenses/LICENSE-2.0
Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package p1_test

import (
	"testing"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/rancher-sandbox/ele-testhelpers/tools"
	. "github.com/rancher-sandbox/qase-ginkgo"
	"github.com/rancher/shepherd/clients/rancher"
	management "github.com/rancher/shepherd/clients/rancher/generated/management/v3"
	"github.com/rancher/shepherd/extensions/clusters"
	namegen "github.com/rancher/shepherd/pkg/namegenerator"
	"k8s.io/utils/pointer"

	"github.com/rancher/hosted-providers-e2e/hosted/gke/helper"
	"github.com/rancher/hosted-providers-e2e/hosted/helpers"
)

var (
	ctx                     helpers.Context
	clusterName, k8sVersion string
	testCaseID              int64
	zone                    = helpers.GetGKEZone()
	project                 = helpers.GetGKEProjectID()
)

func TestP1(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "P1 Suite")
}

var _ = SynchronizedBeforeSuite(func() []byte {
	helpers.CommonSynchronizedBeforeSuite()
	return nil
}, func() {
	ctx = helpers.CommonBeforeSuite()
})

var _ = BeforeEach(func() {
	clusterName = namegen.AppendRandomString(helpers.ClusterNamePrefix)
})

var _ = ReportBeforeEach(func(report SpecReport) {
	// Reset case ID
	testCaseID = -1
})

var _ = ReportAfterEach(func(report SpecReport) {
	// Add result in Qase if asked
	Qase(testCaseID, report)
})

// updateLoggingAndMonitoringServiceCheck tests updating `loggingService` and `monitoringService`
func updateLoggingAndMonitoringServiceCheck(cluster *management.Cluster, client *rancher.Client, updateMonitoringValue, updateLoggingValue string) {
	var err error
	cluster, err = helper.UpdateMonitoringAndLoggingService(cluster, client, updateMonitoringValue, updateLoggingValue, true, true)
	Expect(err).To(BeNil())
}

// updateAutoScaling tests updating `autoscaling` for GKE node pools
func updateAutoScaling(cluster *management.Cluster, client *rancher.Client, autoscale bool) {
	for _, np := range cluster.GKEConfig.NodePools {
		if np.Autoscaling != nil {
			Expect(np.Autoscaling.Enabled).ToNot(BeEquivalentTo(autoscale))
		}
	}

	var err error
	cluster, err = helper.UpdateAutoScaling(cluster, client, autoscale, true, true)
	Expect(err).To(BeNil())
}

func syncK8sVersionUpgradeCheck(cluster *management.Cluster, client *rancher.Client) {
	availableVersions, err := helper.ListGKEAvailableVersions(client, cluster.ID)
	Expect(err).To(BeNil())
	upgradeToVersion := availableVersions[0]
	GinkgoLogr.Info("Upgrading to version " + upgradeToVersion)

	By("upgrading control plane", func() {
		currentVersion := cluster.GKEConfig.KubernetesVersion

		err = helper.UpgradeGKEClusterOnGCloud(zone, clusterName, project, upgradeToVersion, false, "")
		Expect(err).To(BeNil())
		// The cluster errors out and becomes unavailable at some point due to the upgrade , so we wait until the cluster is ready
		cluster, err = helpers.WaitUntilClusterIsReady(cluster, client)
		Expect(err).To(BeNil())
		Eventually(func() string {
			GinkgoLogr.Info("Waiting for k8s upgrade to appear in GKEStatus.UpstreamSpec ...")
			cluster, err = client.Management.Cluster.ByID(cluster.ID)
			Expect(err).To(BeNil())
			return *cluster.GKEStatus.UpstreamSpec.KubernetesVersion
		}, tools.SetTimeout(7*time.Minute), 10*time.Second).Should(Equal(upgradeToVersion), "Failed while waiting for k8s upgrade to appear in GKEStatus.UpstreamSpec")

		// Ensure nodepool version is still the same.
		for _, np := range cluster.GKEStatus.UpstreamSpec.NodePools {
			Expect(np.Version).To(BeEquivalentTo(currentVersion))
		}

		if !helpers.IsImport {
			// For imported clusters, GKEConfig always has null values; so we check GKEConfig only when testing provisioned clusters
			Expect(*cluster.GKEConfig.KubernetesVersion).To(Equal(upgradeToVersion))
			for _, np := range cluster.GKEConfig.NodePools {
				Expect(np.Version).To(BeEquivalentTo(currentVersion), "GKEConfig.NodePools check failed")
			}
		}
	})

	By("upgrading the node pool", func() {
		for _, np := range cluster.GKEStatus.UpstreamSpec.NodePools {
			err = helper.UpgradeGKEClusterOnGCloud(zone, clusterName, project, upgradeToVersion, true, *np.Name)
			Expect(err).To(BeNil())
		}

		Eventually(func() bool {
			GinkgoLogr.Info("Waiting for the nodepool upgrade to appear in GKEStatus.UpstreamSpec ...")

			cluster, err = client.Management.Cluster.ByID(cluster.ID)
			Expect(err).To(BeNil())
			for _, np := range cluster.GKEStatus.UpstreamSpec.NodePools {
				if *np.Version != upgradeToVersion {
					return false
				}
			}
			return true
		}, tools.SetTimeout(5*time.Minute), 10*time.Second).Should(BeTrue(), "GKEStatus.UpstreamSpec.NodePools upgrade check failed")

		if !helpers.IsImport {
			// For imported clusters, GKEConfig always has null values; so we check GKEConfig only when testing provisioned clusters
			for _, np := range cluster.GKEConfig.NodePools {
				Expect(*np.Version).To(BeEquivalentTo(upgradeToVersion), "GKEConfig.NodePools upgrade check failed")
			}
		}
	})
}

func syncNodepoolsCheck(cluster *management.Cluster, client *rancher.Client) {

	var poolName = namegen.AppendRandomString("new-np")
	currentNodeCount := len(cluster.GKEConfig.NodePools)

	By("adding a nodepool", func() {
		err := helper.AddNodePoolOnGCloud(clusterName, zone, project, poolName)
		Expect(err).To(BeNil())

		// The cluster does not go into updating state, so we simply wait until the number of nodepools increases
		Eventually(func() int {
			GinkgoLogr.Info("Waiting for the total nodepool count to increase in GKEStatus.UpstreamSpec ...")

			cluster, err = client.Management.Cluster.ByID(cluster.ID)
			Expect(err).To(BeNil())
			return len(cluster.GKEStatus.UpstreamSpec.NodePools)
		}, tools.SetTimeout(5*time.Minute), 5*time.Second).Should(Equal(currentNodeCount + 1))

		// check that the new node pool has been added
		Expect(func() bool {
			for _, np := range cluster.GKEStatus.UpstreamSpec.NodePools {
				if *np.Name == poolName {
					return true
				}
			}
			return false

		}()).To(BeTrue(), "GKEStatus.UpstreamSpec.NodePools increase check failed")

		if !helpers.IsImport {
			// For imported clusters, GKEConfig always has null values; so we check GKEConfig only when testing provisioned clusters
			Expect(len(cluster.GKEConfig.NodePools)).To(Equal(currentNodeCount + 1))

			Expect(func() bool {
				for _, np := range cluster.GKEConfig.NodePools {
					if *np.Name == poolName {
						return true
					}
				}
				return false
			}()).To(BeTrue(), "GKEConfig.NodePools increase check failed")
		}
	})

	By("deleting the nodepool", func() {
		err := helper.DeleteNodePoolOnGCloud(zone, project, clusterName, poolName)
		Expect(err).To(BeNil())

		// The cluster does not go into updating state, so we simply wait until the number of nodepools decreases
		Eventually(func() int {
			cluster, err = client.Management.Cluster.ByID(cluster.ID)
			Expect(err).To(BeNil())
			return len(cluster.GKEStatus.UpstreamSpec.NodePools)
		}, tools.SetTimeout(5*time.Minute), 2*time.Second).Should(Equal(currentNodeCount))

		// check that the new node pool has been deleted
		Expect(func() bool {
			for _, np := range cluster.GKEStatus.UpstreamSpec.NodePools {
				if *np.Name == poolName {
					return true
				}
			}
			return false

		}()).To(BeFalse(), "GKEStatus.UpstreamSpec.NodePools decrease check failed")

		if !helpers.IsImport {
			// For imported clusters, GKEConfig always has null values; so we check GKEConfig only when testing provisioned clusters
			Expect(len(cluster.GKEConfig.NodePools)).To(Equal(currentNodeCount))

			Expect(func() bool {
				for _, np := range cluster.GKEConfig.NodePools {
					if *np.Name == poolName {
						return true
					}
				}
				return false
			}()).To(BeFalse(), "GKEConfig.NodePools decrease check failed")
		}
	})
}

// updateClusterInUpdatingState runs checks to ensure cluster in an updating state can be updated
func updateClusterInUpdatingState(cluster *management.Cluster, client *rancher.Client) {
	availableVersions, err := helper.ListGKEAvailableVersions(client, cluster.ID)
	Expect(err).To(BeNil())
	upgradeK8sVersion := availableVersions[0]

	currentNodePoolCount := len(cluster.GKEConfig.NodePools)
	cluster, err = helper.UpgradeKubernetesVersion(cluster, upgradeK8sVersion, client, false, false, false)
	Expect(err).To(BeNil())
	Expect(*cluster.GKEConfig.KubernetesVersion).To(Equal(upgradeK8sVersion))

	err = clusters.WaitClusterToBeInUpgrade(client, cluster.ID)
	Expect(err).To(BeNil())

	cluster, err = helper.AddNodePool(cluster, client, 1, "", false, false)
	Expect(err).To(BeNil())

	Expect(len(cluster.GKEConfig.NodePools)).Should(BeNumerically("==", currentNodePoolCount+1))

	err = clusters.WaitClusterToBeUpgraded(client, cluster.ID)
	Expect(err).To(BeNil())

	Eventually(func() bool {
		GinkgoLogr.Info("Waiting for the changes to appear in GKEStatus.UpstreamSpec ...")
		cluster, err = client.Management.Cluster.ByID(cluster.ID)
		Expect(err).To(BeNil())
		return len(cluster.GKEStatus.UpstreamSpec.NodePools) == currentNodePoolCount+1 && *cluster.GKEStatus.UpstreamSpec.KubernetesVersion == upgradeK8sVersion
	}, "5m", "5s").Should(BeTrue())
}

// combinationMutableParameterUpdate runs checks to test if mutable parameters can be updated in combination
func combinationMutableParameterUpdate(cluster *management.Cluster, client *rancher.Client) {
	const (
		disableService = "none"
		maxCount       = int64(5)
		minCount       = int64(2)
	)
	var err error
	cluster, err = helper.UpdateCluster(cluster, client, func(upgradedCluster *management.Cluster) {
		updatedNp := upgradedCluster.GKEConfig.NodePools
		for i := range updatedNp {
			np := updatedNp[i]
			// update autoscaling and initial node count
			updatedNp[i] = management.GKENodePoolConfig{
				Autoscaling: &management.GKENodePoolAutoscaling{
					Enabled:      true,
					MaxNodeCount: maxCount,
					MinNodeCount: minCount,
				},
				Config:            np.Config,
				InitialNodeCount:  pointer.Int64(minCount),
				Management:        np.Management,
				MaxPodsConstraint: np.MaxPodsConstraint,
				Name:              np.Name,
				Version:           np.Version,
			}
		}

		upgradedCluster.GKEConfig.NodePools = updatedNp
		upgradedCluster.GKEConfig.LoggingService = pointer.String(disableService)
		upgradedCluster.GKEConfig.MonitoringService = pointer.String(disableService)
	})
	Expect(err).To(BeNil())

	Expect(*cluster.GKEConfig.LoggingService).To(Equal(disableService))
	Expect(*cluster.GKEConfig.MonitoringService).To(Equal(disableService))
	for _, np := range cluster.GKEConfig.NodePools {
		Expect(*np.InitialNodeCount).Should(Equal(minCount))
		Expect(np.Autoscaling.Enabled).Should(BeTrue())
		Expect(np.Autoscaling.MaxNodeCount).Should(Equal(maxCount))
		Expect(np.Autoscaling.MinNodeCount).Should(Equal(minCount))
	}

	err = clusters.WaitClusterToBeUpgraded(client, cluster.ID)
	Expect(err).To(BeNil())

	Eventually(func() bool {
		GinkgoLogr.Info("Waiting for the combination changes to appear in GKEStatus.UpstreamSpec...")
		var clusterState *management.Cluster
		clusterState, err = client.Management.Cluster.ByID(cluster.ID)
		Expect(err).To(BeNil())
		if !(*clusterState.GKEStatus.UpstreamSpec.LoggingService == disableService && *clusterState.GKEStatus.UpstreamSpec.MonitoringService == disableService) {
			return false
		}

		for _, np := range clusterState.GKEStatus.UpstreamSpec.NodePools {
			if np.Autoscaling != nil && !(np.Autoscaling.Enabled && np.Autoscaling.MinNodeCount == minCount && np.Autoscaling.MaxNodeCount == maxCount && *np.InitialNodeCount == minCount) {
				return false
			}
		}
		return true
	}, "5m", "5s").Should(BeTrue())
}
