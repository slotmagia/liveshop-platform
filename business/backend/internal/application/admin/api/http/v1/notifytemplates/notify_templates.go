package notifytemplates

import (
	"time"

	"github.com/gogf/gf/v2/frame/g"
)

type Template struct {
	Code         string    `json:"code"`
	Channel      string    `json:"channel"`
	TextTemplate string    `json:"textTemplate,omitempty"`
	Subject      string    `json:"subject,omitempty"`
	BodyHTML     string    `json:"bodyHtml,omitempty"`
	Title        string    `json:"title,omitempty"`
	Body         string    `json:"body,omitempty"`
	Variables    []string  `json:"variables"`
	Lifecycle    string    `json:"lifecycle"`
	Version      int64     `json:"version"`
	UpdatedAt    time.Time `json:"updatedAt"`
}

type ListReq struct {
	g.Meta  `path:"/notify-templates" method:"get" tags:"Platform-通知模板" summary:"查询可复用通知模板"`
	Channel string `json:"channel" in:"query"`
	Keyword string `json:"keyword" in:"query"`
}
type ListRes []Template

type GetReq struct {
	g.Meta `path:"/notify-templates/{code}" method:"get" tags:"Platform-通知模板" summary:"读取一份通知模板"`
	Code   string `json:"code" in:"path"`
}
type GetRes Template

type UpdateReq struct {
	g.Meta          `path:"/notify-templates/{code}" method:"put" tags:"Platform-通知模板" summary:"版本化保存通知模板"`
	Code            string   `json:"code" in:"path"`
	CommandKey      string   `json:"commandKey"`
	ExpectedVersion int64    `json:"expectedVersion"`
	Channel         string   `json:"channel"`
	TextTemplate    string   `json:"textTemplate"`
	Subject         string   `json:"subject"`
	BodyHTML        string   `json:"bodyHtml"`
	Title           string   `json:"title"`
	Body            string   `json:"body"`
	Variables       []string `json:"variables"`
}
type UpdateRes Template

type RetireReq struct {
	g.Meta          `path:"/notify-templates/{code}/retire" method:"post" tags:"Platform-通知模板" summary:"退役通知模板"`
	Code            string `json:"code" in:"path"`
	CommandKey      string `json:"commandKey"`
	ExpectedVersion int64  `json:"expectedVersion"`
}
type RetireRes Template
