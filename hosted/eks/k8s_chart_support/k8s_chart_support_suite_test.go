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

	k8sVersion, err = helper.GetK8sVersion(ctx.RancherAdminClient, false)
	Expect(err).To(BeNil())
	Expect(k8sVersion).ToNot(BeEmpty())

	GinkgoLogr.Info(fmt.Sprintf("Using EKS version %s for cluster %s", k8sVersion, clusterName))

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

	initialNodeCount := *cluster.EKSConfig.NodeGroups[0].DesiredSize

	By("making a change(scaling nodegroup up) to the cluster to validate functionality after chart downgrade", func() {
		var err error
		cluster, err = helper.ScaleNodeGroup(cluster, client, initialNodeCount+1, true, true)
		Expect(err).To(BeNil())
	})

	By("uninstalling the operator chart", func() {
		helpers.UninstallOperatorCharts()
	})

	By("making a change(scaling nodegroup down) to the cluster to re-install the operator and validating it is re-installed to the latest/original version", func() {
		var err error
		cluster, err = helper.ScaleNodeGroup(cluster, client, initialNodeCount, false, true)
		Expect(err).To(BeNil())

		By("ensuring that the chart is re-installed to the latest/original version", func() {
			helpers.WaitUntilOperatorChartInstallation(originalChartVersion, "", 0)
		})

		// We do not use WaitClusterToBeUpgraded because it has been flaky here and times out
		var upgradeSuccessful bool
		Eventually(func() bool {
			for i := range cluster.EKSConfig.NodeGroups {
				upgradeSuccessful = *cluster.EKSConfig.NodeGroups[i].DesiredSize == initialNodeCount
			}
			return upgradeSuccessful
		}, tools.SetTimeout(10*time.Minute), 5*time.Second).Should(BeTrue())

	})

}
