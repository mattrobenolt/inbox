package config

type KeyMap struct {
	List             ListKeyMap             `toml:"list"`
	Detail           DetailKeyMap           `toml:"detail"`
	Search           SearchKeyMap           `toml:"search"`
	Attachment       AttachmentKeyMap       `toml:"attachment"`
	Image            ImageKeyMap            `toml:"image"`
	AttachmentsModal AttachmentsModalKeyMap `toml:"attachments_modal"`
}

type ListKeyMap struct {
	Up         []string `toml:"up"`
	Down       []string `toml:"down"`
	PageUp     []string `toml:"page_up"`
	PageDown   []string `toml:"page_down"`
	Open       []string `toml:"open"`
	ToggleRead []string `toml:"toggle_read"`
	Search     []string `toml:"search"`
	Refresh    []string `toml:"refresh"`
	Help       []string `toml:"help"`
	Quit       []string `toml:"quit"`
}

type DetailKeyMap struct {
	Up           []string `toml:"up"`
	Down         []string `toml:"down"`
	ToggleExpand []string `toml:"toggle_expand"`
	ToggleView   []string `toml:"toggle_view"`
	Attachments  []string `toml:"attachments"`
	Back         []string `toml:"back"`
	Help         []string `toml:"help"`
	Quit         []string `toml:"quit"`
}

type SearchKeyMap struct {
	Submit []string `toml:"submit"`
	Cancel []string `toml:"cancel"`
	Quit   []string `toml:"quit"`
}

type AttachmentKeyMap struct {
	Back []string `toml:"back"`
	Quit []string `toml:"quit"`
}

type ImageKeyMap struct {
	Back []string `toml:"back"`
	Quit []string `toml:"quit"`
}

type AttachmentsModalKeyMap struct {
	Up       []string `toml:"up"`
	Down     []string `toml:"down"`
	Download []string `toml:"download"`
	View     []string `toml:"view"`
	Close    []string `toml:"close"`
}
