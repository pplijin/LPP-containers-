package k8s_chart_support_test

import (
	"fmt"
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

	"github.com/rancher/hosted-providers-e2e/hosted/aks/helper"
	"github.com/rancher/hosted-providers-e2e/hosted/helpers"
)

var (
	ctx                     helpers.Context
	clusterName, k8sVersion string
	testCaseID              int64
	location                = helpers.GetAKSLocation()
)

func TestK8sChartSupport(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "K8sChartSupport Suite")
}

var _ = SynchronizedBeforeSuite(func() []byte {
	helpers.CommonSynchronizedBeforeSuite()
	return nil
}, func() {
	Expect(helpers.Kubeconfig).ToNot(BeEmpty())

	By("Adding the necessary chart repos", func() {
		helpers.AddRancherCharts()
	})

	ctx = helpers.CommonBeforeSuite()
})

var _ = BeforeEach(func() {
	var err error
	clusterName = namegen.AppendRandomString(helpers.ClusterNamePrefix)
	k8sVersion, err = helper.GetK8sVersion(ctx.RancherAdminClient, ctx.CloudCredID, location, false)
	Expect(err).To(BeNil())
	Expect(k8sVersion).ToNot(BeEmpty())
	GinkgoLogr.Info(fmt.Sprintf("Using AKS version %s for cluster %s", k8sVersion, clusterName))

})

var _ = AfterEach(func() {

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

func commonchecks(client *rancher.Client, cluster *management.Cluster) {
	var originalChartVersion string

	By("checking the chart version", func() {
		originalChartVersion = helpers.GetCurrentOperatorChartVersion()
		Expect(originalChartVersion).ToNot(BeEmpty())
		GinkgoLogr.Info("Original chart version: " + originalChartVersion)
	})

	var downgradedVersion string
	By("obtaining a version to downgrade", func() {
		downgradedVersion = helpers.GetDowngradeOperatorChartVersion(originalChartVersion)
		Expect(downgradedVersion).ToNot(BeEmpty())
		GinkgoLogr.Info("Downgrading to version: " + downgradedVersion)
	})

	By("downgrading the chart version", func() {
		helpers.DowngradeProviderChart(downgradedVersion)
	})

	By("making a change to the cluster to validate functionality after chart downgrade", func() {
		initialNodeCount := cluster.NodeCount
		var err error
		cluster, err = helper.ScaleNodePool(cluster, client, initialNodeCount+1, true, true)
		Expect(err).To(BeNil())
	})

	By("uninstalling the operator chart", func() {
		helpers.UninstallOperatorCharts()
	})

	By("making a change(adding a nodepool) to the cluster to re-install the operator and validating it is re-installed to the latest/original version", func() {
		currentNodePoolNumber := len(cluster.AKSConfig.NodePools)
		var err error
		cluster, err = helper.AddNodePool(cluster, 1, client, false, false)
		Expect(err).To(BeNil())
		Expect(len(cluster.AKSConfig.NodePools)).To(BeNumerically("==", currentNodePoolNumber+1))

		By("ensuring that the chart is re-installed to the latest/original version", func() {
			helpers.WaitUntilOperatorChartInstallation(originalChartVersion, "", 0)
		})

		By("ensuring that rancher is up", func() {
			helpers.CheckRancherPods(false)
		})

		err = clusters.WaitClusterToBeUpgraded(client, cluster.ID)
		Expect(err).To(BeNil())
		// Check if the desired config has been applied in Rancher
		Eventually(func() int {
			cluster, err = client.Management.Cluster.ByID(cluster.ID)
			Expect(err).To(BeNil())
			return len(cluster.AKSStatus.UpstreamSpec.NodePools)
		}, tools.SetTimeout(10*time.Minute), 3*time.Second).Should(BeNumerically("==", currentNodePoolNumber+1))
	})
}
