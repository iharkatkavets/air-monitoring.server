// Package settings contains defaults
package settings

const (
	SettingKeyMaxAge        = "max_age"
	SettingKeyStoreInterval = "store_interval"
)

var DefaultSettings = map[string]string{
	SettingKeyMaxAge:        "2678400", // 60*60*24*31days
	SettingKeyStoreInterval: "60",      // 60sec
}
