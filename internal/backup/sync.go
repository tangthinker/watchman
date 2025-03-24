package backup

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// FileInfo 存储文件信息
type FileInfo struct {
	Path    string
	Size    int64
	Hash    string
	ModTime int64
	IsDir   bool
}

// calculateHash 计算文件的SHA256哈希值
func calculateHash(path string) (string, error) {
	file, err := os.Open(path)
	if err != nil {
		return "", err
	}
	defer file.Close()

	hash := sha256.New()
	if _, err := io.Copy(hash, file); err != nil {
		return "", err
	}

	return hex.EncodeToString(hash.Sum(nil)), nil
}

// getFileInfo 获取文件信息
func getFileInfo(path string) (*FileInfo, error) {
	info, err := os.Stat(path)
	if err != nil {
		return nil, err
	}

	fileInfo := &FileInfo{
		Path:    path,
		Size:    info.Size(),
		ModTime: info.ModTime().Unix(),
		IsDir:   info.IsDir(),
	}

	if !info.IsDir() {
		hash, err := calculateHash(path)
		if err != nil {
			return nil, err
		}
		fileInfo.Hash = hash
	}

	return fileInfo, nil
}

// scanDirectory 扫描目录下的所有文件
func scanDirectory(dir string) (map[string]*FileInfo, error) {
	files := make(map[string]*FileInfo)

	err := filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		// 跳过.目录
		if strings.HasPrefix(info.Name(), ".") {
			if info.IsDir() {
				return filepath.SkipDir
			}
			return nil
		}

		fileInfo, err := getFileInfo(path)
		if err != nil {
			return err
		}

		// 使用相对路径作为键
		relPath, err := filepath.Rel(dir, path)
		if err != nil {
			return err
		}
		files[relPath] = fileInfo

		return nil
	})

	return files, err
}

// Sync 执行增量同步
func Sync(sourcePath, targetPath string, progressChan chan<- float64) error {
	// 确保目标目录存在
	if err := os.MkdirAll(targetPath, 0755); err != nil {
		return fmt.Errorf("failed to create target directory: %v", err)
	}

	// 扫描源目录和目标目录
	sourceFiles, err := scanDirectory(sourcePath)
	if err != nil {
		return fmt.Errorf("failed to scan source directory: %v", err)
	}

	targetFiles, err := scanDirectory(targetPath)
	if err != nil {
		return fmt.Errorf("failed to scan target directory: %v", err)
	}

	totalFiles := len(sourceFiles)
	if totalFiles == 0 {
		if progressChan != nil {
			progressChan <- 100
		}
		return nil
	}

	processedFiles := 0
	filesToSync := 0

	// 计算需要同步的文件数量
	for relPath, sourceFile := range sourceFiles {
		targetFile, exists := targetFiles[relPath]
		if !exists || sourceFile.Hash != targetFile.Hash {
			filesToSync++
		}
	}

	// 如果没有文件需要同步，直接返回100%进度
	if filesToSync == 0 {
		if progressChan != nil {
			progressChan <- 100
		}
		return nil
	}

	// 同步文件
	for relPath, sourceFile := range sourceFiles {
		targetFile, exists := targetFiles[relPath]
		targetFilePath := filepath.Join(targetPath, relPath)

		// 如果目标文件不存在或哈希值不同，则复制
		if !exists || sourceFile.Hash != targetFile.Hash {
			if sourceFile.IsDir {
				if err := os.MkdirAll(targetFilePath, 0755); err != nil {
					return fmt.Errorf("failed to create directory %s: %v", targetFilePath, err)
				}
			} else {
				// 确保目标文件的目录存在
				if err := os.MkdirAll(filepath.Dir(targetFilePath), 0755); err != nil {
					return fmt.Errorf("failed to create directory for %s: %v", targetFilePath, err)
				}

				// 复制文件
				if err := copyFile(
					filepath.Join(sourcePath, relPath),
					targetFilePath,
					sourceFile.ModTime,
				); err != nil {
					return fmt.Errorf("failed to copy file %s: %v", relPath, err)
				}
			}
			processedFiles++
			if progressChan != nil {
				progress := float64(processedFiles) / float64(filesToSync) * 100
				progressChan <- progress
			}
		}
	}

	// 删除目标目录中不存在的文件
	for relPath := range targetFiles {
		if _, exists := sourceFiles[relPath]; !exists {
			targetFilePath := filepath.Join(targetPath, relPath)
			if err := os.RemoveAll(targetFilePath); err != nil {
				return fmt.Errorf("failed to remove %s: %v", targetFilePath, err)
			}
		}
	}

	// 确保最后发送100%进度
	if progressChan != nil {
		progressChan <- 100
	}

	return nil
}

// copyFile 复制文件并保持修改时间
func copyFile(src, dst string, modTime int64) error {
	source, err := os.Open(src)
	if err != nil {
		return err
	}
	defer source.Close()

	destination, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer destination.Close()

	if _, err := io.Copy(destination, source); err != nil {
		return err
	}

	modTimeObj := time.Unix(modTime, 0)
	return os.Chtimes(dst, modTimeObj, modTimeObj)
}
