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

package p0_test

import (
	"fmt"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/rancher/shepherd/clients/rancher"
	management "github.com/rancher/shepherd/clients/rancher/generated/management/v3"

	"github.com/rancher/hosted-providers-e2e/hosted/eks/helper"
	"github.com/rancher/hosted-providers-e2e/hosted/helpers"
)

var _ = Describe("P0Importing", func() {
	for _, testData := range []struct {
		qaseID    int64
		isUpgrade bool
		testBody  func(cluster *management.Cluster, client *rancher.Client, clusterName string)
		testTitle string
	}{
		{
			qaseID:    234,
			isUpgrade: false,
			testBody:  p0NodesChecks,
			testTitle: "should successfully provision the cluster & add, delete, scale nodepool",
		},
		{
			qaseID:    73,
			isUpgrade: true,
			testBody:  p0upgradeK8sVersionChecks,
			testTitle: "should be able to upgrade k8s version of the imported cluster",
		},
	} {
		testData := testData
		When("a cluster is created", func() {
			var cluster *management.Cluster

			BeforeEach(func() {
				k8sVersion, err := helper.GetK8sVersion(ctx.RancherAdminClient, testData.isUpgrade)
				Expect(err).To(BeNil())
				GinkgoLogr.Info(fmt.Sprintf("Using K8s version %s for cluster %s", k8sVersion, clusterName))
				err = helper.CreateEKSClusterOnAWS(region, clusterName, k8sVersion, "1", helpers.GetCommonMetadataLabels())
				Expect(err).To(BeNil())

				cluster, err = helper.ImportEKSHostedCluster(ctx.RancherAdminClient, clusterName, ctx.CloudCred.ID, region)
				Expect(err).To(BeNil())
				cluster, err = helpers.WaitUntilClusterIsReady(cluster, ctx.RancherAdminClient)
				Expect(err).To(BeNil())
				// Workaround to add new Nodegroup till https://github.com/rancher/aks-operator/issues/251 is fixed
				cluster.EKSConfig = cluster.EKSStatus.UpstreamSpec
			})
			AfterEach(func() {
				if ctx.ClusterCleanup && cluster != nil {
					err := helper.DeleteEKSHostCluster(cluster, ctx.RancherAdminClient)
					Expect(err).To(BeNil())
					err = helper.DeleteEKSClusterOnAWS(region, clusterName)
					Expect(err).To(BeNil())
				} else {
					fmt.Println("Skipping downstream cluster deletion: ", clusterName)
				}
			})

			It(testData.testTitle, func() {
				testCaseID = testData.qaseID
				testData.testBody(cluster, ctx.RancherAdminClient, clusterName)
			})
		})
	}
})
