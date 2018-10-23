package configuration

import (
	"fmt"
	"net/http"
	"os"
	"regexp"
	"testing"
	"time"

	"github.com/fabric8-services/fabric8-auth/resource"
	"github.com/goadesign/goa"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var reqLong *goa.RequestData
var reqShort *goa.RequestData
var config *ConfigurationData

func init() {

	// ensure that the content here is executed only once.
	reqLong = &goa.RequestData{
		Request: &http.Request{Host: "api.service.domain.org"},
	}
	reqShort = &goa.RequestData{
		Request: &http.Request{Host: "api.domain.org"},
	}
	resetConfiguration()
}

func resetConfiguration() {
	var err error
	config, err = GetConfigurationData()
	if err != nil {
		panic(fmt.Errorf("Failed to setup the configuration: %s", err.Error()))
	}
}

func TestGetLogLevelOK(t *testing.T) {
	resource.Require(t, resource.UnitTest)

	key := "AUTH_LOG_LEVEL"
	realEnvValue := os.Getenv(key)

	os.Unsetenv(key)
	defer func() {
		os.Setenv(key, realEnvValue)
		resetConfiguration()
	}()

	assert.Equal(t, defaultLogLevel, config.GetLogLevel())

	os.Setenv(key, "warning")
	resetConfiguration()

	assert.Equal(t, "warning", config.GetLogLevel())
}

func TestGetTransactionTimeoutOK(t *testing.T) {
	resource.Require(t, resource.UnitTest)

	key := "AUTH_POSTGRES_TRANSACTION_TIMEOUT"
	realEnvValue := os.Getenv(key)

	os.Unsetenv(key)
	defer func() {
		os.Setenv(key, realEnvValue)
		resetConfiguration()
	}()

	assert.Equal(t, time.Duration(5*time.Minute), config.GetPostgresTransactionTimeout())

	os.Setenv(key, "6m")
	resetConfiguration()

	assert.Equal(t, time.Duration(6*time.Minute), config.GetPostgresTransactionTimeout())
}

func TestValidRedirectURLsInDevModeCanBeOverridden(t *testing.T) {
	resource.Require(t, resource.UnitTest)

	key := "AUTH_REDIRECT_VALID"
	realEnvValue := os.Getenv(key)

	defer func() {
		if realEnvValue != "" {
			os.Setenv(key, realEnvValue)
		} else {
			os.Unsetenv(key)
		}
		resetConfiguration()
	}()
	os.Unsetenv(key)
	assert.Equal(t, devModeValidRedirectURLs, config.GetValidRedirectURLs())

	os.Setenv(key, "https://someDomain.org/redirect")
	resetConfiguration()
	assert.Equal(t, "https://someDomain.org/redirect", config.GetValidRedirectURLs())
}

func TestDefaultRedirectURLs(t *testing.T) {
	resource.Require(t, resource.UnitTest)
	t.Parallel()

	// Valid if requesting prod-preview to redirect to localhost or to openshift.io
	// OR if requesting openshift to redirect to openshift.io
	// Invalid otherwise
	assert.True(t, validateRedirectURL(t, "https://api.prod-preview.openshift.io/api"))
	assert.True(t, validateRedirectURL(t, "http://api.prod-preview.openshift.io/api"))
	assert.True(t, validateRedirectURL(t, "https://api.prod-preview.openshift.io:8080/api"))
	assert.True(t, validateRedirectURL(t, "https://api.openshift.io/api"))
	assert.True(t, validateRedirectURL(t, "https://api.openshift.io:8080/api"))
	assert.True(t, validateRedirectURL(t, "http://api.openshift.io/api"))
	assert.True(t, validateRedirectURL(t, "http://localhost/api"))
	assert.True(t, validateRedirectURL(t, "http://localhost"))
	assert.False(t, validateRedirectURL(t, "https://api.prod-preview.domain.io/api"))
	assert.False(t, validateRedirectURL(t, "https://api.domain.io?redirect=openshift.io"))
	assert.False(t, validateRedirectURL(t, "https://api.domain.io?redirect=api.openshift.io"))
	assert.False(t, validateRedirectURL(t, "https://api.domain.io/api.openshift.io"))
	assert.False(t, validateRedirectURL(t, "https://api.domain.io?api.openshift.io"))
	assert.False(t, validateRedirectURL(t, "https://api.domain.io#api.openshift.io"))
	assert.False(t, validateRedirectURL(t, "https://api.openshift.io.domain/api"))
	assert.False(t, validateRedirectURL(t, "https://api.domain.io?redirect=localhost"))
	assert.False(t, validateRedirectURL(t, "https://api.domain.io?redirect=localhost"))
	assert.False(t, validateRedirectURL(t, "https://api.domain.io/localhost"))
	assert.False(t, validateRedirectURL(t, "https://api.domain.io?localhost"))
	assert.False(t, validateRedirectURL(t, "https://api.domain.io#localhost"))
	assert.False(t, validateRedirectURL(t, "https://localhost.domain/api"))
}

func validateRedirectURL(t *testing.T, redirect string) bool {
	matched, err := regexp.MatchString(DefaultValidRedirectURLs, redirect)
	require.Nil(t, err)
	return matched
}
