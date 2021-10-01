package azure

import "context"

type Config struct {
	OrganizationUrl string `json:"organization_url"`
	Token           string `json:"personal_access_token"`
	Context         context.Context `json:"-"`
}

func NewConfig() *Config {
	return &Config{
		Context: context.Background(),
		OrganizationUrl: "",
		Token: "",
	}
}
