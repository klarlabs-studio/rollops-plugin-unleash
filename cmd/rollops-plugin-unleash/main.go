// Command rollops-plugin-unleash is a Rollops feature-flag provider plugin
// backed by Unleash. Build it, pin its sha256, and point a rollout's
// featureFlags.plugin at the binary.
package main

import (
	"fmt"
	"os"

	unleash "github.com/klarlabs-studio/rollops-plugin-unleash"
	"go.klarlabs.de/rollops/pkg/plugin"
)

// version is overwritten at build time via -ldflags.
var version = "dev"

func main() {
	safety := plugin.Safety{
		// The plugin reaches the Unleash Admin API; operators set the concrete
		// host via UNLEASH_API_URL and allow-list it in their pluginhost policy.
		NetworkHosts: []string{"unleash.example.com:443"},
		EnvVars:      []string{"UNLEASH_API_URL", "UNLEASH_TOKEN", "UNLEASH_PROJECT"},
		RiskClass:    plugin.RiskActive,
	}
	if err := plugin.ServeFlagProvider("klarlabs/unleash", version, unleash.FromEnv(), safety); err != nil {
		fmt.Fprintln(os.Stderr, "rollops-plugin-unleash:", err)
		os.Exit(1)
	}
}
