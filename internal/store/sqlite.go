package store

import (
	"database/sql"
	"fmt"
	"time"

	_ "modernc.org/sqlite"

	"github.com/YongStack/dai-demo/internal/model"
)

// Store 封装 SQLite 持久化操作
type Store struct {
	db *sql.DB
}

// NewStore 创建并初始化 SQLite 存储
func NewStore(dbPath string) (*Store, error) {
	db, err := sql.Open("sqlite", dbPath)
	if err != nil {
		return nil, fmt.Errorf("open sqlite: %w", err)
	}
	db.SetMaxOpenConns(1) // SQLite 单写
	s := &Store{db: db}
	if err := s.init(); err != nil {
		db.Close()
		return nil, err
	}
	return s, nil
}

func (s *Store) init() error {
	_, err := s.db.Exec(`
		CREATE TABLE IF NOT EXISTS buckets (
			bucket            TEXT PRIMARY KEY,
			quota_capacity    INTEGER NOT NULL DEFAULT 0,
			quota_read_bps    INTEGER NOT NULL DEFAULT 0,
			quota_write_bps   INTEGER NOT NULL DEFAULT 0,
			quota_read_iops   INTEGER NOT NULL DEFAULT 0,
			quota_write_iops  INTEGER NOT NULL DEFAULT 0,
			used_capacity     INTEGER NOT NULL DEFAULT 0,
			used_read_bps     INTEGER NOT NULL DEFAULT 0,
			used_write_bps    INTEGER NOT NULL DEFAULT 0,
			used_read_iops    INTEGER NOT NULL DEFAULT 0,
			used_write_iops   INTEGER NOT NULL DEFAULT 0,
			object_count      INTEGER NOT NULL DEFAULT 0,
			created_at        DATETIME NOT NULL,
			updated_at        DATETIME NOT NULL
		);
		CREATE TABLE IF NOT EXISTS objects (
			id         INTEGER PRIMARY KEY AUTOINCREMENT,
			bucket     TEXT NOT NULL,
			key        TEXT NOT NULL,
			size       INTEGER NOT NULL,
			created_at DATETIME NOT NULL,
			UNIQUE(bucket, key),
			FOREIGN KEY (bucket) REFERENCES buckets(bucket)
		);
	`)
	return err
}

// Close 关闭数据库连接
func (s *Store) Close() error {
	return s.db.Close()
}

// CreateBucket 插入新桶记录
func (s *Store) CreateBucket(b *model.Bucket) error {
	_, err := s.db.Exec(`
		INSERT INTO buckets (bucket, quota_capacity, quota_read_bps, quota_write_bps,
			quota_read_iops, quota_write_iops, used_capacity, used_read_bps, used_write_bps,
			used_read_iops, used_write_iops, object_count, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, 0, 0, 0, 0, 0, 0, ?, ?)`,
		b.Name, b.QuotaCapacity, b.QuotaReadBPS, b.QuotaWriteBPS,
		b.QuotaReadIOPS, b.QuotaWriteIOPS,
		b.CreatedAt, b.UpdatedAt,
	)
	return err
}

// GetBucket 查询单个桶
func (s *Store) GetBucket(name string) (*model.Bucket, error) {
	b := &model.Bucket{}
	err := s.db.QueryRow(`
		SELECT bucket, quota_capacity, quota_read_bps, quota_write_bps,
			quota_read_iops, quota_write_iops, used_capacity, used_read_bps,
			used_write_bps, used_read_iops, used_write_iops, object_count,
			created_at, updated_at
		FROM buckets WHERE bucket = ?`, name,
	).Scan(
		&b.Name, &b.QuotaCapacity, &b.QuotaReadBPS, &b.QuotaWriteBPS,
		&b.QuotaReadIOPS, &b.QuotaWriteIOPS, &b.UsedCapacity, &b.UsedReadBPS,
		&b.UsedWriteBPS, &b.UsedReadIOPS, &b.UsedWriteIOPS, &b.ObjectCount,
		&b.CreatedAt, &b.UpdatedAt,
	)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	return b, err
}

