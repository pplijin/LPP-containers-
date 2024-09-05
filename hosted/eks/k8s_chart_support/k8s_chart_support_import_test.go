package k8s_chart_support_test

import (
	"fmt"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	management "github.com/rancher/shepherd/clients/rancher/generated/management/v3"
	"github.com/rancher/shepherd/extensions/clusters/eks"
	"github.com/rancher/shepherd/pkg/config"

	"github.com/rancher/hosted-providers-e2e/hosted/eks/helper"
	"github.com/rancher/hosted-providers-e2e/hosted/helpers"
)

var _ = Describe("K8sChartSupportImport", func() {
	var cluster *management.Cluster
	BeforeEach(func() {
		var err error
		err = helper.CreateEKSClusterOnAWS(region, clusterName, k8sVersion, "1")
		Expect(err).To(BeNil())

		eksConfig := new(helper.ImportClusterConfig)
		config.LoadAndUpdateConfig(eks.EKSClusterConfigConfigurationFileKey, eksConfig, func() {
			eksConfig.Region = region
			tags := helper.GetTags()
			eksConfig.Tags = &tags
		})

		cluster, err = helper.ImportEKSHostedCluster(ctx.RancherClient, clusterName, ctx.CloudCred.ID, false, false, false, false, map[string]string{})
		Expect(err).To(BeNil())
		cluster, err = helpers.WaitUntilClusterIsReady(cluster, ctx.RancherClient)
		Expect(err).To(BeNil())
		//Workaround to add new Nodegroup till https://github.com/rancher/aks-operator/issues/251 is fixed
		cluster.EKSConfig = cluster.EKSStatus.UpstreamSpec

	})
	AfterEach(func() {
		if ctx.ClusterCleanup {
			err := helper.DeleteEKSHostCluster(cluster, ctx.RancherClient)
			Expect(err).To(BeNil())
			err = helper.DeleteEKSClusterOnAWS(region, clusterName)
			Expect(err).To(BeNil())
			// TODO: Force delete EKS cluster
		} else {
			fmt.Println("Skipping downstream cluster deletion: ", clusterName)
		}
	})
	It(fmt.Sprintf("should successfully test k8s %s chart support on rancher %s", helpers.K8sUpgradedMinorVersion, helpers.RancherUpgradeVersion), func() {
		commonchecks(&ctx, cluster, clusterName, helpers.RancherUpgradeVersion, helpers.RancherHostname, helpers.K8sUpgradedMinorVersion)
	})
	//	TODO: Automate It(fmt.Sprintf("should successfully test k8s %s chart support importing on upgraded rancher %s", k8sUpgradedMinorVersion, rancherUpgradedVersion)
})
