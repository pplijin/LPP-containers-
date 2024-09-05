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
	"fmt"
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	. "github.com/rancher-sandbox/qase-ginkgo"
	management "github.com/rancher/shepherd/clients/rancher/generated/management/v3"
	namegen "github.com/rancher/shepherd/pkg/namegenerator"

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
	var err error
	clusterName = namegen.AppendRandomString(helpers.ClusterNamePrefix)
	k8sVersion, err = helper.GetK8sVersion(ctx.RancherAdminClient, project, ctx.CloudCred.ID, zone, "", false)
	Expect(err).To(BeNil())
	GinkgoLogr.Info(fmt.Sprintf("Using kubernetes version %s for cluster %s", k8sVersion, clusterName))
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
func updateLoggingAndMonitoringServiceCheck(ctx helpers.Context, cluster *management.Cluster, updateMonitoringValue, updateLoggingValue string) {
	var err error
	cluster, err = helper.UpdateMonitoringAndLoggingService(cluster, ctx.RancherAdminClient, updateMonitoringValue, updateLoggingValue, true, true)
	Expect(err).To(BeNil())
}

// updateAutoScaling tests updating `autoscaling` for GKE node pools
func updateAutoScaling(ctx helpers.Context, cluster *management.Cluster, autoscale bool) {
	for _, np := range cluster.GKEConfig.NodePools {
		if np.Autoscaling != nil {
			Expect(np.Autoscaling.Enabled).ToNot(BeEquivalentTo(autoscale))
		}
	}

	var err error
	cluster, err = helper.UpdateAutoScaling(cluster, ctx.RancherAdminClient, autoscale, true, true)
	Expect(err).To(BeNil())
}
