package base

import (
	"mj/gameServer/db"

	"github.com/lovelly/leaf/log"
)

//This file is generate by scripts,don't edit it

//game_service_option
//

// +gen
type GameServiceOption struct {
	KindID               int    `db:"KindID" json:"KindID"`                             // 名称号码
	NodeID               int    `db:"NodeID" json:"NodeID"`                             // 挂接节点
	SortID               int    `db:"SortID" json:"SortID"`                             // 排列标识
	ServerID             int    `db:"ServerID" json:"ServerID"`                         // 房间标识
	CellScore            int    `db:"CellScore" json:"CellScore"`                       // 单位积分
	AndroidMaxCellScore  int    `db:"AndroidMaxCellScore" json:"AndroidMaxCellScore"`   // 机器人最大进入底注
	RevenueRatio         int    `db:"RevenueRatio" json:"RevenueRatio"`                 // 税收比例
	ServiceScore         int    `db:"ServiceScore" json:"ServiceScore"`                 // 服务费用
	RestrictScore        int    `db:"RestrictScore" json:"RestrictScore"`               // 限制积分
	MinTableScore        int    `db:"MinTableScore" json:"MinTableScore"`               // 最低积分
	MinEnterScore        int    `db:"MinEnterScore" json:"MinEnterScore"`               // 最低积分
	MaxEnterScore        int    `db:"MaxEnterScore" json:"MaxEnterScore"`               // 最高积分
	MinEnterMember       int    `db:"MinEnterMember" json:"MinEnterMember"`             // 最低会员
	MaxEnterMember       int    `db:"MaxEnterMember" json:"MaxEnterMember"`             // 最高会员
	ServerRule           int    `db:"ServerRule" json:"ServerRule"`                     // 房间规则
	AttachUserRight      int    `db:"AttachUserRight" json:"AttachUserRight"`           // 附加权限
	MaxPlayer            int    `db:"MaxPlayer" json:"MaxPlayer"`                       // 最大数目
	TableCount           int    `db:"TableCount" json:"TableCount"`                     // 桌子数目
	ServerPort           int    `db:"ServerPort" json:"ServerPort"`                     // 服务端口
	ServerKind           int    `db:"ServerKind" json:"ServerKind"`                     // 房间类别
	ServerType           int    `db:"ServerType" json:"ServerType"`                     // 房间类型
	ServerLevel          int    `db:"ServerLevel" json:"ServerLevel"`                   // 房间等级
	ServerName           string `db:"ServerName" json:"ServerName"`                     // 房间名称
	ServerPasswd         string `db:"ServerPasswd" json:"ServerPasswd"`                 // 房间密码
	DistributeRule       int    `db:"DistributeRule" json:"DistributeRule"`             // 分组规则
	MinDistributeUser    int    `db:"MinDistributeUser" json:"MinDistributeUser"`       // 最少人数
	MaxDistributeUser    int    `db:"MaxDistributeUser" json:"MaxDistributeUser"`       // 最多人数
	DistributeTimeSpace  int    `db:"DistributeTimeSpace" json:"DistributeTimeSpace"`   // 分组间隔
	DistributeDrawCount  int    `db:"DistributeDrawCount" json:"DistributeDrawCount"`   // 分组局数
	DistributeStartDelay int    `db:"DistributeStartDelay" json:"DistributeStartDelay"` // 开始延时
	CustomRule           string `db:"CustomRule" json:"CustomRule"`                     // 自定规则
}

var DefaultGameServiceOption = GameServiceOption{}

type gameServiceOptionCache struct {
	objMap  map[int]*GameServiceOption
	objList []*GameServiceOption
}

var GameServiceOptionCache = &gameServiceOptionCache{}

func (c *gameServiceOptionCache) LoadAll() {
	sql := "select * from game_service_option"
	c.objList = make([]*GameServiceOption, 0)
	err := db.BaseDB.Select(&c.objList, sql)
	if err != nil {
		log.Fatal(err.Error())
	}
	c.objMap = make(map[int]*GameServiceOption)
	log.Debug("Load all game_service_option success: %d", len(c.objList))
	for _, v := range c.objList {
		c.objMap[v.KindID] = v
	}
}

func (c *gameServiceOptionCache) All() []*GameServiceOption {
	return c.objList
}

func (c *gameServiceOptionCache) Count() int {
	return len(c.objList)
}

func (c *gameServiceOptionCache) Get(KindID int) (*GameServiceOption, bool) {
	key := KindID
	v, ok := c.objMap[key]
	return v, ok
}

// 仅限运营后台实时刷新服务器数据用
func (c *gameServiceOptionCache) Update(v *GameServiceOption) {
	key := v.KindID
	c.objMap[key] = v
}
