package settings

import "time"

type SettingsCache struct {
	storeInterval time.Duration
}

func (s *SettingsCache) GetStoreInterval() time.Duration {
	return s.storeInterval

}

func (s *SettingsCache) SetStoreInteval(value time.Duration) {
	s.storeInterval = value
}
