package acceptance_test

import (
	"crypto/sha512"
	"encoding/json"
	"fmt"
	"github.com/PuerkitoBio/goquery"
	"github.com/cloudfoundry-incubator/cf-test-helpers/cf"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	. "github.com/onsi/gomega/gexec"
	"io/ioutil"
	"net/url"
	"strings"
)

var _ = Describe("UAA authorization webpage styling", func() {
	var authURL *url.URL
	var authLoginURL url.URL
	var authStylesheetURL url.URL
	var customAppURL url.URL
	var customLogoURL url.URL
	var customStylesheetURL url.URL

	BeforeEach(func() {
		infoCommand := cf.Cf("curl", "/v2/info")
		Expect(infoCommand.Wait(testConfig.DefaultTimeoutDuration())).To(Exit(0))

		var infoResp struct {
			AuthorizationEndpoint string `json:"authorization_endpoint"`
		}
		err := json.Unmarshal(infoCommand.Buffer().Contents(), &infoResp)
		Expect(err).NotTo(HaveOccurred())

		authURL, err = url.Parse(infoResp.AuthorizationEndpoint)
		Expect(err).NotTo(HaveOccurred())

		authLoginURL = *authURL
		authLoginURL.Path = "/login"

		authStylesheetURL = *authURL
		authStylesheetURL.Path = "/resources/oss/stylesheets/application.css"

		customAppURL = url.URL{
			Scheme: "https",
			Host:   "paas-uaa-assets." + testConfig.AppsDomain,
		}
		customLogoURL = customAppURL
		customLogoURL.Path = "/images/product-logo.png"
		customStylesheetURL = customAppURL
		customStylesheetURL.Path = "/stylesheets/application.css"
	})

	Context("the login page", func() {
		var authLoginDoc *goquery.Document

		BeforeEach(func() {
			response, err := httpClient.Get(authLoginURL.String())
			Expect(err).NotTo(HaveOccurred())
			Expect(response.StatusCode).To(Equal(200))
			authLoginDoc, err = goquery.NewDocumentFromResponse(response)
			Expect(err).NotTo(HaveOccurred())
		})

		It("should be using our custom stylesheet", func() {
			stylesheetHrefs := authLoginDoc.Find("link[rel=stylesheet]").Map(func(_ int, linkTag *goquery.Selection) string {
				href, _ := linkTag.Attr("href")
				return href
			})
			Expect(stylesheetHrefs).To(ConsistOf([]string{
				"/vendor/font-awesome/css/font-awesome.min.css",
				customStylesheetURL.String(),
			}))
		})

		It("should be using our custom logo", func() {
			styleCSS, err := authLoginDoc.Find("style").Html()
			Expect(err).NotTo(HaveOccurred())
			styleCSS = strings.TrimSpace(styleCSS)
			expectedCSS := fmt.Sprintf(".header-image {background-image: url(%s);}", customLogoURL.String())
			Expect(styleCSS).To(Equal(expectedCSS))
		})
	})

	Context("the built-in default stylesheet", func() {
		It("should not have been changed", func() {
			response, err := httpClient.Get(authStylesheetURL.String())
			Expect(err).NotTo(HaveOccurred())
			stylesheet, err := ioutil.ReadAll(response.Body)
			Expect(err).NotTo(HaveOccurred())

			stylesheetChecksum := fmt.Sprintf("%x", sha512.Sum512(stylesheet))
			expectedChecksum := "3ad54656fb0fb874a5ae8929a9959b22f6c82ba134a6bb5bb2fd46d45bbd0377ef863c83a0fcf751b885bbe0b6ee60644f4a70a0847bd4fe34bf6f9e909e1e24"

			failureExplanation := `UAA's default stylesheet has been changed. We use a custom stylesheet deployed in an app, but it might need to change in similar ways.
 - Visit UAA's login and accept invitation pages. Check our styling still looks good.
 - If it looks bad, fix our stylesheet.
 - Once it looks good, update the checksum in this test.`
			Expect(stylesheetChecksum).To(Equal(expectedChecksum), failureExplanation)
		})
	})
})
