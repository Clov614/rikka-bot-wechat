// Package manager
// @Author Clover
// @Data 2024/7/28 下午10:37:00
// @Desc etcd/bbolt 嵌入式 键值存储 用户持久化缓存，以及支持一些缓存的备份操作
package manager

import (
	"encoding/json"
	"errors"
	"fmt"
	"github.com/rs/zerolog/log"
	"go.etcd.io/bbolt"
	"os"
	"path/filepath"
	"sync"
	"wechat-demo/rikkabot/config"
)

var (
	mDBPath string

	db          *bbolt.DB
	oncedbclose sync.Once
)

var (
	ErrDefaultDB = errors.New("default db error")
)

const (
	defaultDBName = "/rikka.db"
)

func init() {
	// 初始化bbolt桶
	cfg := config.GetConfig()
	mDBPath = cfg.DBDirPath + defaultDBName
	var err error
	_, err = validPath(mDBPath, true)
	db, err = bbolt.Open(mDBPath, 0600, nil)
	if err != nil {
		log.Fatal().Err(fmt.Errorf("err: %w detail: %w", ErrDefaultDB, err)).Msg("cannot open db")
	}
}

// SaveCache 保存 cache
func SaveCache(cache any) error {
	var err error
	err = db.Update(func(tx *bbolt.Tx) error {
		b, err := tx.CreateBucketIfNotExists([]byte("cache"))
		if err != nil {
			return fmt.Errorf("create bucket: %w", err)
		}
		marshal, err := json.Marshal(cache)
		if err != nil {
			return fmt.Errorf("marshal cache: %w", err)
		}
		err = b.Put([]byte("cache"), marshal)
		if err != nil {
			return fmt.Errorf("put cache: %w", err)
		}
		return nil
	})
	if err != nil {
		return fmt.Errorf("save cache: %w", err)
	}
	return nil
}

// LoadCache 读取cache
func LoadCache(cache any) (any, error) {
	var err error
	err = db.View(func(tx *bbolt.Tx) error {
		b := tx.Bucket([]byte("cache"))
		if b == nil {
			return nil
		}
		bytes := b.Get([]byte("cache"))
		err = json.Unmarshal(bytes, cache)
		if err != nil {
			return fmt.Errorf("unmarshal cache: %w", err)
		}
		return nil
	})
	return cache, err
}

// CloseDB 关闭数据库
func CloseDB() {
	oncedbclose.Do(func() {
		db.Close()
	})
}

func validPath(path string, isCreate bool) (bool, error) {
	dir := filepath.Dir(path)
	if _, err := os.Stat(dir); os.IsNotExist(err) {
		if isCreate {
			if err := os.MkdirAll(dir, os.ModePerm); err != nil {
				return false, fmt.Errorf("error creating ./data/db directory: %w", err)
			}
		}
	}
	return true, nil
}
