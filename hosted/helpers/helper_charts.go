package helpers

import (
	"encoding/json"
	"fmt"
	"os/exec"
	"strings"
	"time"

	"github.com/blang/semver"
	"github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/rancher-sandbox/ele-testhelpers/kubectl"
	"github.com/rancher-sandbox/ele-testhelpers/tools"
	"github.com/rancher/shepherd/clients/rancher/catalog"
)

// AddRancherCharts adds the repo from which rancher can be installed
func AddRancherCharts() {
	err := kubectl.RunHelmBinaryWithCustomErr("repo", "add", catalog.RancherChartRepo, "https://charts.rancher.io")
	Expect(err).To(BeNil())
	err = kubectl.RunHelmBinaryWithCustomErr("repo", "add", fmt.Sprintf("rancher-%s", RancherChannel), fmt.Sprintf("https://releases.rancher.com/server-charts/%s", RancherChannel))
	Expect(err).To(BeNil())
}

// GetCurrentOperatorChartVersion returns the current version of a Provider chart.
func GetCurrentOperatorChartVersion() string {
	charts := ListOperatorChart()
	if len(charts) == 0 {
		return ""
	}
	return charts[0].DerivedVersion
}

// GetDowngradeOperatorChartVersion returns a version to downgrade to from a given chart version.
func GetDowngradeOperatorChartVersion(currentChartVersion string) string {
	var chartName string
	if charts := ListOperatorChart(); len(charts) > 0 {
		chartName = charts[0].Name
	} else {
		ginkgo.GinkgoLogr.Info("Could not find downgrade chart; chart is not installed")
		return ""
	}
	chartVersions := ListChartVersions(chartName)
	for _, chartVersion := range chartVersions {
		if VersionCompare(chartVersion.DerivedVersion, currentChartVersion) == -1 {
			return chartVersion.DerivedVersion
		}
	}
	return ""
}

func DowngradeProviderChart(downgradeChartVersion string) {
	currentChartVersion := GetCurrentOperatorChartVersion()
	Expect(currentChartVersion).ToNot(BeEmpty())
	UpdateOperatorChartsVersion(downgradeChartVersion)
	Expect(VersionCompare(downgradeChartVersion, currentChartVersion)).To(Equal(-1))
}

// WaitUntilOperatorChartInstallation waits until the current operator chart version compares to the input chartVersion using the comparator.
// comparator values can be >,<,<=,>=,==,!=
// compareTo value can be 0 if current == input; -1 if current < input; 1 if current > input; defaults to 0
// compareTo is helpful in case either of the chart version is unknown;
// for e.g. after a rancher upgrade, if only the old chart version is known then we can wait until the current chart version is greater than it
func WaitUntilOperatorChartInstallation(chartVersion, comparator string, compareTo int) {
	if comparator == "" {
		comparator = "=="
	}

	if !(compareTo >= -1 && compareTo <= 1) {
		compareTo = 0
	}
	Eventually(func() int {
		currentChartVersion := GetCurrentOperatorChartVersion()
		if currentChartVersion == "" {
			return 10
		}
		return VersionCompare(currentChartVersion, chartVersion)
	}, tools.SetTimeout(1*time.Minute), 5*time.Second).Should(BeNumerically(comparator, compareTo))

}

// UpdateOperatorChartsVersion updates the operator charts to a given chart version and validates that the current version is same as provided
func UpdateOperatorChartsVersion(updateChartVersion string) {
	for _, chart := range ListOperatorChart() {
		err := kubectl.RunHelmBinaryWithCustomErr("upgrade", "--install", chart.Name, fmt.Sprintf("%s/%s", catalog.RancherChartRepo, chart.Name), "--namespace", CattleSystemNS, "--version", updateChartVersion, "--wait")
		if err != nil {
			Expect(err).To(BeNil(), "UpdateOperatorChartsVersion Failed")
		}
	}
	Expect(VersionCompare(GetCurrentOperatorChartVersion(), updateChartVersion)).To(Equal(0))
}

// UninstallOperatorCharts uninstalls the operator charts
func UninstallOperatorCharts() {
	for _, chart := range ListOperatorChart() {
		args := []string{"uninstall", chart.Name, "--namespace", CattleSystemNS}
		err := kubectl.RunHelmBinaryWithCustomErr(args...)
		if err != nil {
			Expect(err).To(BeNil(), "Failed to uninstall chart %s", chart.Name)
		}
	}
}

// ListOperatorChart lists the installed provider charts for a provider in cattle-system; it fetches the provider value using Provider
func ListOperatorChart() (operatorCharts []HelmChart) {
	cmd := exec.Command("helm", "list", "--namespace", CattleSystemNS, "-o", "json", "--filter", fmt.Sprintf("%s-operator", Provider))
	output, err := cmd.Output()
	Expect(err).To(BeNil(), "Failed to list chart %s", Provider)
	ginkgo.GinkgoLogr.Info(string(output))
	err = json.Unmarshal(output, &operatorCharts)
	Expect(err).To(BeNil(), "Failed to unmarshal chart %s", Provider)
	for i := range operatorCharts {
		operatorCharts[i].DerivedVersion = strings.TrimPrefix(operatorCharts[i].Chart, fmt.Sprintf("%s-", operatorCharts[i].Name))
	}
	return
}

// ListChartVersions lists all the available the chart version for a given chart name
func ListChartVersions(chartName string) (charts []HelmChart) {
	cmd := exec.Command("helm", "search", "repo", chartName, "--versions", "-ojson", "--devel")
	output, err := cmd.Output()
	Expect(err).To(BeNil())
	ginkgo.GinkgoLogr.Info(string(output))
	err = json.Unmarshal(output, &charts)
	Expect(err).To(BeNil())
	return
}

// VersionCompare compares Versions v to o:
// -1 == v is less than o
// 0 == v is equal to o
// 1 == v is greater than o
func VersionCompare(v, o string) int {
	latestVer, err := semver.ParseTolerant(v)
	Expect(err).To(BeNil())
	oldVer, err := semver.ParseTolerant(o)
	Expect(err).To(BeNil())
	return latestVer.Compare(oldVer)
}
