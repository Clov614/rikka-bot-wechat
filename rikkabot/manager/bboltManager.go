// Package manager
// @Author Clover
// @Data 2024/7/28 下午10:37:00
// @Desc etcd/bbolt 嵌入式 键值存储 用户持久化缓存，以及支持一些缓存的备份操作
package manager

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"github.com/Clov614/logging"
	"github.com/Clov614/rikka-bot-wechat/rikkabot/config"
	"github.com/rs/zerolog/log"
	"go.etcd.io/bbolt"
)

var (
	mDBPath string // db文件完整路径

	db          *bbolt.DB
	oncedbclose sync.Once
)

var (
	ErrDefaultDB          = errors.New("default db error")
	ErrGetImgBucketByDate = errors.New("get image bucket by date error")
	ErrGetImgNil          = errors.New("get image nil error in bolt")
)

const (
	defaultDBName = "/rikka.db"
)

// nolint
func init() {
	// 初始化bbolt桶
	cfg := config.GetConfig()
	mDBPath = cfg.DBDirPath + defaultDBName
	var err error
	_, err = ValidPath(cfg.DBDirPath, true)
	if err != nil {
		log.Fatal().Err(err).Str("path", mDBPath).Msg("validate db path")
	}
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
	if err != nil {
		return nil, fmt.Errorf("load cache: %w", err)
	}
	logging.Debug("load cache in rikka.db", map[string]interface{}{"cache": cache})
	return cache, nil
}

// CloseDB 关闭数据库
func CloseDB() {
	oncedbclose.Do(func() {
		db.Close()
	})
}

func ValidPath(path string, isCreate bool) (bool, error) {
	// 使用 filepath.Split 来逐层分离目录
	directories := strings.Split(filepath.Clean(path), string(os.PathSeparator))

	currentPath := ""
	for _, dir := range directories {
		if dir == "" {
			continue
		}
		currentPath = filepath.Join(currentPath, dir)

		if _, err := os.Stat(currentPath); os.IsNotExist(err) {
			if isCreate {
				if err := os.Mkdir(currentPath, os.ModePerm); err != nil {
					return false, fmt.Errorf("error creating directory %s: %w", currentPath, err)
				}
			} else {
				return false, fmt.Errorf("directory does not exist: %s: %w", currentPath)
			}
		} else if err != nil {
			return false, fmt.Errorf("error accessing directory %s: %w", currentPath, err)
		}
	}

	return true, nil
}

//// DbExists 判断db文件是否已经创建
//func DbExists() bool {
//	info, err := os.Stat(mDBPath)
//	if err != nil {
//		if os.IsNotExist(err) {
//			return false
//		}
//	}
//	return !info.IsDir()
//}

// GetDB 返回全局的 bbolt.DB 实例
// 这个函数允许其他包访问由 bboltManager 初始化和管理的数据库连接。
// 注意：直接访问全局数据库实例应谨慎处理，确保并发安全和资源管理得当。
func GetDB() *bbolt.DB {
	return db
}
