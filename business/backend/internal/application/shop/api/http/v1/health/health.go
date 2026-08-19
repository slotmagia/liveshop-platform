package health

import "github.com/gogf/gf/v2/frame/g"

type GetReq struct {
	g.Meta `path:"/health" method:"get" tags:"Platform-Shop" summary:"店铺面存活检查"`
}

type GetRes struct {
	Status string `json:"status"`
}
