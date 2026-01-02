package config

type LinkConfig struct {
	UnwrapDomains []string `toml:"unwrap_domains"`
	DoNotResolve  []string `toml:"do_not_resolve"`
	DNSMode       string   `toml:"dns_mode"`
	DNSServers    []string `toml:"dns_servers"`
	AutoScan      bool     `toml:"auto_scan"`
}
