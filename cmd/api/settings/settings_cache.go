package settings

import "time"

type SettingsCache struct {
	storeInterval time.Duration
	maxAge        time.Duration
}

func (s *SettingsCache) GetStoreInterval() time.Duration {
	return s.storeInterval
}

func (s *SettingsCache) SetStoreInterval(value time.Duration) {
	s.storeInterval = value
}

func (s *SettingsCache) GetMaxAge() time.Duration {
	return s.maxAge
}

func (s *SettingsCache) SetMaxAge(value time.Duration) {
	s.maxAge = value
}
