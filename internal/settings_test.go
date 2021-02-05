package internal_test

import (
	. "github.com/naveego/plugin-oracle/internal"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("Settings", func() {

	var (
		settings *Settings
	)

	BeforeEach(func() {
		settings = GetTestSettings()
	})

	Describe("Validate", func() {

		It("Should be ok if connection string is set", func() {
			settings = &Settings{
				StringWithPassword: &SettingsStringWithPassword{
					ConnectionString: "test-connection-string:PASSWORD",
					Password: "pass",
					DisableDiscoverAllSchemas: true,
				},
			}
			Expect(settings.Validate()).To(Succeed())
			Expect(settings.GetConnectionString()).To(Equal("test-connection-string:pass"))
			Expect(settings.ShouldDisableDiscoverAll()).To(Equal(true))
		})

		It("Should error if server is not set", func() {
			settings.Form.Hostname = ""
			Expect(settings.Validate()).ToNot(Succeed())
		})

		It("Should error if database is not set", func() {
			settings.Form.Port = 0
			Expect(settings.Validate()).ToNot(Succeed())
		})


		It("Should error if username is not set", func() {
			settings.Form.Username = ""
			Expect(settings.Validate()).ToNot(Succeed())
		})

		It("Should error if password is not set", func() {
			settings.Form.Password = ""
			Expect(settings.Validate()).ToNot(Succeed())
		})

		It("Should succeed if settings are valid for sql", func() {
			Expect(settings.Validate()).To(Succeed())
		})
	})
})
