// Package manager
// @Author Clover
// @Data 2024/8/10 下午11:27:00
// @Desc 聊天图片管理类（文件形式存储）（数据库方式参考：bboltManager.go）
package manager

import (
	"errors"
	"fmt"
	"github.com/rs/zerolog/log"
	"os"
	"strings"
	"wechat-demo/rikkabot/config"
	"wechat-demo/rikkabot/utils/timeutil"
)

var (
	imgDirpath string // 图片保存路径
	// 错误类型
	errInvalidDateString = errors.New("invalid date string")
	errUnFindDirByDate   = errors.New("un find dir by date")
)

type dateDirEntry map[string]os.DirEntry // 日期文件夹-文件夹实体映射
type imgid2PathEntry map[string]string   // 图片id-图片路径实体映射

// 图片文件缓存
type tFileImgCache struct {
	dateDirEntry
	imgid2PathEntry
}

var fileImgCache tFileImgCache

func init() {
	// 初始化图片文件缓存
	fileImgCache = tFileImgCache{
		make(map[string]os.DirEntry),
		make(map[string]string),
	}
	// 初始化目录 && 校验目录是否存在
	cfg := config.GetConfig()
	var err error
	imgDirpath = cfg.ImgDirPath
	_, err = ValidPath(imgDirpath, true)
	if err != nil {
		log.Fatal().Err(err).Str("path", imgDirpath).Msg("validate img_dir path")
	}
}

// 文件形式保存图片
func (i imgCache) saveImgAsFile(imgId string, imgData []byte, imgDate string) (err error) {
	dpath, err := fileImgCache.findImgDateDirPath(imgDate, true) // 找到日期目录
	err = handlefindDirErr(err)
	if err != nil {
		return err
	}
	path := dpath + "/" + imgId
	err = fileImgCache.saveImgByData(path, imgData)
	if err != nil {
		return err
	}
	return nil
}

// 文件方式获取图片
func (i imgCache) getImgByfile(imgId string, imgDate string) (imgdata []byte, err error) {
	return fileImgCache.findImgData(imgId, imgDate)
}

// 单次检查日期文件夹是否过期
// nolint
func (i imgCache) checkByFile(err error) {
	// 读取日期缓存中的所有日期目录
	for date, _ := range fileImgCache.dateDirEntry {
		// 判断日期是否过期
		if timeutil.IsBeforeThatDay(date, i.ImgValidDuration) {
			// 过期删除文件夹和缓存
			delete(fileImgCache.dateDirEntry, date)
			// 清除缓存
			fileImgCache.clearImgPathCache(date)
			// 删除过期图片（包括文件夹）
			err = os.RemoveAll(imgDirpath + "/" + date)
			if err != nil {
				err = fmt.Errorf("delete img dir err: %w", err)
				if err != nil {
					log.Error().Err(err).Str("dir", imgDirpath+"/"+date).Msg("remove img_date_dir err")
				}
			}
		}
	}
}

// 清除过期文件夹的图片路径缓存
func (fic *tFileImgCache) clearImgPathCache(imgDate string) {
	for imgid, imgPath := range fic.imgid2PathEntry {
		if strings.HasPrefix(imgPath, imgDirpath+"/"+imgDate) {
			delete(fic.imgid2PathEntry, imgid)
		}
	}
}

// 找出当前请求图片对应目录
func (fic *tFileImgCache) findImgDateDirPath(imgDate string, isCreate bool) (dirpath string, err error) {
	// 不满足日期格式
	if !timeutil.IsDateValid(imgDate) {
		return "", errInvalidDateString
	}
	// 先查询文件缓存是否存在，如不存在再遍历
	if fic.dateDirEntry[imgDate] != nil {
		return imgDirpath + "/" + fic.dateDirEntry[imgDate].Name(), nil
	}
	// 遍历所有目录，满足目录
	imgDir, err := os.ReadDir(imgDirpath)
	if err != nil {
		return "", fmt.Errorf("read imgDir in img_path err: %w", err)
	}
	for _, d := range imgDir {
		if d.IsDir() {
			if timeutil.IsDateValid(d.Name()) { // 满足日期格式，缓存该目录
				fic.dateDirEntry[d.Name()] = d
				if d.Name() == imgDate {
					dirpath = imgDirpath + "/" + imgDate
				}
			}
		}
	}
	if dirpath == "" {
		if !isCreate {
			return "", errUnFindDirByDate
		}
		// 创建模式，新建文件夹
		err = os.MkdirAll(imgDirpath+"/"+imgDate, os.ModePerm)
		if err != nil {
			return "", fmt.Errorf("create img_date_dir err: %w", err)
		}
		return imgDirpath + "/" + imgDate, nil
	}
	return
}

// 找到图片名对应文件数据
func (fic *tFileImgCache) findImgData(imgId string, imgDate string) (data []byte, err error) {
	dirpath, err := fic.findImgDateDirPath(imgDate, false)
	err = handlefindDirErr(err)
	if err != nil {
		return nil, err
	}
	// 命中缓存直接返回
	ipath, ok := fic.imgid2PathEntry[imgId]
	if ok {
		// 检查是否符合日期目录
		ok := strings.HasPrefix(ipath, dirpath)
		if ok {
			return fic.readImgByPath(ipath)
		}
	}
	// 遍历目录
	imgs, err := os.ReadDir(dirpath)
	if err != nil {
		return nil, fmt.Errorf("read img_dir err: %w", err)
	}
	for _, d := range imgs {
		if !d.IsDir() { // 是图片文件
			// 缓存图片路径 id-path
			fic.imgid2PathEntry[d.Name()] = dirpath + "/" + d.Name()
			if d.Name() == imgId {
				data, err = fic.readImgByPath(dirpath + "/" + d.Name())
			}
		}
	}
	if err != nil {
		return nil, fmt.Errorf("read imgData in img_dir err: %w", err)
	}
	return
}

// 保存图片为文件
func (fic *tFileImgCache) saveImgByData(path string, imgData []byte) (err error) {
	file, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, os.ModePerm)
	if err != nil {
		return fmt.Errorf("save img as file err: %w", err)
	}
	defer file.Close()
	_, err = file.Write(imgData)
	if err != nil {
		return fmt.Errorf("save img as file err: %w", err)
	}
	return nil
}

func (fic *tFileImgCache) readImgByPath(path string) (data []byte, err error) {
	data, err = os.ReadFile(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			delete(fic.imgid2PathEntry, path) // 清除缓存
		}
		return nil, fmt.Errorf("read img by path err: %w", err)
	}
	return
}

func handlefindDirErr(err error) error {
	if err != nil {
		switch {
		case errors.Is(err, errInvalidDateString):
			log.Error().Err(err).Msg("不是有效的图片目录")
			return err
		case errors.Is(err, errUnFindDirByDate):
			log.Error().Err(err).Msg("找不到该图片目录（过期或不曾存在）")
			return err
		}
	}
	return nil
}
