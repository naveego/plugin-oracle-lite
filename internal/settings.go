package internal

import (
	"fmt"
	"github.com/pkg/errors"
	"gopkg.in/goracle.v2"
	"strings"
)

type Settings struct {
	Strategy           SettingsStrategy            `json:"strategy"`
	Form               *SettingsForm               `json:"form"`
	StringWithPassword *SettingsStringWithPassword `json:"stringWithPassword"`
}

type SettingsStrategy string

const StrategyForm = SettingsStrategy("Form")
const StrategyStringWithPassword = SettingsStrategy("Connection String")

type SettingsForm struct {
	Hostname                  string `json:"hostname"`
	Port                      int    `json:"port"`
	ServiceName               string `json:"serviceName"`
	Username                  string `json:"username"`
	Password                  string `json:"password"`
}

type SettingsStringWithPassword struct {
	ConnectionString          string `json:"connectionString"`
	Password                  string `json:"password"`
}

// Validate returns an error if the Settings are not valid.
// It also populates the internal fields of settings.
func (s *Settings) Validate() error {

	if s.Strategy == "" {
		if s.Form != nil {
			s.Strategy = StrategyForm
		} else if s.StringWithPassword != nil {
			s.Strategy = StrategyStringWithPassword
		}
	}

	switch s.Strategy {
	case StrategyForm:
		if s.Form == nil {
			return errors.New("the form property must be set")
		}

		form := s.Form

		if form.Hostname == "" {
			return errors.New("the hostname property must be set")
		}

		if form.Port == 0 {
			return errors.New("the port property must be set")
		}

		if form.ServiceName == "" {
			return errors.New("the serviceName property must be set")
		}

		if form.Username == "" {
			return errors.New("the username property must be set")
		}

		if form.Password == "" {
			return errors.New("the password property must be set")
		}

		return nil
	case StrategyStringWithPassword:
		if s.StringWithPassword == nil {
			return errors.New("the stringWithPassword property must be set")
		}

		if s.StringWithPassword.ConnectionString == "" {
			return errors.New("the stringWithPassword.connectionString property must be set")
		}

		if !strings.Contains(s.StringWithPassword.ConnectionString, "PASSWORD") {
			return errors.New("the connection string should contain 'PASSWORD' in place of the password")
		}

		if s.StringWithPassword.Password == "" {
			return errors.New("the stringWithPassword.password property must be set")
		}

		return nil

	default:
		return errors.Errorf("unrecognized strategy %q", s.Strategy)

	}
}

func (s *Settings) GetConnectionString() (string, error) {
	err := s.Validate()
	if err != nil {
		return "", err
	}

	switch s.Strategy {
	case StrategyForm:

		f := s.Form

		sid := fmt.Sprintf("%s:%d/%s", f.Hostname, f.Port, f.ServiceName)

		cp := goracle.ConnectionParams{
			SID:         sid,
			Username:    f.Username,
			Password:    f.Password,
			MinSessions: 1,
			MaxSessions: 10,
			ConnClass:   "POOLED",
			IsSysOper:   true,
		}

		return cp.StringWithPassword(), nil

	case StrategyStringWithPassword:
		c := strings.Replace(s.StringWithPassword.ConnectionString, "PASSWORD", s.StringWithPassword.Password, 1)
		return c, nil
	default:
		return "", errors.Errorf("unrecognized strategy %q", s.Strategy)
	}

}

func (s *Settings) ShouldDisableDiscoverAll() bool {
	return true
}

func (s *Settings) ShouldDiscoverWrite() bool {
	return false
}
