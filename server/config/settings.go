package config

type Settings struct {
	DebugLog     bool   `json:"debug"`
	BindAddress  string `json:"bind"`
	AuthDisabled bool   `json:"auth_disabled"`
	MaxMsgSize   int    `json:"max_msg_size"`
	AsyncWrite   bool   `json:"async_write`

	DB struct {
		Dirs struct {
			Meta string `json:"meta"`
			Data string `json:"data"`
		} `json:"dirs"`
	} `json:"db"`
}
