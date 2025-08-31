package moderation

import (
	"sync"

	"github.com/kirklin/boot-backend-go-clean/internal/domain/usecase"
	"github.com/kirklin/boot-backend-go-clean/pkg/logger"
	"github.com/kirklin/go-swd/pkg/core"
	"github.com/kirklin/go-swd/pkg/swd"
)

// LocalModerationDetector 本地内容审核实现
type LocalModerationDetector struct {
	detector core.Detector
	once     sync.Once
}

// NewLocalModerationDetector 创建本地内容审核实例
func NewLocalModerationDetector() usecase.ModerationUseCase {
	detector := &LocalModerationDetector{}
	detector.init()
	return detector
}

// init 初始化内容审核器
func (d *LocalModerationDetector) init() {
	d.once.Do(func() {
		factory := swd.NewDefaultFactory()
		var err error
		d.detector, err = swd.New(factory)
		if err != nil {
			logger.GetLogger().Errorf("Failed to initialize content moderator: %v", err)
			return
		}
	})
}

// DetectSensitiveWords 检测文本中是否包含敏感词
func (d *LocalModerationDetector) DetectSensitiveWords(text string) bool {
	if d.detector == nil {
		return false
	}
	return d.detector.Detect(text)
}

// DetectSensitiveImage 检测图片中是否包含敏感内容
// 本地内容审核器不支持图片检测，返回错误
func (d *LocalModerationDetector) DetectSensitiveImage(imageURL string) (bool, error) {
	// 本地内容审核器不支持图片检测，返回错误
	return false, usecase.ErrLocalImageModeration
}

// GetSensitiveWords 获取文本中的所有敏感词
func (d *LocalModerationDetector) GetSensitiveWords(text string) []string {
	if d.detector == nil {
		return nil
	}

	matches := d.detector.MatchAll(text)
	words := make([]string, 0, len(matches))
	for _, match := range matches {
		words = append(words, match.Word)
	}
	return words
}
