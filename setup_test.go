package drovedns

import (
	"testing"

	"github.com/coredns/caddy"
	"github.com/stretchr/testify/assert"
)

// TestSetup tests the various things that should be parsed by setup.
// Make sure you also test for parse errors.
func TestSetup(t *testing.T) {
	tests := []struct {
		config  string
		error   bool
		message string
	}{
		{
			`drove {
				endpoint http://url.random
				access_token "O-Bearer token"
			}`,
			false,
			"Valid config",
		},

		{
			`drove {
				endpoint http://url.random
				user_pass user pass
			}`,
			false,
			"Valid config",
		},
		{
			`drove {
				endpoint http://url.random 8080
				access_token token
			}`,
			true,
			"Invalid endpoint",
		},
		{
			`drove {
				endpoint http://url.random 
				access_token O-Bearer token
			}`,
			true,
			"Invalid Access token",
		},
		{
			`drove {
				endpoint http://url.random
				user_pass user:blah
			}`,
			true,
			"User pass should be space delimited",
		},
		{
			`drove {
				access_token token
			}`,
			true,
			"Missing endpoint",
		},

		{
			`drove {
				endpoint http://url.random
				access_token token
				user_pass user blah
			}`,
			true,
			"Access TOken and user_pass cant be added",
		},
		{
			`drove {
				endpoint http://url.random
				access_token token
				blaharg
			}`,
			true,
			"Random arg cannot be added",
		},

		{
			`drove {
				endpoint http://url.random
				access_token token
				blaharg
			}`,
			true,
			"Empty Stanza is invalid",
		},
	}

	for _, tt := range tests {
		c := caddy.NewTestController("drovedns", tt.config)
		_, err := parseAndCreate(c)
		if tt.error {
			assert.Error(t, err, tt.message)
		} else {
			assert.NoError(t, err, tt.message)
		}

	}

}
