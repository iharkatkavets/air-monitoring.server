package storage

import (
	"context"
	"fmt"
	"log"
	"sensor/cmd/api/settings"
	"time"
)

type StorageCleaner struct {
	storage  *SQLStorage
	infoLog  *log.Logger
	errLog   *log.Logger
	settings *settings.SettingsCache
}

func NewStorageCleaner(storage *SQLStorage, infoLog *log.Logger, errLog *log.Logger, settings *settings.SettingsCache) *StorageCleaner {
	return &StorageCleaner{storage: storage, infoLog: infoLog, errLog: errLog, settings: settings}
}

func (c *StorageCleaner) StartCleanupJob(ctx context.Context, interval time.Duration) {
	go func() {
		ticker := time.NewTicker(interval)
		defer ticker.Stop()

		for {
			select {
			case <-ctx.Done():
				c.infoLog.Println("Cleanup job stopped")
				return
			case <-ticker.C:
				c.infoLog.Println("Timer tick request cleanup op")
				if err := c.performCleanup(ctx); err != nil {
					c.errLog.Printf("Cleanup operation has failed with error %v", err)
					break
				}
				c.infoLog.Println("Cleanup has finished")
			}
		}
	}()
}

func (c *StorageCleaner) performCleanup(ctx context.Context) error {
	maxAge := c.settings.GetMaxAge()
	cutOffTime := time.Now().UTC().Add(-maxAge)

	for {
		res, err := c.storage.DB.ExecContext(ctx, `
        DELETE FROM measurement 
        WHERE id IN (
            SELECT id FROM measurement 
            WHERE "timestamp_unix" < ? 
            ORDER BY "timestamp_unix"
            LIMIT 500
        )
        `, cutOffTime.Unix())
		if err != nil {
			return fmt.Errorf("cleanup delete measurements: %w", err)
		}
		n, err := res.RowsAffected()
		if err != nil {
			return fmt.Errorf("cleanup rows affected: %w", err)
		}
		c.infoLog.Printf("Cleanup %d records with timestamp_unix before %s max_age %s", n, cutOffTime.Format(time.RFC3339), maxAge.Round(time.Second))
		if n == 0 {
			return nil
		}
	}
}
