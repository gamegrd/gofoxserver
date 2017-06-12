package room

import (
	"fmt"
	. "mj/common/cost"
	"mj/common/msg"
	"mj/gameServer/base"
	"mj/gameServer/common"
	"mj/gameServer/common/room_base"
	tbase "mj/gameServer/db/model/base"
	"mj/gameServer/idGenerate"
	"strconv"
	"sync"
	"time"

	"github.com/lovelly/leaf/chanrpc"
	"github.com/lovelly/leaf/log"
	"github.com/lovelly/leaf/module"
)

var (
	idLock sync.RWMutex
	IncId  = 0
)

func NewRoom(mgrCh *chanrpc.Server, param *msg.C2G_CreateTable, t *tbase.GameServiceOption, rid, userCnt, uid int) *Room {
	skeleton := base.NewSkeleton()
	room := new(Room)
	room.Skeleton = skeleton
	room.ChanRPC = skeleton.ChanRPCServer
	room.mgrCh = mgrCh
	room.RoomInfo = room_base.NewRoomInfo(userCnt, rid)
	room.Kind = t.KindID
	room.ServerId = t.ServerID
	room.Name = fmt.Sprintf(strconv.Itoa(common.KIND_TYPE_HZMJ)+"_%v", room.GetRoomId())
	room.CloseSig = make(chan bool, 1)
	room.TimeLimit = param.DrawTimeLimit
	room.CountLimit = param.DrawCountLimit
	room.Source = param.CellScore
	room.Password = param.Password
	room.JoinGamePeopleCount = param.JoinGamePeopleCount
	room.CreateUser = uid
	room.CustomRule = new(msg.CustomRule)
	room.Response = make([]bool, userCnt)
	room.gameLogic = NewGameLogic()

	room.Owner = uid
	room.BankerUser = INVALID_CHAIR
	room.Record = &msg.G2C_Record{HuCount: make([]int, room.UserCnt), MaCount: make([]int, room.UserCnt), AnGang: make([]int, room.UserCnt), MingGang: make([]int, room.UserCnt), AllScore: make([]int, room.UserCnt), DetailScore: make([][]int, room.UserCnt)}
	now := time.Now().Unix()
	room.TimeStartGame = now
	room.EendTime = now + 900
	room.CardIndex = make([][]int, room.UserCnt)
	room.HeapCardInfo = make([][]int, room.UserCnt) //堆牌信息
	room.HistoryScores = make([]*HistoryScore, room.UserCnt)
	RegisterHandler(room)
	room.OnInit()
	go room.run()
	log.Debug("new room ok .... ")
	return room
}

//吧room 当一张桌子理解
type Room struct {
	// module 必须字段
	*module.Skeleton
	ChanRPC  *chanrpc.Server //接受客户端消息的chan
	mgrCh    *chanrpc.Server //管理类的chan 例如红中麻将 就是红中麻将module的 ChanRPC
	CloseSig chan bool
	wg       sync.WaitGroup //

	// 游戏字段
	*room_base.RoomInfo
	ChatRoomId          int                //聊天房间id
	Name                string             //房间名字
	Kind                int                //第一类型
	ServerId            int                //第二类型 注意 非房间id
	Source              int                //底分
	IniSource           int                //初始分数
	TimeLimit           int                //时间显示
	CountLimit          int                //局数限制
	IsGoldOrGameScore   int                //金币场还是积分场 0 标识 金币场 1 标识 积分场
	Password            string             // 密码
	JoinGamePeopleCount int                //参与游戏的人数
	*msg.CustomRule                        //自定义规则
	Record              *msg.G2C_Record    //约战类型特殊记录
	IsDissumGame        bool               //是否强制解散游戏
	MagicIndex          int                //财神索引
	ProvideCard         int                //供应扑克
	ResumeUser          int                //还原用户
	ProvideUser         int                //供应用户
	LeftCardCount       int                //剩下拍的数量
	EndLeftCount        int                //荒庄牌数
	LastCatchCardUser   int                //最后一个摸牌的用户
	Owner               int                //房主id
	OutCardCount        int                //出牌数目
	ChiHuCard           int                //吃胡扑克
	MinusHeadCount      int                //头部空缺
	MinusLastCount      int                //尾部空缺
	SiceCount           int                //色子大小
	SendCardCount       int                //发牌数目
	UserActionDone      bool               //操作完成
	SendStatus          int                //发牌状态
	GangStatus          int                //杠牌状态
	GangOutCard         bool               //杠后出牌
	ProvideGangUser     int                //供杠用户
	GangCard            []bool             //杠牌状态
	GangCount           []int              //杠牌次数
	RepertoryCard       []int              //库存扑克
	UserGangScore       []int              //游戏中杠的输赢
	Response            []bool             //响应标志
	ChiHuKind           []int              //吃胡结果
	ChiHuRight          []int              //胡牌类型
	UserMaCount         []int              //下注用户数
	UserAction          []int              //用户动作
	OperateCard         [][]int            //操作扑克
	ChiPengCount        []int              //吃碰杠次数
	PerformAction       []int              //执行动作
	HandCardCount       []int              //扑克数目
	CardIndex           [][]int            //用户扑克[GAME_PLAYER][MAX_INDEX]
	WeaveItemCount      []int              //组合数目
	WeaveItemArray      [][]*msg.WeaveItem //组合扑克
	DiscardCount        []int              //丢弃数目
	DiscardCard         [][]int            //丢弃记录
	OutCardData         int                //出牌扑克
	OutCardUser         int                //当前出牌用户
	HeapHead            int                //堆立头部
	HeapTail            int                //堆立尾部
	HeapCardInfo        [][]int            //堆牌信息
	SendCardData        int                //发牌扑克
	HistoryScores       []*HistoryScore

	gameLogic *GameLogic
}

func (r *Room) run() {
	log.Debug("room Room start run Name:%s", r.Name)
	r.Run(r.CloseSig)
	log.Debug("room Room End run Name:%s", r.Name)
}

func (r *Room) Destroy() {
	defer func() {
		if r := recover(); r != nil {
			log.Recover(r)
		}
	}()

	r.CloseSig <- true
	r.OnDestroy()
	log.Debug("room Room Destroy ok,  Name:%s", r.Name)
}

func (r *Room) GetCurlPlayerCount() int {
	cnt := 0
	for _, u := range r.Users {
		if u != nil {
			cnt++
		}
	}

	return cnt
}

func (r *Room) GetChanRPC() *chanrpc.Server {
	return r.ChanRPC
}

////////////////// 上面run 和 Destroy 请勿随意修改 //////  下面函数自由操作
func (r *Room) OnInit() {
	r.Skeleton.AfterFunc(10*time.Second, r.checkDestroyRoom)
}

func (r *Room) OnDestroy() {
	idGenerate.DelRoomId(r.GetRoomId())
}

//这里添加定时操作
func (r *Room) checkDestroyRoom() {
	nowTime := time.Now().Unix()
	if r.CheckDestroy(nowTime) {
		r.Destroy()
		return
	}

	r.Skeleton.AfterFunc(10*time.Second, r.checkDestroyRoom)
}