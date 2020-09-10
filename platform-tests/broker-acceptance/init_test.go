package broker_acceptance_test

import (
	"crypto/tls"
	"fmt"
	"net/http"
	"os"
	"os/exec"
	"testing"
	"time"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	. "github.com/onsi/gomega/gbytes"
	. "github.com/onsi/gomega/gexec"

	"github.com/cloudfoundry-community/go-cfclient"
	"github.com/cloudfoundry-incubator/cf-test-helpers/cf"
	"github.com/cloudfoundry-incubator/cf-test-helpers/helpers"
	"github.com/cloudfoundry-incubator/cf-test-helpers/workflowhelpers"
	"github.com/cloudfoundry/cf-acceptance-tests/helpers/config"
)

const (
	DB_CREATE_TIMEOUT = 30 * time.Minute
)

var (
	testConfig  config.CatsConfig
	httpClient  *http.Client
	testContext *workflowhelpers.ReproducibleTestSuiteSetup

	systemDomain = os.Getenv("SYSTEM_DNS_ZONE_NAME")

	cfClient *cfclient.Client
)

func TestSuite(t *testing.T) {
	var err error
	RegisterFailHandler(Fail)

	testConfig, err = config.NewCatsConfig(os.Getenv("CONFIG"))
	if err != nil {
		t.Fatal(err)
	}

	httpClient = &http.Client{
		Transport: &http.Transport{
			TLSClientConfig: &tls.Config{InsecureSkipVerify: testConfig.GetSkipSSLValidation()},
		},
	}

	testContext = workflowhelpers.NewTestSuiteSetup(testConfig)

	BeforeSuite(func() {
		testContext.Setup()

		Expect(systemDomain).NotTo(Equal(""))

		var err error
		cfClient, err = cfclient.NewClient(&cfclient.Config{
			ApiAddress: "https://" + testContext.RegularUserContext().ApiUrl,
			Username:   testContext.RegularUserContext().Username,
			Password:   testContext.RegularUserContext().Password,
		})
		Expect(err).NotTo(HaveOccurred())

		// Enable service access for the ephemeral test org.
		// FIXME: remove this block once sqs is enabled for all
		workflowhelpers.AsUser(testContext.AdminUserContext(), testContext.ShortTimeout(), func() {
			standard := cf.Cf("enable-service-access", "aws-sqs-queue",
				"-o", testContext.TestSpace.OrganizationName(),
				"-b", "sqs-broker",
				"-p", "standard",
			).Wait(testConfig.DefaultTimeoutDuration())
			Expect(standard).To(Exit(0))
			fifo := cf.Cf("enable-service-access", "aws-sqs-queue",
				"-o", testContext.TestSpace.OrganizationName(),
				"-b", "sqs-broker",
				"-p", "fifo",
			).Wait(testConfig.DefaultTimeoutDuration())
			Expect(fifo).To(Exit(0))
		})
	})

	AfterSuite(func() {
		testContext.Teardown()
	})

	componentName := "Custom-Acceptance-Tests"
	if testConfig.GetArtifactsDirectory() != "" {
		helpers.EnableCFTrace(testConfig, componentName)
	}

	RunSpecs(t, componentName)
}

// quietCf is an equivelent of cf.Cf that doesn't send the output to
// GinkgoWriter. Used when you don't want the output, even in verbose mode (eg
// when polling the API)
func quietCf(program string, args ...string) *Session {
	command, err := Start(exec.Command(program, args...), nil, nil)
	Expect(err).NotTo(HaveOccurred())
	return command
}

func pollForServiceCreationCompletion(dbInstanceName string) {
	fmt.Fprint(GinkgoWriter, "Polling for service creation to complete")
	Eventually(func() *Buffer {
		fmt.Fprint(GinkgoWriter, ".")
		command := quietCf("cf", "service", dbInstanceName).Wait(testConfig.DefaultTimeoutDuration())
		Expect(command).To(Exit(0), fmt.Sprint("Error calling cf service creation phase: ", string(command.Out.Contents())))
		return command.Out
	}, DB_CREATE_TIMEOUT, 15*time.Second).Should(Say("create succeeded"))
	fmt.Fprint(GinkgoWriter, "done\n")
}

func pollForServiceUpdateCompletion(dbInstanceName string) {
	fmt.Fprint(GinkgoWriter, "Polling for service update to complete")
	Eventually(func() *Buffer {
		fmt.Fprint(GinkgoWriter, ".")
		command := quietCf("cf", "service", dbInstanceName).Wait(testConfig.DefaultTimeoutDuration())
		Expect(command).To(Exit(0), fmt.Sprint("Error calling cf service update phase: ", string(command.Out.Contents())))
		return command.Out
	}, DB_CREATE_TIMEOUT, 15*time.Second).Should(Say("update succeeded"))
	fmt.Fprint(GinkgoWriter, "done\n")
}

func pollForServiceDeletionCompletion(dbInstanceName string) {
	fmt.Fprint(GinkgoWriter, "Polling for service destruction to complete")
	Eventually(func() *Buffer {
		fmt.Fprint(GinkgoWriter, ".")
		command := quietCf("cf", "services").Wait(testConfig.DefaultTimeoutDuration())
		Expect(command).To(Exit(0), fmt.Sprint("Error calling cf services: ", string(command.Out.Contents())))
		return command.Out
	}, DB_CREATE_TIMEOUT, 15*time.Second).ShouldNot(Say(dbInstanceName))
	fmt.Fprint(GinkgoWriter, "done\n")
}

func pollForServiceBound(dbInstanceName, boundAppName string) {
	fmt.Fprint(GinkgoWriter, "Polling for async bind operation to complete")
	Eventually(func() *Buffer {
		fmt.Fprint(GinkgoWriter, ".")
		command := quietCf("cf", "service", dbInstanceName).Wait(testConfig.DefaultTimeoutDuration())
		Expect(command).To(Exit(0), fmt.Sprint("Error calling cf service: ", string(command.Out.Contents())))
		return command.Out
	}, DB_CREATE_TIMEOUT, 5*time.Second).Should(Say(boundAppName))
	Eventually(func() *Buffer {
		fmt.Fprint(GinkgoWriter, ".")
		command := quietCf("cf", "service", dbInstanceName).Wait(testConfig.DefaultTimeoutDuration())
		Expect(command).To(Exit(0), fmt.Sprint("Error calling cf service: ", string(command.Out.Contents())))
		return command.Out
	}, DB_CREATE_TIMEOUT, 5*time.Second).ShouldNot(Say("in progress"))
	fmt.Fprint(GinkgoWriter, "done\n")
}

func pollForServiceUnbound(dbInstanceName, boundAppName string) {
	fmt.Fprint(GinkgoWriter, "Polling for async unbind operation to complete")
	Eventually(func() *Buffer {
		fmt.Fprint(GinkgoWriter, ".")
		command := quietCf("cf", "service", dbInstanceName).Wait(testConfig.DefaultTimeoutDuration())
		Expect(command).To(Exit(0), fmt.Sprint("Error calling cf service: ", string(command.Out.Contents())))
		return command.Out
	}, DB_CREATE_TIMEOUT, 5*time.Second).ShouldNot(Say(boundAppName))
	fmt.Fprint(GinkgoWriter, "done\n")
}

type basicAuthRoundTripper struct {
	username string
	password string
}

func (r basicAuthRoundTripper) RoundTrip(req *http.Request) (*http.Response, error) {
	req.SetBasicAuth(r.username, r.password)
	return http.DefaultTransport.RoundTrip(req)
}
