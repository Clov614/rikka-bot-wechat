// Package manager
// @Author Clover
// @Data 2024/7/28 下午10:37:00
// @Desc etcd/bbolt 嵌入式 键值存储 用户持久化缓存，以及支持一些缓存的备份操作
package manager

import (
	"encoding/json"
	"errors"
	"fmt"
	"github.com/Clov614/rikka-bot-wechat/rikkabot/config"
	"github.com/Clov614/rikka-bot-wechat/rikkabot/logging"
	"github.com/Clov614/rikka-bot-wechat/rikkabot/utils/imgutil"
	"github.com/Clov614/rikka-bot-wechat/rikkabot/utils/timeutil"
	"github.com/rs/zerolog/log"
	"go.etcd.io/bbolt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"time"
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

	imgBucketName = "chat_image"
)

var ic *imgCache

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
	// 初始化聊天图片桶
	err = db.Update(func(tx *bbolt.Tx) error {
		_, e := tx.CreateBucketIfNotExists([]byte(imgBucketName))
		if e != nil {
			return fmt.Errorf("创建聊天图片桶失败: %w", e)
		}
		return nil
	})
	if err != nil {
		log.Fatal().Err(err).Msg("cannot init db")
	}
	// 初始化图片缓存
	var isCacheByFile bool
	if cfg.ImgSaveType == "file" {
		isCacheByFile = true
	}
	ic = &imgCache{
		ImgValidDuration: cfg.ImgValidDuration,      // 图片有效期
		CheckInterval:    cfg.ImgCacheCheckInterval, // 检查是否过期间隔
		IsCacheByFile:    isCacheByFile,             // 是否文件方式存储图片
	}
	// 循环检测图片是否过期
	go ic.cycleCheckOutDate()
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

// SaveImg uuid: 可置空
func SaveImg(uuid string, imgData []byte) (imgName string, imgDate string) {
	//if uuid == common.UUID_NOT_UNIQUE_INGROUPS || uuid == common.UUID_NOT_UNIQUE_INFRIENDS {
	//	uuid = "" // 重复 uuid 置空
	//}
	var err error
	imgId := timeutil.GetTimeStamp() + "_" + uuid
	nowDate := timeutil.GetNowDate()
	// 获取img后缀
	fileType, err := imgutil.DetectFileType(imgData)
	if err != nil {
		log.Warn().Err(err).Msg("detect file type of image err")
	}
	imgId = concatExt(imgId, string(fileType)) // 拼接后缀
	if ic.IsCacheByFile {
		err = ic.saveImgAsFile(imgId, imgData, nowDate)
	} else {
		err = ic.saveImg(imgId, imgData, nowDate)
	}
	if err != nil {
		log.Error().Err(err).Msg(fmt.Sprintf("save img %s err", imgId))
	}
	return imgId, nowDate
}

// GetImg 获取图片 imgId: 由时间戳和uuid拼接 imgDate: 图片创建的日期
func GetImg(imgId string, imgDate string) (imgData []byte) {
	var err error
	if ic.IsCacheByFile {
		imgData, err = ic.getImgByfile(imgId, imgDate)
	} else {
		imgData, err = ic.getImg(imgId, imgDate)
	}
	if err != nil {
		log.Error().Err(err).Msg(fmt.Sprintf("get img %s err", imgId))
	}
	return imgData
}

// imgCache 图片存储相关
type imgCache struct {
	ImgValidDuration int  // 有效日期单位为天
	CheckInterval    int  // 单位为小时
	IsCacheByFile    bool // 是否文件方式缓存
}

// saveImg imgId: 外部通过时间戳+uuid拼接而成  nowDate: 当天日期，外部传递
func (i imgCache) saveImg(imgId string, imgData []byte, nowDate string) error {
	var err error
	err = db.Update(func(tx *bbolt.Tx) error {
		b := tx.Bucket([]byte(imgBucketName))
		// 根据当天日期判断桶是否存在，不存在则创建
		_, e3 := b.CreateBucketIfNotExists([]byte(nowDate))
		if e3 != nil {
			return fmt.Errorf("create bucket nowDate: %w", e3)
		}
		// 获取日期桶（日期桶分类图片，便于循环检查是否过期）
		dateBucket := b.Bucket([]byte(nowDate))
		// 存入日期桶
		e2 := dateBucket.Put([]byte(concatImgId(nowDate, imgId)), imgData)
		if e2 != nil {
			return fmt.Errorf("save img: %w", e2)
		}
		return nil
	})
	if err != nil {
		return fmt.Errorf("save img %s err: %w", imgId, err)
	}
	return nil
}

// getImg 获取图片
func (i imgCache) getImg(imgId string, imgDate string) ([]byte, error) {
	var err error
	var imgData []byte
	err = db.View(func(tx *bbolt.Tx) error {
		b := tx.Bucket([]byte(imgBucketName))
		// 获取日期对应桶
		dateBucket := b.Bucket([]byte(imgDate))
		if dateBucket == nil {
			return fmt.Errorf("获取该日期%s桶失败: %w", imgDate, ErrGetImgBucketByDate)
		}
		data := dateBucket.Get([]byte(concatImgId(imgDate, imgId)))
		if data == nil {
			return fmt.Errorf("获取%s的图片为空: %w", imgDate, ErrGetImgNil)
		}
		imgData = make([]byte, len(data))
		copy(imgData, data)
		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("get img %s err: %w", imgId, err)
	}
	return imgData, nil
}

// cycleCheckOutDate 循环检查图片桶是否过期
func (i imgCache) cycleCheckOutDate() {
	for {
		logging.Warn("循环校验图片是否过期，间隔: " + strconv.Itoa(i.CheckInterval) + " Hour")
		var err error
		if i.IsCacheByFile {
			i.checkByFile(err)
		} else {
			i.checkByDB(err)
		}
		time.Sleep(time.Duration(i.CheckInterval) * time.Hour)
	}
}

// nolint
func (i imgCache) checkByDB(err error) {
	err = db.Update(func(tx *bbolt.Tx) error {
		b := tx.Bucket([]byte(imgBucketName))
		e := b.ForEachBucket(func(k []byte) error {
			bucketDate := string(k)
			if timeutil.IsBeforeThatDay(bucketDate, i.ImgValidDuration) {
				// 过期删除该桶
				e2 := b.DeleteBucket(k)
				if e2 != nil {
					return fmt.Errorf("delete bucket: %w", e2)
				}
			}
			return nil
		})
		if e != nil {
			return fmt.Errorf("cycleCheckOutDate: %w", e)
		}
		return nil
	})
	if err != nil {
		log.Error().Err(err).Msg("cycleCheckOutDate")
	}
}

func concatImgId(nowDate string, imgId string) string {
	return nowDate + "_" + imgId
}

func concatExt(key string, etx string) string {
	if etx == "" {
		return key
	}
	return key + "." + etx
}

// 定时清理图床桶

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
				return false, fmt.Errorf("directory does not exist: %s: %w", currentPath, errUnFindDirByDate)
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
