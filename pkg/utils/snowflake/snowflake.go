package snowflake

import (
	"fmt"
	"sync"
	"time"

	"github.com/kirklin/snowflake"
)

var (
	node *snowflake.Node
	once sync.Once
)

// Config holds the configuration for the Snowflake node generator.
type Config struct {
	Epoch       string
	MachineBits int
	StepBits    int
}

// InitNode configures and initializes the global snowflake node.
func InitNode(config *Config) error {
	var initErr error
	once.Do(func() {
		// Default values to prevent unexpected behavior
		mb := uint8(10)
		sb := uint8(12)
		if config.MachineBits > 0 {
			mb = uint8(config.MachineBits)
		}
		if config.StepBits > 0 {
			sb = uint8(config.StepBits)
		}

		st := snowflake.Settings{
			MachineBits: mb,
			StepBits:    sb,
			// 我们需要 1ms 级别的雪花 ID
			TimeUnit: 1 * time.Millisecond,
		}

		if config.Epoch != "" {
			t, err := time.Parse(time.RFC3339, config.Epoch)
			if err != nil {
				initErr = fmt.Errorf("failed to parse snowflake epoch: %w", err)
				return
			}
			st.StartTime = t
		} else {
			st.StartTime = time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
		}

		var err error
		node, err = snowflake.NewNodeWithSettings(st)
		if err != nil {
			initErr = fmt.Errorf("failed to initialize snowflake node: %w", err)
		}
	})
	return initErr
}

// NextID 生成一个全局唯一的雪花 ID
func NextID() int64 {
	if node == nil {
		panic("snowflake node not initialized")
	}
	id, err := node.NextID()
	if err != nil {
		panic("failed to generate snowflake id: " + err.Error())
	}
	return id.Int64()
}
