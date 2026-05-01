package usecase

// ModerationUseCase 内容审核接口
type ModerationUseCase interface {
	// DetectSensitiveWords 检测文本中是否包含敏感词
	DetectSensitiveWords(text string) bool

	// DetectSensitiveImage 检测图片中是否包含敏感内容
	// 参数为图片URL，如果不支持图片检测，应该返回错误
	DetectSensitiveImage(imageURL string) (bool, error)
}

