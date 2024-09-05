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
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	. "github.com/rancher-sandbox/qase-ginkgo"

	"github.com/rancher/shepherd/clients/rancher"
	management "github.com/rancher/shepherd/clients/rancher/generated/management/v3"
	namegen "github.com/rancher/shepherd/pkg/namegenerator"

	"github.com/rancher/hosted-providers-e2e/hosted/eks/helper"
	"github.com/rancher/hosted-providers-e2e/hosted/helpers"
)

const (
	increaseBy = 1
)

var (
	ctx         helpers.Context
	clusterName string
	testCaseID  int64
	region      = helpers.GetEKSRegion()
)

func TestP0(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "P0 Suite")
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

func p0upgradeK8sVersionChecks(cluster *management.Cluster, client *rancher.Client, clusterName string) {
	helpers.ClusterIsReadyChecks(cluster, client, clusterName)

	// Default version is highest supported version
	upgradeToVersion, err := helper.GetK8sVersion(client, false)
	Expect(err).To(BeNil())
	GinkgoLogr.Info(fmt.Sprintf("Upgrading cluster to EKS version %s", upgradeToVersion))

	By("upgrading the ControlPlane", func() {
		cluster, err = helper.UpgradeClusterKubernetesVersion(cluster, upgradeToVersion, client, true)
		Expect(err).To(BeNil())
	})

	// Does not upgrades version since using custom LT, skip for imported cluster
	if !helpers.IsImport {
		By("upgrading the NodeGroups", func() {
			cluster, err = helper.UpgradeNodeKubernetesVersion(cluster, upgradeToVersion, client, true, true)
			Expect(err).To(BeNil())
		})
	}
}

func p0NodesChecks(cluster *management.Cluster, client *rancher.Client, clusterName string) {
	helpers.ClusterIsReadyChecks(cluster, client, clusterName)
	initialNodeCount := *cluster.EKSConfig.NodeGroups[0].DesiredSize

	By("scaling up the NodeGroup", func() {
		var err error
		cluster, err = helper.ScaleNodeGroup(cluster, client, initialNodeCount+increaseBy, true, true)
		Expect(err).To(BeNil())
	})

	By("scaling down the NodeGroup", func() {
		var err error
		cluster, err = helper.ScaleNodeGroup(cluster, client, initialNodeCount, true, true)
		Expect(err).To(BeNil())
	})

	By("adding a NodeGroup", func() {
		var err error
		cluster, err = helper.AddNodeGroup(cluster, increaseBy, client, true, true)
		Expect(err).To(BeNil())
	})
	By("deleting the NodeGroup", func() {
		var err error
		cluster, err = helper.DeleteNodeGroup(cluster, client, true, true)
		Expect(err).To(BeNil())
	})
}
