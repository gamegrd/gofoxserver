package base

import (
	"mj/hallServer/db"

	"github.com/lovelly/leaf/log"
)

//This file is generate by scripts,don't edit it

//game_service_option
//

// +gen
type GameServiceOption struct {
	KindID                      int    `db:"KindID" json:"KindID"`                                           // 名称号码
	NodeID                      int    `db:"NodeID" json:"NodeID"`                                           //
	SortID                      int    `db:"SortID" json:"SortID"`                                           // 排列标识
	ServerID                    int    `db:"ServerID" json:"ServerID"`                                       // 房间标识
	CellScore                   int    `db:"CellScore" json:"CellScore"`                                     // 单位积分
	AndroidMaxCellScore         int    `db:"AndroidMaxCellScore" json:"AndroidMaxCellScore"`                 // 机器人最大进入底注
	RevenueRatio                int    `db:"RevenueRatio" json:"RevenueRatio"`                               // 税收比例
	ServiceScore                int    `db:"ServiceScore" json:"ServiceScore"`                               // 服务费用
	RestrictScore               int    `db:"RestrictScore" json:"RestrictScore"`                             // 限制积分
	MinTableScore               int    `db:"MinTableScore" json:"MinTableScore"`                             // 最低积分
	MinEnterScore               int    `db:"MinEnterScore" json:"MinEnterScore"`                             // 最低积分
	MaxEnterScore               int    `db:"MaxEnterScore" json:"MaxEnterScore"`                             // 最高积分
	ServerRule                  int    `db:"ServerRule" json:"ServerRule"`                                   // 房间规则
	MaxPlayer                   int    `db:"MaxPlayer" json:"MaxPlayer"`                                     // 最大数目
	ServerType                  int    `db:"ServerType" json:"ServerType"`                                   // 房间类型
	ServerName                  string `db:"ServerName" json:"ServerName"`                                   // 房间名称
	CbOffLineTrustee            int    `db:"cbOffLineTrustee" json:"cbOffLineTrustee"`                       // 是否短线代打
	CardOrBean                  int8   `db:"CardOrBean" json:"CardOrBean"`                                   // 消耗房卡还是游戏豆
	FeeBeanOrRoomCard           int    `db:"FeeBeanOrRoomCard" json:"FeeBeanOrRoomCard"`                     // 消耗房卡或游戏豆的数量
	PersonalRoomTax             int    `db:"PersonalRoomTax" json:"PersonalRoomTax"`                         // 私人房税收
	MaxCellScore                int    `db:"MaxCellScore" json:"MaxCellScore"`                               // 房间最大底分
	PlayTurnCount               int    `db:"PlayTurnCount" json:"PlayTurnCount"`                             // 房间能够进行游戏的最大局数
	PlayTimeLimit               int    `db:"PlayTimeLimit" json:"PlayTimeLimit"`                             // 房间能够进行游戏的最大时间
	TimeAfterBeginCount         int    `db:"TimeAfterBeginCount" json:"TimeAfterBeginCount"`                 // 一局游戏开始后多长时间后解散桌子
	TimeOffLineCount            int    `db:"TimeOffLineCount" json:"TimeOffLineCount"`                       // 玩家掉线多长时间后解散桌子
	TimeNotBeginGame            int    `db:"TimeNotBeginGame" json:"TimeNotBeginGame"`                       // 多长时间未开始游戏解散桌子	 单位秒
	TimeNotBeginAfterCreateRoom int    `db:"TimeNotBeginAfterCreateRoom" json:"TimeNotBeginAfterCreateRoom"` // 私人房创建多长时间后无人坐桌解散桌子
	DynamicJoin                 int    `db:"DynamicJoin" json:"DynamicJoin"`                                 // 是够允许游戏开始后加入 1是允许
	OutCardTime                 int    `db:"OutCardTime" json:"OutCardTime"`                                 // 多久没出牌自动出牌
	OperateCardTime             int    `db:"OperateCardTime" json:"OperateCardTime"`                         // 操作最大时间
}

var DefaultGameServiceOption = GameServiceOption{}

type gameServiceOptionCache struct {
	objMap  map[int]map[int]*GameServiceOption
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
	c.objMap = make(map[int]map[int]*GameServiceOption)
	log.Debug("Load all game_service_option success %v", len(c.objList))
	for _, v := range c.objList {
		obj, ok := c.objMap[v.KindID]
		if !ok {
			obj = make(map[int]*GameServiceOption)
			c.objMap[v.KindID] = obj
		}
		obj[v.ServerID] = v

	}
}

func (c *gameServiceOptionCache) All() []*GameServiceOption {
	return c.objList
}

func (c *gameServiceOptionCache) Count() int {
	return len(c.objList)
}

func (c *gameServiceOptionCache) Get(KindID int, ServerID int) (*GameServiceOption, bool) {
	return c.GetKey2(KindID, ServerID)
}

func (c *gameServiceOptionCache) GetKey1(KindID int) (map[int]*GameServiceOption, bool) {
	v, ok := c.objMap[KindID]
	return v, ok
}

func (c *gameServiceOptionCache) GetKey2(KindID int, ServerID int) (*GameServiceOption, bool) {
	v, ok := c.objMap[KindID]
	if !ok {
		return nil, false
	}
	v1, ok1 := v[ServerID]
	return v1, ok1
}
