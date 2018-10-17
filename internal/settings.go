package internal

import (
	"errors"
	"fmt"
	"gopkg.in/goracle.v2"
)

type Settings struct {
	Hostname             string   `json:"host"`
	Port int `json:"port"`
	ServiceName string `json:"serviceName"`
	Username         string   `json:"username"`
	Password         string   `json:"password"`
	// PrePublishQuery  string   `json:"prePublishQuery"`
	// PostPublishQuery string   `json:"postPublishQuery"`
	ConnectionString string   `json:"connectionString"`
}

type AuthType string
const (
	AuthTypeSID = AuthType("SID")
	AuthTypeServiceName = AuthType("ServiceName")
)


// Validate returns an error if the Settings are not valid.
// It also populates the internal fields of settings.
func (s *Settings) Validate() error {

	if s.ConnectionString != "" {
		return nil
	}

	if s.Hostname == "" {
		return errors.New("the hostname property must be set")
	}

	if s.Port == 0 {
		return errors.New("the port property must be set")
	}

	if s.ServiceName == "" {
		return errors.New("the serviceName property must be set")
	}

	if s.Username == "" {
		return errors.New("the username property must be set")
	}

	if s.Password == "" {
		return errors.New("the password property must be set")
	}

	return nil
}

func (s *Settings) GetConnectionString() (string, error){
	if s.ConnectionString != "" {
		return s.ConnectionString, nil
	}

	err := s.Validate()
	if err != nil {
		return "", err
	}

	sid := fmt.Sprintf("%s:%d/%s", s.Hostname, s.Port, s.ServiceName)

	cp := goracle.ConnectionParams{
		SID:         sid,
		Username:    s.Username,
		Password:    s.Password,
		MinSessions: 1,
		MaxSessions: 10,
		ConnClass:   "POOLED",
		IsSysOper:   true,
	}

	return cp.StringWithPassword(), nil
}