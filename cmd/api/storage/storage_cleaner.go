package storage

import (
	"context"
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
				c.infoLog.Println("Run cleanup job")
				if err := c.cleanupOnce(ctx); err != nil {
					c.errLog.Printf("Cleanup job has failed with error %v", err)
				}
				c.infoLog.Println("Cleanup has finished")
			}
		}
	}()
}

func (c *StorageCleaner) cleanupOnce(ctx context.Context) error {
	for {
		cutOffTime := time.Now().Add(-c.settings.GetMaxAge())
		res, err := c.storage.DB.ExecContext(ctx, `
        DELETE FROM measurement WHERE rowid IN (
            SELECT id FROM measurement WHERE "timestamp" < ? LIMIT 500
        )
        `, cutOffTime)
		if err != nil {
			return err
		}
		n, err := res.RowsAffected()
		if err != nil {
			return err
		}
		c.infoLog.Printf("Cleanup %d records with timestamp after %s", n, cutOffTime.Format(time.RFC3339))
		if n == 0 {
			return nil
		}
	}
}
