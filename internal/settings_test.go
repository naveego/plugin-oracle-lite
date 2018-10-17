package internal_test

import (
	. "github.com/naveego/plugin-oracle/internal"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("Settings", func() {

	var (
		settings Settings
	)

	BeforeEach(func() {
		settings = Settings{
			Hostname:     "10.250.1.10",
			Port:32769,
			ServiceName: "ORCLCDB.localdomain",
			Username: "C##NAVEEGO",
			Password: "test123",
		}
	})

	Describe("Validate", func() {

		It("Should be ok if connection string is set", func() {
			settings = Settings{
				ConnectionString: "test-connection-string",
			}
			Expect(settings.Validate()).To(Succeed())
		})

		It("Should error if server is not set", func() {
			settings.Hostname = ""
			Expect(settings.Validate()).ToNot(Succeed())
		})

		It("Should error if database is not set", func() {
			settings.Port = 0
			Expect(settings.Validate()).ToNot(Succeed())
		})


		It("Should error if username is not set", func() {
			settings.Username = ""
			Expect(settings.Validate()).ToNot(Succeed())
		})

		It("Should error if password is not set", func() {
			settings.Password = ""
			Expect(settings.Validate()).ToNot(Succeed())
		})

		It("Should succeed if settings are valid for sql", func() {
			Expect(settings.Validate()).To(Succeed())
		})


	})
})
