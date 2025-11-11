// Package settings contains defaults
package settings

const (
	SettingKeyMaxAge        = "maxage"
	SettingKeyStoreInterval = "store_interval"
)

var DefaultSettings = map[string]string{
	SettingKeyMaxAge:        "2678400",
	SettingKeyStoreInterval: "60",
}
