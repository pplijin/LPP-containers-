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

package support_matrix_test

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"testing"

	"github.com/rancher/shepherd/extensions/clusters/kubernetesversions"

	"github.com/rancher/hosted-providers-e2e/hosted/helpers"
)

var (
	availableVersionList []string
	ctx                  helpers.Context
	region               = helpers.GetEKSRegion()
)

func TestSupportMatrix(t *testing.T) {
	RegisterFailHandler(Fail)
	var err error
	ctx, err = helpers.CommonBeforeSuite("eks")
	Expect(err).To(BeNil())
	availableVersionList, err = kubernetesversions.ListEKSAllVersions(ctx.RancherClient)
	Expect(err).To(BeNil())
	Expect(availableVersionList).ToNot(BeEmpty())
	RunSpecs(t, "SupportMatrix Suite")
}
