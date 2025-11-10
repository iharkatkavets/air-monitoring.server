package storage

import (
	"context"
	"log"
	"time"
)

type StorageCleaner struct {
	storage *SQLStorage
	infoLog *log.Logger
	errLog  *log.Logger
}

func NewStorageCleaner(storage *SQLStorage, infoLog *log.Logger, errLog *log.Logger) *StorageCleaner {
	return &StorageCleaner{storage: storage, infoLog: infoLog, errLog: errLog}
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
		maxAge, err := c.storage.GetSetting(ctx, "maxage")
		if err != nil {
			return err
		}
		duration, err := time.ParseDuration(maxAge.Value)
		if err != nil {
			return err
		}
		cutOffTime := time.Now().Add(duration)
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
