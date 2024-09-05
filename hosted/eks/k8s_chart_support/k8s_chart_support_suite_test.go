package k8s_chart_support_test

import (
	"fmt"
	"testing"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/rancher-sandbox/ele-testhelpers/kubectl"
	"github.com/rancher-sandbox/ele-testhelpers/tools"
	"github.com/rancher/shepherd/clients/rancher"
	"github.com/rancher/shepherd/clients/rancher/catalog"
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
	ctx         helpers.Context
	clusterName string
	region      = helpers.GetEKSRegion()
	k8sVersion  string
)

func TestK8sChartSupport(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "K8sChartSupport Suite")
}

var _ = BeforeSuite(func() {
	Expect(helpers.RancherVersion).ToNot(BeEmpty())
	// For upgrade tests, the rancher version should not be an unreleased version (for e.g. 2.8-head)
	Expect(helpers.RancherVersion).ToNot(ContainSubstring("head"))

	Expect(helpers.RancherUpgradeVersion).ToNot(BeEmpty())
	Expect(helpers.K8sUpgradedMinorVersion).ToNot(BeEmpty())
	Expect(helpers.Kubeconfig).ToNot(BeEmpty())

	By("Adding the necessary chart repos", func() {
		err := kubectl.RunHelmBinaryWithCustomErr("repo", "add", catalog.RancherChartRepo, "https://charts.rancher.io")
		Expect(err).To(BeNil())
		err = kubectl.RunHelmBinaryWithCustomErr("repo", "add", fmt.Sprintf("rancher-%s", helpers.RancherChannel), fmt.Sprintf("https://releases.rancher.com/server-charts/%s", helpers.RancherChannel))
		Expect(err).To(BeNil())
	})

	By(fmt.Sprintf("Installing Rancher Manager v%s", helpers.RancherVersion), func() {
		helpers.DeployRancherManager(helpers.RancherVersion, true)
	})

})

var _ = BeforeEach(func() {
	var err error
	ctx, err = helpers.CommonBeforeSuite(helpers.Provider)
	Expect(err).To(BeNil())
	clusterName = namegen.AppendRandomString(helpers.ClusterNamePrefix)
	k8sVersion, err = helper.DefaultEKS(ctx.RancherClient)
	Expect(err).To(BeNil())
	Expect(k8sVersion).ToNot(BeEmpty())
	GinkgoLogr.Info(fmt.Sprintf("Using EKS version %s", k8sVersion))

})

var _ = AfterEach(func() {
	// The test must restore the env to its original state, so we install rancher back to its original version and uninstall the operator charts
	// Restoring rancher back to its original state is necessary because in case DOWNSTREAM_CLUSTER_CLEANUP is set to false; in which case clusters will be retained for the next test.
	// Once the operator is uninstalled, it might be reinstalled since the cluster exists, and installing rancher back to its original state ensures that the version is not the one we want to test.
	By(fmt.Sprintf("Installing Rancher back to its original version %s", helpers.RancherVersion), func() {
		helpers.DeployRancherManager(helpers.RancherVersion, true)
	})

	By("Uninstalling the existing operator charts", func() {
		charts := helpers.ListOperatorChart()
		for _, chart := range charts {
			args := []string{"uninstall", chart.Name, "--namespace", helpers.CattleSystemNS}
			err := kubectl.RunHelmBinaryWithCustomErr(args...)
			Expect(err).To(BeNil())
		}
	})
})

