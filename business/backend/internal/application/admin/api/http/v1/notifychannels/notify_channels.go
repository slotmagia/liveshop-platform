package notifychannels

import (
	"time"

	"github.com/gogf/gf/v2/frame/g"
)

type InAppConfig struct {
	Driver    string    `json:"driver"`
	Enabled   bool      `json:"enabled"`
	Version   int64     `json:"version"`
	UpdatedAt time.Time `json:"updatedAt"`
}

type GetInAppReq struct {
	g.Meta `path:"/notify-channels/in-app" method:"get" tags:"Platform-通知方式" summary:"读取站内信驱动配置"`
}
type GetInAppRes InAppConfig

type UpdateInAppReq struct {
	g.Meta          `path:"/notify-channels/in-app" method:"put" tags:"Platform-通知方式" summary:"版本化保存站内信驱动配置"`
	CommandKey      string `json:"commandKey"`
	ExpectedVersion int64  `json:"expectedVersion"`
	Enabled         bool   `json:"enabled"`
}
type UpdateInAppRes InAppConfig
