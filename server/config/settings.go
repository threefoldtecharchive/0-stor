package config

type Settings struct {
	DebugLog     bool   `json:"debug"`
	BindAddress  string `json:"bind"`
	AuthDisabled bool   `json:"auth_disabled"`
	MaxMsgSize   int    `json:"max_msg_size"`

	DB struct {
		Dirs struct {
			Meta string `json:"meta"`
			Data string `json:"data"`
		} `json:"dirs"`
	} `json:"db"`
}
