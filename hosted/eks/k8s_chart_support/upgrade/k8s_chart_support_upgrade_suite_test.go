package k8s_chart_support_upgrade_test

import (
	"fmt"
	"testing"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/rancher-sandbox/ele-testhelpers/kubectl"
	"github.com/rancher-sandbox/ele-testhelpers/tools"
	. "github.com/rancher-sandbox/qase-ginkgo"
	"github.com/rancher/shepherd/clients/rancher"
	management "github.com/rancher/shepherd/clients/rancher/generated/management/v3"
	"github.com/rancher/shepherd/extensions/clusters"
	nodestat "github.com/rancher/shepherd/extensions/nodes"
	"github.com/rancher/shepherd/extensions/pipeline"
	"github.com/rancher/shepherd/extensions/workloads/pods"
	"github.com/rancher/shepherd/pkg/config"
	namegen "github.com/rancher/shepherd/pkg/namegenerator"

	"github.com/rancher/hosted-providers-e2e/hosted/eks/helper"
	"github.com/rancher/hosted-providers-e2e/hosted/helpers"
)

var (
	ctx                     helpers.Context
	clusterName, k8sVersion string
	region                  = helpers.GetEKSRegion()
	testCaseID              int64
)

func TestK8sChartSupportUpgrade(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "K8sChartSupportUpgrade Suite")
}

var _ = SynchronizedBeforeSuite(func() []byte {
	helpers.CommonSynchronizedBeforeSuite()
	return nil
}, func() {
	Expect(helpers.RancherVersion).ToNot(BeEmpty())
	// For upgrade tests, the rancher version should not be an unreleased version (for e.g. 2.8-head)
	Expect(helpers.RancherVersion).ToNot(ContainSubstring("head"))

	Expect(helpers.RancherUpgradeVersion).ToNot(BeEmpty())
	Expect(helpers.K8sUpgradedMinorVersion).ToNot(BeEmpty())
	Expect(helpers.Kubeconfig).ToNot(BeEmpty())

	By("Adding the necessary chart repos", func() {
		helpers.AddRancherCharts()
	})

	By(fmt.Sprintf("Installing Rancher Manager v%s", helpers.RancherVersion), func() {
		helpers.DeployRancherManager(helpers.RancherVersion, true)
	})

	ctx = helpers.CommonBeforeSuite()
})

var _ = BeforeEach(func() {
	var err error
	clusterName = namegen.AppendRandomString(helpers.ClusterNamePrefix)
	k8sVersion, err = helper.GetK8sVersion(ctx.RancherAdminClient, false)
	Expect(err).To(BeNil())
	Expect(k8sVersion).ToNot(BeEmpty())
	GinkgoLogr.Info(fmt.Sprintf("Using EKS version %s for cluster %s", k8sVersion, clusterName))

})

var _ = AfterEach(func() {
	// The test must restore the env to its original state, so we install rancher back to its original version and uninstall the operator charts
	// Restoring rancher back to its original state is necessary because in case DOWNSTREAM_CLUSTER_CLEANUP is set to false; in which case clusters will be retained for the next test.
	// Once the operator is uninstalled, it might be reinstalled since the cluster exists, and installing rancher back to its original state ensures that the version is not the one we want to test.
	By(fmt.Sprintf("Installing Rancher back to its original version %s", helpers.RancherVersion), func() {
		helpers.DeployRancherManager(helpers.RancherVersion, true)
	})

	By("Uninstalling the existing operator charts", func() {
		helpers.UninstallOperatorCharts()
	})
})

var _ = ReportBeforeEach(func(report SpecReport) {
	// Reset case ID
	testCaseID = -1
})

var _ = ReportAfterEach(func(report SpecReport) {
	// Add result in Qase if asked
	Qase(testCaseID, report)
})