// ListBuckets 列出所有桶
func (s *Store) ListBuckets() ([]*model.Bucket, error) {
	rows, err := s.db.Query(`
		SELECT bucket, quota_capacity, quota_read_bps, quota_write_bps,
			quota_read_iops, quota_write_iops, used_capacity, used_read_bps,
			used_write_bps, used_read_iops, used_write_iops, object_count,
			created_at, updated_at
		FROM buckets ORDER BY created_at`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var buckets []*model.Bucket
	for rows.Next() {
		b := &model.Bucket{}
		if err := rows.Scan(
			&b.Name, &b.QuotaCapacity, &b.QuotaReadBPS, &b.QuotaWriteBPS,
			&b.QuotaReadIOPS, &b.QuotaWriteIOPS, &b.UsedCapacity, &b.UsedReadBPS,
			&b.UsedWriteBPS, &b.UsedReadIOPS, &b.UsedWriteIOPS, &b.ObjectCount,
			&b.CreatedAt, &b.UpdatedAt,
		); err != nil {
			return nil, err
		}
		buckets = append(buckets, b)
	}
	return buckets, rows.Err()
}

// DeleteBucket 删除桶及其关联对象记录
func (s *Store) DeleteBucket(name string) error {
	tx, err := s.db.Begin()
	if err != nil {
		return err
	}
	if _, err := tx.Exec("DELETE FROM objects WHERE bucket = ?", name); err != nil {
		tx.Rollback()
		return err
	}
	if _, err := tx.Exec("DELETE FROM buckets WHERE bucket = ?", name); err != nil {
		tx.Rollback()
		return err
	}
	return tx.Commit()
}

// AddObject 添加对象记录并更新已用容量
func (s *Store) AddObject(bucket, key string, size int64) error {
	tx, err := s.db.Begin()
	if err != nil {
		return err
	}
	now := time.Now()
	if _, err := tx.Exec(
		"INSERT INTO objects (bucket, key, size, created_at) VALUES (?, ?, ?, ?)",
		bucket, key, size, now,
	); err != nil {
		tx.Rollback()
		return err
	}
	if _, err := tx.Exec(
		"UPDATE buckets SET used_capacity = used_capacity + ?, object_count = object_count + 1, updated_at = ? WHERE bucket = ?",
		size, now, bucket,
	); err != nil {
		tx.Rollback()
		return err
	}
	return tx.Commit()
}

// ListObjects 列出桶内所有对象
func (s *Store) ListObjects(bucket string) ([]model.ObjectMeta, error) {
	rows, err := s.db.Query(
		"SELECT key, size, bucket, created_at FROM objects WHERE bucket = ? ORDER BY created_at DESC",
		bucket,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var objects []model.ObjectMeta
	for rows.Next() {
		o := model.ObjectMeta{}
		if err := rows.Scan(&o.Key, &o.Size, &o.Bucket, &o.CreatedAt); err != nil {
			return nil, err
		}
		objects = append(objects, o)
	}
	return objects, rows.Err()
}

// DeleteObject 删除单个对象并扣减已用容量
func (s *Store) DeleteObject(bucket, key string) error {
	tx, err := s.db.Begin()
	if err != nil {
		return err
	}
	var size int64
	err = tx.QueryRow("SELECT size FROM objects WHERE bucket = ? AND key = ?", bucket, key).Scan(&size)
	if err == sql.ErrNoRows {
		tx.Rollback()
		return fmt.Errorf("object not found")
	}
	if err != nil {
		tx.Rollback()
		return err
	}
	if _, err := tx.Exec("DELETE FROM objects WHERE bucket = ? AND key = ?", bucket, key); err != nil {
		tx.Rollback()
		return err
	}
	if _, err := tx.Exec(
		"UPDATE buckets SET used_capacity = used_capacity - ?, object_count = object_count - 1, updated_at = ? WHERE bucket = ?",
		size, time.Now(), bucket,
	); err != nil {
		tx.Rollback()
		return err
	}
	return tx.Commit()
}
