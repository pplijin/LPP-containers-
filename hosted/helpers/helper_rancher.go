package helpers

import (
	"fmt"
	"strings"
	"time"

	"github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/rancher-sandbox/ele-testhelpers/kubectl"
	rancherhelper "github.com/rancher-sandbox/ele-testhelpers/rancher"
	"github.com/rancher-sandbox/ele-testhelpers/tools"
)

// DeployRancherManager deploys Rancher. If checkPods is true, it waits until all the necessary pods are running
// fullVersion: devel/2.9, 2.9.0, 2.9.0-rc1
func DeployRancherManager(fullVersion string, checkPods bool) {
	channelName := "rancher-" + RancherChannel

	var version, headVersion string
	versionSplit := strings.Split(fullVersion, "/")
	version = versionSplit[0]
	if len(versionSplit) == 2 {
		headVersion = versionSplit[1]
	}

	// Set flags for Rancher Manager installation
	flags := []string{
		"upgrade", "--install", "rancher", channelName + "/rancher",
		"--namespace", CattleSystemNS,
		"--create-namespace",
		"--set", "hostname=" + RancherHostname,
		"--set", "bootstrapPassword=" + RancherPassword,
		"--set", "replicas=1",
		"--set", "global.cattle.psp.enabled=false",
		"--wait",
	}

	// Set specified version if needed
	if version != "" && version != "latest" {
		if version == "devel" {
			flags = append(flags,
				"--devel",
				"--set", "rancherImageTag=v"+headVersion+"-head",
			)
		} else if strings.Contains(version, "-rc") {
			flags = append(flags,
				"--devel",
				"--version", version,
			)
		} else {
			flags = append(flags, "--version", version)
		}
	}

	ginkgo.GinkgoLogr.Info(fmt.Sprintf("Deploying rancher: %v", flags))
	err := kubectl.RunHelmBinaryWithCustomErr(flags...)
	Expect(err).To(BeNil())

	if checkPods {
		CheckRancherPods(true)
	}
}

func CheckRancherPods(wait bool) {
	// Wait for all pods to be started
	checkList := [][]string{
		{CattleSystemNS, "app=rancher"},
		{"cattle-fleet-system", "app=fleet-controller"},
		{CattleSystemNS, "app=rancher-webhook"},
	}
	Eventually(func() error {
		return rancherhelper.CheckPod(kubectl.New(), checkList)
	}, tools.SetTimeout(4*time.Minute), 30*time.Second).Should(BeNil())

	if wait {
		// A bit dirty be better to wait a little here for all to be correctly started
		time.Sleep(2 * time.Minute)
	}
}
