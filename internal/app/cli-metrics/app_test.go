package cli_metrics

import (
	"go-marathon-team-3/pkg/tfsmetrics/azure"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestReadConfigFile(t *testing.T) {
	configpath := "../../../configs/config.json"
	_, err := ReadConfigFile(&configpath)
	assert.NoError(t, err)
}

func TestWriteConfigFile(t *testing.T) {
	config := azure.NewConfig()
	config.OrganizationUrl = "url.com"
	config.Token = "12345"
	configpath := "../../../configs/config.json"
	err := WriteConfigFile(&configpath, config)
	assert.NoError(t, err)
	readConfig, err := ReadConfigFile(&configpath)
	assert.Equal(t, config.OrganizationUrl, readConfig.OrganizationUrl)
	assert.Equal(t, config.Token, readConfig.Token)
}
