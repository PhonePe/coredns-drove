package drovedns

import (
	"fmt"

	"github.com/coredns/caddy"
	"github.com/coredns/coredns/core/dnsserver"
	"github.com/coredns/coredns/plugin"
	clog "github.com/coredns/coredns/plugin/pkg/log"
)

var pluginName = "drove"

var log = clog.NewWithPlugin(pluginName)

// init registers this plugin.
func init() { plugin.Register(pluginName, setup) }

// setup is the function that gets called when the config parser see the token "example". Setup is responsible
// for parsing any extra options the example plugin may have. The first token this function sees is "example".
func setup(c *caddy.Controller) error {

	handler, err := parseAndCreate(c)
	if err != nil {
		return err
	}
	// Add the Plugin to CoreDNS, so Servers can use it in their plugin chain.
	dnsserver.GetConfig(c).AddPlugin(func(next plugin.Handler) plugin.Handler {
		handler.Next = next
		return handler

	})

	return nil
}

func parseAndCreate(c *caddy.Controller) (*DroveHandler, error) {
	c.Next() // Ignore "example" and give us the next token.
	config := NewDroveConfig()
	for c.NextBlock() {
		switch c.Val() {
		case "endpoint":
			args := c.RemainingArgs()
			if len(args) != 1 {
				return nil, c.ArgErr()
			}
			config.Endpoint = args[0]
		case "access_token":
			args := c.RemainingArgs()
			if len(args) != 1 {
				return nil, c.ArgErr()
			}
			config.AuthConfig.AccessToken = args[0]

		case "user_pass":
			args := c.RemainingArgs()
			if len(args) != 2 {
				return nil, c.ArgErr()
			}
			config.AuthConfig.User, config.AuthConfig.Pass = args[0], args[1]
		case "skip_ssl_check":
			config.SkipSSL = true
		default:
			return nil, fmt.Errorf("Drove: Unknown argument %s found", c.Val())
		}
	}

	if err := config.Validate(); err != nil {
		return nil, err
	}

	drove_client := NewDroveClient(config)
	drove_client.Init()
	return NewDroveHandler(&drove_client), nil
}
