package config

type UIConfig struct {
	ListSnippetLines       int `toml:"list_snippet_lines"`
	RefreshIntervalSeconds int `toml:"refresh_interval_seconds"`
}

func (u UIConfig) WithDefaults() UIConfig {
	if u.ListSnippetLines <= 0 {
		u.ListSnippetLines = 2
	}
	if u.RefreshIntervalSeconds == 0 {
		u.RefreshIntervalSeconds = 300
	}
	return u
}
