package unleash

import "os"

// FromEnv builds a Provider from the plugin's environment. Secrets and endpoint
// come from the plugin process, never from the Rollops target spec (Rollops
// passes only the flag name, environment, and percentage).
//
//	UNLEASH_API_URL   base URL of the Unleash instance (required, e.g. https://unleash.example.com)
//	UNLEASH_TOKEN     Admin API token (required)
//	UNLEASH_PROJECT   project id (default "default")
func FromEnv() Provider {
	return Provider{
		BaseURL: os.Getenv("UNLEASH_API_URL"),
		Token:   os.Getenv("UNLEASH_TOKEN"),
		Project: os.Getenv("UNLEASH_PROJECT"),
	}
}