func commonchecks(ctx *helpers.Context, cluster *management.Cluster, clusterName, rancherUpgradedVersion, hostname, k8sUpgradedVersion string) {

	helpers.ClusterIsReadyChecks(cluster, ctx.RancherAdminClient, clusterName)
	var originalChartVersion string
	By("checking the chart version", func() {
		originalChartVersion = helpers.GetCurrentOperatorChartVersion()
		Expect(originalChartVersion).ToNot(BeEmpty())
		GinkgoLogr.Info("Original chart version: " + originalChartVersion)
	})

	By("upgrading rancher", func() {
		helpers.DeployRancherManager(rancherUpgradedVersion, true)
		By("ensuring operator pods are also up", func() {
			Eventually(func() error {
				return kubectl.New().WaitForNamespaceWithPod(helpers.CattleSystemNS, fmt.Sprintf("ke.cattle.io/operator=%s", helpers.Provider))
			}, tools.SetTimeout(4*time.Minute), 30*time.Second).Should(BeNil())
		})
		By("regenerating the token and initiating a new rancher client", func() {
			//	regenerate the tokens and initiate a new rancher client
			rancherConfig := new(rancher.Config)
			config.LoadConfig(rancher.ConfigurationFileKey, rancherConfig)

			token, err := pipeline.CreateAdminToken(helpers.RancherPassword, rancherConfig)
			Expect(err).To(BeNil())

			config.LoadAndUpdateConfig(rancher.ConfigurationFileKey, rancherConfig, func() {
				rancherConfig.AdminToken = token
			})
			rancherAdminClient, err := rancher.NewClient(rancherConfig.AdminToken, ctx.Session)
			Expect(err).To(BeNil())
			ctx.RancherAdminClient = rancherAdminClient

			setting := new(management.Setting)
			resp, err := rancherAdminClient.Management.Setting.ByID("server-url")
			Expect(err).To(BeNil())

			setting.Source = "env"
			setting.Value = fmt.Sprintf("https://%s", hostname)
			resp, err = rancherAdminClient.Management.Setting.Update(resp, setting)
			Expect(err).To(BeNil())

			var isConnected bool
			isConnected, err = ctx.RancherAdminClient.IsConnected()
			Expect(err).To(BeNil())
			Expect(isConnected).To(BeTrue())
		})
	})

	By("making sure the local cluster is ready", func() {
		localClusterID := "local"
		By("checking all management nodes are ready", func() {
			err := nodestat.AllManagementNodeReady(ctx.RancherAdminClient, localClusterID, helpers.Timeout)
			Expect(err).To(BeNil())
		})

		By("checking all pods are ready", func() {
			podErrors := pods.StatusPods(ctx.RancherAdminClient, localClusterID)
			Expect(podErrors).To(BeEmpty())
		})
	})

	var upgradedChartVersion string
	By("checking the chart version and validating it is > the old version", func() {
		// the chart is sometimes auto-upgraded to the latest version (mostly happens when running the test on un-rc-ed charts, so we check with `>=`
		helpers.WaitUntilOperatorChartInstallation(originalChartVersion, ">=", 0)
		upgradedChartVersion = helpers.GetCurrentOperatorChartVersion()
		GinkgoLogr.Info("Upgraded chart version: " + upgradedChartVersion)
	})

	var latestVersion *string
	By(fmt.Sprintf("fetching a list of available k8s versions and ensure the v%s is present in the list and upgrading the cluster to it", k8sUpgradedVersion), func() {
		versions, err := helper.ListEKSAvailableVersions(ctx.RancherAdminClient, cluster.ID)
		Expect(err).To(BeNil())
		Expect(versions).ToNot(BeEmpty())

		latestVersion = &versions[0]
		Expect(*latestVersion).To(ContainSubstring(k8sUpgradedVersion))
		Expect(helpers.VersionCompare(*latestVersion, cluster.Version.GitVersion)).To(BeNumerically("==", 1))

		cluster, err = helper.UpgradeClusterKubernetesVersion(cluster, *latestVersion, ctx.RancherAdminClient, true)
		Expect(err).To(BeNil())
		err = clusters.WaitClusterToBeUpgraded(ctx.RancherAdminClient, cluster.ID)
		Expect(err).To(BeNil())
		Expect(cluster.EKSConfig.KubernetesVersion).To(BeEquivalentTo(latestVersion))
	})

	var downgradeVersion string
	By("fetching a value to downgrade to", func() {
		downgradeVersion = helpers.GetDowngradeOperatorChartVersion(upgradedChartVersion)
	})

	By("downgrading the chart version", func() {
		helpers.DowngradeProviderChart(downgradeVersion)
	})

	By("making a change to the cluster to validate functionality after chart downgrade", func() {
		var err error
		cluster, err = helper.UpgradeNodeKubernetesVersion(cluster, *latestVersion, ctx.RancherAdminClient, true, false)
		Expect(err).To(BeNil())
		if !cluster.EKSConfig.Imported {
			// TODO Does not upgrade version for imported cluster, since they use custom Launch Templates
			err = clusters.WaitClusterToBeUpgraded(ctx.RancherAdminClient, cluster.ID)
			Expect(err).To(BeNil())
			for _, ng := range cluster.EKSConfig.NodeGroups {
				Expect(ng.Version).To(BeEquivalentTo(latestVersion))
			}
		}
	})

	By("uninstalling the operator chart", func() {
		helpers.UninstallOperatorCharts()
	})

	By("making a change(adding a nodepool) to the cluster to re-install the operator and validating it is re-installed to the latest/upgraded version", func() {
		currentNodeGroupNumber := len(cluster.EKSConfig.NodeGroups)
		var err error
		cluster, err = helper.AddNodeGroup(cluster, 1, ctx.RancherAdminClient, false, false)
		Expect(err).To(BeNil())

		By("ensuring that the chart is re-installed to the latest/upgraded version", func() {
			helpers.WaitUntilOperatorChartInstallation(upgradedChartVersion, "", 0)
		})

		err = clusters.WaitClusterToBeUpgraded(ctx.RancherAdminClient, cluster.ID)
		Expect(err).To(BeNil())
		// Check if the desired config has been applied in Rancher
		Eventually(func() int {
			cluster, err = ctx.RancherAdminClient.Management.Cluster.ByID(cluster.ID)
			Expect(err).To(BeNil())
			return len(cluster.EKSStatus.UpstreamSpec.NodeGroups)
		}, tools.SetTimeout(20*time.Minute), 10*time.Second).Should(BeNumerically("==", currentNodeGroupNumber+1))
	})

}
