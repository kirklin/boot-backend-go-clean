package snowflake

import (
	"sync"
	"time"

	"github.com/kirklin/snowflake"
)

var (
	node *snowflake.Node
	once sync.Once
)

// init 初始化雪花 ID 节点 (自定义配置)
func init() {
	once.Do(func() {
		st := snowflake.Settings{
			// 自定义纪元，延长可用年限
			StartTime:   time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC),
			MachineBits: 10,
			StepBits:    12,
			// 我们需要 1ms 级别的雪花 ID
			TimeUnit: 1 * time.Millisecond,
		}

		var err error
		node, err = snowflake.NewNodeWithSettings(st)
		if err != nil {
			panic("failed to init snowflake node: " + err.Error())
		}
	})
}

// NextID 生成一个全局唯一的雪花 ID
func NextID() int64 {
	id, err := node.NextID()
	if err != nil {
		panic("failed to generate snowflake id: " + err.Error())
	}
	return id.Int64()
}
