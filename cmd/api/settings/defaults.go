// Package settings contains defaults
package settings

const (
	SettingKeyMaxAge        = "maxage"
	SettingKeyStoreInterval = "store_interval"
)

var DefaultSettings = map[string]string{
	SettingKeyMaxAge:        "720h",
	SettingKeyStoreInterval: "1m",
}
