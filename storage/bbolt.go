package storage

import (
	"bytes"
	"errors"
	"os"
	"time"

	bbolt "go.etcd.io/bbolt"
)

var bucketName = []byte("koris")

// BboltStore 把 Koris 的全部键持久化到一个有序 Bbolt bucket。
//
// Bbolt 返回的 key/value 只在事务期间有效，因此 Get 和 Scan 都会在只读事务关闭
// 前复制数据。这样 Iterator 不会长期占用 read transaction，也不会阻止数据库
// 回收旧页面。
type BboltStore struct{ db *bbolt.DB }

// OpenBbolt 打开或创建数据库，并确保 Koris bucket 已存在。
// mode 为 0 时使用 0600，避免索引内容默认暴露给其他系统用户；Timeout 防止在
// 文件已被另一进程占用时无限等待。
func OpenBbolt(path string, mode uint32) (*BboltStore, error) {
	db, err := bbolt.Open(path, fileMode(mode), &bbolt.Options{Timeout: time.Second})
	if err != nil {
		return nil, err
	}
	store := &BboltStore{db: db}
	if err := db.Update(func(tx *bbolt.Tx) error {
		_, createErr := tx.CreateBucketIfNotExists(bucketName)
		return createErr
	}); err != nil {
		_ = db.Close()
		return nil, err
	}
	return store, nil
}

func fileMode(mode uint32) os.FileMode {
	if mode == 0 {
		mode = 0o600
	}
	return os.FileMode(mode)
}

func (s *BboltStore) Get(key []byte) ([]byte, error) {
	if s == nil || s.db == nil {
		return nil, ErrClosed
	}
	var result []byte
	err := s.db.View(func(tx *bbolt.Tx) error {
		// Bucket.Get 返回 mmap 页面上的视图，必须在 View 返回前复制。
		value := tx.Bucket(bucketName).Get(key)
		if value == nil {
			return ErrNotFound
		}
		result = cloneBytes(value)
		return nil
	})
	return result, normalizeBboltError(err)
}

func (s *BboltStore) Put(key, value []byte) error {
	return s.Transaction(func(tx Tx) error { return tx.Put(key, value) })
}

func (s *BboltStore) Delete(key []byte) error {
	return s.Transaction(func(tx Tx) error { return tx.Delete(key) })
}

func (s *BboltStore) Scan(prefix []byte) Iterator {
	if s == nil || s.db == nil {
		return errorIterator(ErrClosed)
	}
	var entries []entry
	err := s.db.View(func(tx *bbolt.Tx) error {
		// 将结果做成独立快照后立即释放 Bbolt 只读事务。
		entries = scanBucket(tx.Bucket(bucketName), prefix)
		return nil
	})
	if err != nil {
		return errorIterator(normalizeBboltError(err))
	}
	return newSliceIterator(entries)
}

func (s *BboltStore) Transaction(fn func(Tx) error) error {
	if s == nil || s.db == nil {
		return ErrClosed
	}
	// Bbolt Update 对回调错误自动回滚，对 nil 自动提交，直接满足 Store 契约。
	return normalizeBboltError(s.db.Update(func(tx *bbolt.Tx) error {
		return fn(&bboltTx{bucket: tx.Bucket(bucketName)})
	}))
}

func (s *BboltStore) Close() error {
	if s == nil || s.db == nil {
		return nil
	}
	err := s.db.Close()
	s.db = nil
	return err
}

type bboltTx struct{ bucket *bbolt.Bucket }

func (tx *bboltTx) Get(key []byte) ([]byte, error) {
	value := tx.bucket.Get(key)
	if value == nil {
		return nil, ErrNotFound
	}
	return cloneBytes(value), nil
}

func (tx *bboltTx) Put(key, value []byte) error { return tx.bucket.Put(key, value) }
func (tx *bboltTx) Delete(key []byte) error     { return tx.bucket.Delete(key) }
func (tx *bboltTx) Scan(prefix []byte) Iterator {
	return newSliceIterator(scanBucket(tx.bucket, prefix))
}

func scanBucket(bucket *bbolt.Bucket, prefix []byte) []entry {
	entries := make([]entry, 0)
	cursor := bucket.Cursor()
	// Bbolt cursor 按字节序排列。Seek 直接定位到首个可能命中的 key，一旦前缀
	// 不再匹配即可停止，无需扫描整个 bucket。
	for key, value := cursor.Seek(prefix); key != nil && bytes.HasPrefix(key, prefix); key, value = cursor.Next() {
		entries = append(entries, entry{key: cloneBytes(key), value: cloneBytes(value)})
	}
	return entries
}

func normalizeBboltError(err error) error {
	// 对外统一使用 storage.ErrClosed，避免上层依赖具体后端错误类型。
	if errors.Is(err, bbolt.ErrDatabaseNotOpen) {
		return ErrClosed
	}
	return err
}
