package k8s_chart_support_test

import (
	"fmt"
	"testing"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/rancher-sandbox/ele-testhelpers/tools"
	"github.com/rancher/shepherd/clients/rancher"
	namegen "github.com/rancher/shepherd/pkg/namegenerator"

	. "github.com/rancher-sandbox/qase-ginkgo"
	management "github.com/rancher/shepherd/clients/rancher/generated/management/v3"
	"github.com/rancher/shepherd/extensions/clusters"

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

	k8sVersion, err = helper.GetK8sVersion(ctx.RancherAdminClient, project, ctx.CloudCred.ID, zone, "", false)
	Expect(err).To(BeNil())

	GinkgoLogr.Info(fmt.Sprintf("Using GKE version %s for cluster %s", k8sVersion, clusterName))
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

// commonChartSupport runs the common checks required for testing chart support
func commonChartSupport(client *rancher.Client, cluster *management.Cluster) {
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
		initialNodeCount := *cluster.GKEConfig.NodePools[0].InitialNodeCount
		var err error
		cluster, err = helper.ScaleNodePool(cluster, client, initialNodeCount+1, true, true)
		Expect(err).To(BeNil())
	})

	By("uninstalling the operator chart", func() {
		helpers.UninstallOperatorCharts()
	})

	By("making a change(adding a nodepool) to the cluster to re-install the operator and validating it is re-installed to the latest/original version", func() {
		currentNodePoolNumber := len(cluster.GKEConfig.NodePools)
		var err error
		cluster, err = helper.AddNodePool(cluster, client, 1, "", false, false)
		Expect(err).To(BeNil())

		Expect(len(cluster.GKEConfig.NodePools)).To(BeNumerically("==", currentNodePoolNumber+1))

		By("ensuring that the chart is re-installed to the latest/original version", func() {
			helpers.WaitUntilOperatorChartInstallation(originalChartVersion, "", 0)
		})

		By("ensuring that rancher is up", func() {
			helpers.CheckRancherPods(false)
		})

		err = clusters.WaitClusterToBeUpgraded(client, cluster.ID)
		Expect(err).To(BeNil())
		Eventually(func() int {
			GinkgoLogr.Info("Waiting for the total nodepool count to increase in GKEStatus.UpstreamSpec ...")
			cluster, err = client.Management.Cluster.ByID(cluster.ID)
			Expect(err).To(BeNil())
			return len(cluster.GKEStatus.UpstreamSpec.NodePools)
		}, tools.SetTimeout(12*time.Minute), 10*time.Second).Should(BeNumerically("==", currentNodePoolNumber+1))

	})
}
