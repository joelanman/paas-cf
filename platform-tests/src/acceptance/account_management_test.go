package acceptance_test

import (
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	. "github.com/onsi/gomega/gexec"

	"encoding/json"
	"io/ioutil"
	"net/http"
	"net/url"

	"github.com/cloudfoundry-incubator/cf-test-helpers/cf"
	"github.com/cloudfoundry-incubator/cf-test-helpers/generator"
)

var _ = Describe("AccountManagement", func() {
	const email = "the-multi-cloud-paas-team+account-test@digital.cabinet-office.gov.uk"

	var (
		params   url.Values
		password string
		authURL  *url.URL
	)

	BeforeEach(func() {
		params = url.Values{}
		params.Set("client_id", "")
		params.Set("redirect_uri", "")

		password = generator.PrefixedRandomName("CATS-PASSWORD-")

		infoCommand := cf.Cf("curl", "/v2/info")
		Expect(infoCommand.Wait(DEFAULT_TIMEOUT)).To(Exit(0))

		var infoResp struct {
			AuthorizationEndpoint string `json:"authorization_endpoint"`
		}

		err := json.Unmarshal(infoCommand.Buffer().Contents(), &infoResp)
		Expect(err).NotTo(HaveOccurred())

		authURL, err = url.Parse(infoResp.AuthorizationEndpoint)
		Expect(err).NotTo(HaveOccurred())
	})

	Describe("login server /create_account endpoint", func() {
		It("should not allow access to the create account page", func() {
			createAccountURL := authURL
			createAccountURL.Path = "/create_account"

			resp, err := httpClient.Get(createAccountURL.String())
			Expect(err).NotTo(HaveOccurred())

			defer resp.Body.Close()
			body, err := ioutil.ReadAll(resp.Body)
			Expect(err).NotTo(HaveOccurred())
			Expect(resp.StatusCode).To(Equal(http.StatusNotFound), "wrong status code, body:\n\n %s", body)
			Expect(body).To(ContainSubstring("Something went amiss"))
		})

		It("should not allow anonymous users to create accounts", func() {
			createAccountURL := authURL
			createAccountURL.Path = "/create_account.do"

			params.Set("email", email)
			params.Set("password", password)
			params.Set("password_confirmation", password)

			resp, err := httpClient.PostForm(createAccountURL.String(), params)
			Expect(err).NotTo(HaveOccurred())

			defer resp.Body.Close()
			body, err := ioutil.ReadAll(resp.Body)
			Expect(err).NotTo(HaveOccurred())
			Expect(resp.StatusCode).To(Equal(http.StatusNotFound), "wrong status code, body:\n\n %s", body)
			Expect(body).To(ContainSubstring("Something went amiss"))
		})
	})

	Describe("login server /forgot_password endpoint", func() {
		It("should not allow anonymous users to initiate password resets", func() {
			resetPasswordURL := authURL
			resetPasswordURL.Path = "/forgot_password.do"

			params.Set("email", email)

			resp, err := httpClient.PostForm(resetPasswordURL.String(), params)
			Expect(err).NotTo(HaveOccurred())

			defer resp.Body.Close()
			body, err := ioutil.ReadAll(resp.Body)
			Expect(err).NotTo(HaveOccurred())
			Expect(resp.StatusCode).To(Equal(http.StatusNotFound), "wrong status code, body:\n\n %s", body)
			Expect(body).To(ContainSubstring("Something went amiss"))
		})
		It("should not allow anonymous users to initiate password resets page", func() {
			resetPasswordURL := authURL
			resetPasswordURL.Path = "/forgot_password"

			params.Set("email", email)

			resp, err := httpClient.Get(resetPasswordURL.String())
			Expect(err).NotTo(HaveOccurred())

			defer resp.Body.Close()
			body, err := ioutil.ReadAll(resp.Body)
			Expect(err).NotTo(HaveOccurred())
			Expect(resp.StatusCode).To(Equal(http.StatusNotFound), "wrong status code, body:\n\n %s", body)
			Expect(body).To(ContainSubstring("Something went amiss"))
		})
	})
})