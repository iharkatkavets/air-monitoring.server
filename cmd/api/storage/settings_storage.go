package storage

import (
	"context"
	"database/sql"
	"errors"
	"time"
)

type SettingItem struct {
	Key       string
	Value     string
	UpdatedAt time.Time
}

type SettingsStorage interface {
	GetAllSettings(ctx context.Context) ([]SettingItem, error)
	GetSetting(ctx context.Context, key string) (*SettingItem, error)
	UpsertSetting(ctx context.Context, key, value string) (*SettingItem, error)
}

func (s *SQLStorage) GetAllSettings(ctx context.Context) ([]SettingItem, error) {
	query := `SELECT key, value, updated_at FROM settings ORDER BY key`
	rows, err := s.DB.QueryContext(ctx, query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []SettingItem
	for rows.Next() {
		var item SettingItem
		if err := rows.Scan(&item.Key, &item.Value, &item.UpdatedAt); err != nil {
			return nil, err
		}
		out = append(out, item)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	return out, nil
}

func (s *SQLStorage) GetSetting(ctx context.Context, key string) (*SettingItem, error) {
	query := `SELECT key, value, updated_at FROM settings WHERE key = ?`

	row := s.DB.QueryRowContext(ctx, query, key)

	var item SettingItem
	if err := row.Scan(&item.Key, &item.Value, &item.UpdatedAt); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil // not found
		}
		return nil, err
	}

	return &item, nil
}

func (s *SQLStorage) UpsertSetting(ctx context.Context, key, value string) (*SettingItem, error) {
	query := `
	INSERT INTO settings (key, value, updated_at)
	VALUES (?, ?, CURRENT_TIMESTAMP)
	ON CONFLICT(key) DO UPDATE
	SET value = excluded.value,
	    updated_at = CURRENT_TIMESTAMP;
	`

	if _, err := s.DB.ExecContext(ctx, query, key, value); err != nil {
		return nil, err
	}

	row := s.DB.QueryRowContext(ctx, `SELECT key, value, updated_at FROM settings WHERE key = ?`, key)

	var item SettingItem
	if err := row.Scan(&item.Key, &item.Value, &item.UpdatedAt); err != nil {
		return nil, err
	}

	return &item, nil
}