func commonchecks(ctx *helpers.Context, cluster *management.Cluster, clusterName, rancherUpgradedVersion, hostname, k8sUpgradedVersion string) {
	var newChartVersion, oldChartVersion string

	By("checking cluster name is same", func() {
		Expect(cluster.Name).To(BeEquivalentTo(clusterName))
	})

	By("checking service account token secret", func() {
		success, err := clusters.CheckServiceAccountTokenSecret(ctx.RancherClient, clusterName)
		Expect(err).To(BeNil())
		Expect(success).To(BeTrue())
	})

	By("checking all management nodes are ready", func() {
		err := nodestat.AllManagementNodeReady(ctx.RancherClient, cluster.ID, helpers.Timeout)
		Expect(err).To(BeNil())
	})

	By("checking all pods are ready", func() {
		podErrors := pods.StatusPods(ctx.RancherClient, cluster.ID)
		Expect(podErrors).To(BeEmpty())
	})

	By("checking the chart version", func() {
		oldCharts := helpers.ListOperatorChart()
		oldChartVersion = oldCharts[0].DerivedVersion
		Expect(oldChartVersion).ToNot(BeEmpty())
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
			rancherClient, err := rancher.NewClient(rancherConfig.AdminToken, ctx.Session)
			Expect(err).To(BeNil())
			ctx.RancherClient = rancherClient

			setting := new(management.Setting)
			resp, err := rancherClient.Management.Setting.ByID("server-url")
			Expect(err).To(BeNil())

			setting.Source = "env"
			setting.Value = fmt.Sprintf("https://%s", hostname)
			resp, err = rancherClient.Management.Setting.Update(resp, setting)
			Expect(err).To(BeNil())

			var isConnected bool
			isConnected, err = ctx.RancherClient.IsConnected()
			Expect(err).To(BeNil())
			Expect(isConnected).To(BeTrue())
		})
	})

	By("making sure the local cluster is ready", func() {
		localClusterID := "local"
		By("checking all management nodes are ready", func() {
			err := nodestat.AllManagementNodeReady(ctx.RancherClient, localClusterID, helpers.Timeout)
			Expect(err).To(BeNil())
		})

		By("checking all pods are ready", func() {
			podErrors := pods.StatusPods(ctx.RancherClient, localClusterID)
			Expect(podErrors).To(BeEmpty())
		})
	})

	By("checking the chart version and validating it is > the old version", func() {

		Eventually(func() int {
			charts := helpers.ListOperatorChart()
			newChartVersion = charts[0].DerivedVersion
			return helpers.VersionCompare(newChartVersion, oldChartVersion)
		}, tools.SetTimeout(1*time.Minute), 5*time.Second).Should(BeNumerically("==", 1))

	})

	var latestVersion *string
	By(fmt.Sprintf("fetching a list of available k8s versions and ensure the v%s is present in the list and upgrading the cluster to it", k8sUpgradedVersion), func() {
		versions, err := helper.ListEKSAvailableVersions(ctx.RancherClient, cluster.ID)
		Expect(err).To(BeNil())
		Expect(versions).ToNot(BeEmpty())
		latestVersion = &versions[0]
		Expect(*latestVersion).To(ContainSubstring(k8sUpgradedVersion))
		Expect(helpers.VersionCompare(*latestVersion, cluster.Version.GitVersion)).To(BeNumerically("==", 1))

		cluster, err = helper.UpgradeClusterKubernetesVersion(cluster, latestVersion, ctx.RancherClient)
		Expect(err).To(BeNil())
		err = clusters.WaitClusterToBeUpgraded(ctx.RancherClient, cluster.ID)
		Expect(err).To(BeNil())
		Expect(cluster.EKSConfig.KubernetesVersion).To(BeEquivalentTo(latestVersion))
	})

	By("downgrading the chart version", func() {
		newCharts := helpers.ListOperatorChart()
		for _, chart := range newCharts {
			// helm requires 2 args
			err := kubectl.RunHelmBinaryWithCustomErr("upgrade", "--install", chart.Name, fmt.Sprintf("rancher-charts/%s", chart.Name), "--namespace", helpers.CattleSystemNS, "--version", oldChartVersion, "--wait")
			Expect(err).To(BeNil())
		}
		// wait until the downgraded chart version is same as the old version
		Eventually(func() int {
			charts := helpers.ListOperatorChart()
			downgradedChartVersion := charts[0].DerivedVersion
			return helpers.VersionCompare(downgradedChartVersion, oldChartVersion)
		}, tools.SetTimeout(1*time.Minute), 5*time.Second).Should(BeNumerically("==", 0))

	})

	By("making a change to the cluster to validate functionality after chart downgrade", func() {
		var err error
		cluster, err = helper.UpgradeNodeKubernetesVersion(cluster, latestVersion, ctx.RancherClient)
		Expect(err).To(BeNil())
		if !cluster.EKSConfig.Imported {
			// TODO Does not upgrade version for imported cluster, since they use custom Launch Templates
			err = clusters.WaitClusterToBeUpgraded(ctx.RancherClient, cluster.ID)
			Expect(err).To(BeNil())
			for _, ng := range cluster.EKSConfig.NodeGroups {
				Expect(ng.Version).To(BeEquivalentTo(latestVersion))
			}
		}
	})

	By("uninstalling the operator chart", func() {
		charts := helpers.ListOperatorChart()
		for _, chart := range charts {
			args := []string{"uninstall", chart.Name, "--namespace", helpers.CattleSystemNS}
			err := kubectl.RunHelmBinaryWithCustomErr(args...)
			Expect(err).To(BeNil())
		}
	})

	By("making a change(adding a nodepool) to the cluster to re-install the operator and validating it is re-installed to the latest version", func() {
		currentNodeGroupNumber := len(cluster.EKSConfig.NodeGroups)
		var err error
		cluster, err = helper.AddNodeGroup(cluster, 1, ctx.RancherClient)
		Expect(err).To(BeNil())

		By("ensuring that the chart is re-installed to the latest version", func() {
			Eventually(func() int {
				charts := helpers.ListOperatorChart()
				if len(charts) == 0 {
					return 10
				}
				reinstalledChartVersion := charts[0].DerivedVersion
				return helpers.VersionCompare(reinstalledChartVersion, newChartVersion)
			}, tools.SetTimeout(1*time.Minute), 5*time.Second).Should(BeNumerically("==", 0))

		})

		err = clusters.WaitClusterToBeUpgraded(ctx.RancherClient, cluster.ID)
		Expect(err).To(BeNil())
		Expect(len(cluster.EKSConfig.NodeGroups)).To(BeNumerically("==", currentNodeGroupNumber+1))
	})

}
