package room_base

import (
	. "mj/common/cost"
	"mj/common/msg"
	"mj/gameServer/Chat"
	"mj/gameServer/RoomMgr"
	"mj/gameServer/db/model"
	"mj/gameServer/db/model/base"
	"mj/gameServer/user"

	"time"

	"runtime/debug"

	"github.com/lovelly/leaf/log"
	"github.com/lovelly/leaf/nsq/cluster"
)

func NewRoomUserMgr(info *msg.L2G_CreatorRoom, Temp *base.GameServiceOption) *RoomUserMgr {
	r := new(RoomUserMgr)
	r.MinUserCount = Temp.MinPlayer
	r.UserCnt = info.MaxPlayerCnt
	r.id = info.RoomID
	r.PayType = info.PayType
	r.Users = make([]*user.User, r.UserCnt)
	r.Trustee = make([]bool, r.UserCnt)
	r.Public = info.Public
	r.Onlookers = make(map[int]*user.User)
	r.ReqLeave = make(map[int64]*ReqLeaveSet)
	return r
}

type RoomUserMgr struct {
	id           int //唯一id 房间id
	Kind         int //模板表第一类型
	ServerId     int //模板表第二类型 注意 非房间id
	PayType      int //支付类型
	Public       int
	EendTime     int64              //结束时间
	MinUserCount int                //最少用户数量
	UserCnt      int                //可以容纳的用户数量
	BeginPlayer  int                //开局的人数
	PlayerCount  int                //当前用户人数
	JoinCount    int                //房主设置的游戏人数
	Users        []*user.User       /// index is chairId
	Onlookers    map[int]*user.User /// 旁观的玩家
	ChatRoomId   int                //聊天房间id
	Trustee      []bool             //是否托管 index 就是椅子id
	ReqLeave     map[int64]*ReqLeaveSet
}

type ReqLeaveSet struct {
	Refuse  []int64 //J拒绝的人uid
	Agree   []int64 //同意的人uid
	CreTime int64   //创建是按
}

func (r *RoomUserMgr) GetTrustees() []bool {
	return r.Trustee
}

func (r *RoomUserMgr) GetLeaveInfo(uid int64) *msg.LeaveReq {
	info := r.ReqLeave[uid]
	if info != nil {
		m := &msg.LeaveReq{}
		m.AgreeInfo = info.Agree
		m.LeftTimes = time.Now().Unix() - info.CreTime
		return m
	}
	return nil
}

func (r *RoomUserMgr) SetUsetTrustee(chairId int, isTruste bool) {
	r.Trustee[chairId] = isTruste
}

func (r *RoomUserMgr) GetPayType() int {
	return r.PayType
}

func (r *RoomUserMgr) IsPublic() bool {
	return r.Public == 1
}

func (r *RoomUserMgr) IsTrustee(chairId int) bool {
	return r.Trustee[chairId]
}

func (r *RoomUserMgr) GetCurPlayerCnt() int {
	r.PlayerCount = 0
	for _, u := range r.Users {
		if u != nil {
			r.PlayerCount++
		}
	}
	return r.PlayerCount
}

//开始时确定的玩家
func (r *RoomUserMgr) GetBeginPlayer() int {
	return r.BeginPlayer
}

func (r *RoomUserMgr) ResetBeginPlayer() {
	r.BeginPlayer = r.GetCurPlayerCnt()
}

func (r *RoomUserMgr) GetMaxPlayerCnt() int {
	return r.UserCnt
}

func (r *RoomUserMgr) IsInRoom(userId int64) bool {
	for _, u := range r.Users {
		if u == nil {
			continue
		}
		if u.Id == userId {
			return true
		}
	}
	return false
}

func (r *RoomUserMgr) GetUserByChairId(chairId int) *user.User {
	if len(r.Users) <= chairId {
		return nil
	}
	return r.Users[chairId]
}

func (r *RoomUserMgr) GetUserByUid(userId int64) (*user.User, int) {
	for i, u := range r.Users {
		if u == nil {
			continue
		}
		if u.Id == userId {
			return u, i
		}
	}
	return nil, -1
}

func (r *RoomUserMgr) EnterRoom(chairId int, u *user.User, status int) bool {
	_, oldChairId := r.GetUserByUid(u.Id)
	if oldChairId != -1 {
		r.Users[oldChairId] = nil
	}

	r.Users[chairId] = u
	u.ChairId = chairId
	u.RoomId = r.id
	u.ChatRoomId = r.ChatRoomId

	Chat.ChanRPC.Go("addRoomMember", r.ChatRoomId, u.Agent)

	//log.Debug("=============================u.HallNodeName:", u.HallNodeName)
	RoomMgr.UpdateRoomToHall(&msg.UpdateRoomInfo{
		RoomId: r.id,
		OpName: "AddPlayerId",
		Data: map[string]interface{}{
			"UID":          u.Id,
			"Name":         u.NickName,
			"HeadUrl":      u.HeadImgUrl,
			"Icon":         u.IconID,
			"HallNodeName": GetHallSvrName(u.HallNodeId),
			"Status":       status,
		},
	})

	return true
}

func (r *RoomUserMgr) GetChairId() int {
	for i, u := range r.Users {
		if u == nil {
			return i
		}
	}
	return -1
}

func (r *RoomUserMgr) ReplyLeave(player *user.User, Agree bool, ReplyUid int64, status int) int {
	reqPlayer, _ := r.GetUserByUid(ReplyUid)
	if reqPlayer == nil {
		log.Debug("at ReplyLeave not foud user")
		return 0
	}

	r.SendMsgAllNoSelf(player.Id, &msg.G2C_ReplyRsp{UserID: player.Id, ReplyUid: ReplyUid, Agree: Agree})
	if Agree {
		//reqPlayer.WriteMsg(&msg.G2C_ReplyRsp{UserID: player.Id, Agree: true})
		req := r.ReqLeave[ReplyUid]
		if req != nil {
			req.Agree = append(req.Agree, player.Id)
			if len(req.Agree) >= r.BeginPlayer-1 { // - 1 is self
				r.DelLeavePly(reqPlayer.Id)
				return 1
			}
		}
	} else {
		if _, ok := r.ReqLeave[reqPlayer.Id]; ok {
			r.DelLeavePly(reqPlayer.Id)
			return -1
		}
	}

	return 0
}

func (r *RoomUserMgr) DelLeavePly(uid int64) {
	delete(r.ReqLeave, uid)
}

func (r *RoomUserMgr) AddLeavePly(uid int64) {
	req := &ReqLeaveSet{CreTime: time.Now().Unix()}
	r.ReqLeave[uid] = req
}

func (r *RoomUserMgr) LeaveRoom(u *user.User, status int) bool {
	log.Debug("at LeaveRoom uid:%d", u.Id)
	if len(r.Users) <= u.ChairId {
		log.Error("at LeaveRoom u.chairId max .... chaird %d, stack:%s", u.ChairId, string(debug.Stack()))
		return false
	}
	err := model.GamescorelockerOp.UpdateWithMap(u.Id, map[string]interface{}{
		"GameNodeID": 0,
		"EnterIP":    "",
	})
	if err != nil {
		log.Error("at EnterRoom  updaye .Gamescorelocker error:%s", err.Error())
	}

	r.SetUsetStatus(u, US_FREE)

	u.ChanRPC().Go("LeaveRoom")
	r.Users[u.ChairId] = nil
	u.ChairId = INVALID_CHAIR
	u.RoomId = 0
	u.ChatRoomId = 0

	Chat.ChanRPC.Go("delRoomMember", r.ChatRoomId, u.Id)

	RoomMgr.UpdateRoomToHall(&msg.UpdateRoomInfo{
		RoomId: r.id,
		OpName: "DelPlayerId",
		Data: map[string]interface{}{
			"Status":  status,
			"UID":     u.Id,
			"PayType": r.PayType,
		},
	})
	log.Debug("%v user leave room,  left %v count", u.Id, r.GetCurPlayerCnt())
	return true
}

func (r *RoomUserMgr) SendMsg(chairId int, data interface{}) bool {
	if len(r.Users) <= chairId {
		return false
	}

	u := r.Users[chairId]
	if u == nil {
		return false
	}

	u.WriteMsg(data)
	return true
}

func (r *RoomUserMgr) SendMsgAll(data interface{}) {
	for _, u := range r.Users {
		if u != nil {
			u.WriteMsg(data)
		}
	}
}

func (r *RoomUserMgr) SendOnlookers(data interface{}) {
	for _, u := range r.Onlookers {
		if u != nil {
			u.WriteMsg(data)
		}
	}
}

func (r *RoomUserMgr) SendMsgAllNoSelf(selfid int64, data interface{}) {
	for _, u := range r.Users {
		if u != nil && u.Id != selfid {
			u.WriteMsg(data)
		}
	}
}

func (r *RoomUserMgr) ForEachUser(fn func(u *user.User)) {
	for _, u := range r.Users {
		if u != nil {
			fn(u)
		}
	}
}

func (r *RoomUserMgr) WriteTableScore(source []*msg.TagScoreInfo, usercnt, Type int) {
	for _, u := range r.Users {
		if u.ChanRPC() != nil {
			u.ChanRPC().Go("WriteUserScore", source[u.ChairId], Type)
		}
	}
}

//获取对方信息
func (room *RoomUserMgr) GetUserInfoByChairId(ChairID int) interface{} {
	tagUser := room.GetUserByChairId(ChairID)
	if tagUser == nil {
		log.Error("at GetUserChairInfo no foud tagUser %v", ChairID)
		return nil
	}

	return &msg.G2C_UserEnter{
		UserID:      tagUser.Id,          //用户 I D
		FaceID:      tagUser.FaceID,      //头像索引
		CustomID:    tagUser.CustomID,    //自定标识
		Gender:      tagUser.Gender,      //用户性别
		MemberOrder: tagUser.MemberOrder, //会员等级
		TableID:     tagUser.RoomId,      //桌子索引
		ChairID:     tagUser.ChairId,     //椅子索引
		UserStatus:  tagUser.Status,      //用户状态
		Score:       tagUser.Score,       //用户分数
		WinCount:    tagUser.WinCount,    //胜利盘数
		LostCount:   tagUser.LostCount,   //失败盘数
		DrawCount:   tagUser.DrawCount,   //和局盘数
		FleeCount:   tagUser.FleeCount,   //逃跑盘数
		Experience:  tagUser.Experience,  //用户经验
		NickName:    tagUser.NickName,    //昵称
		HeaderUrl:   tagUser.HeadImgUrl,  //头像
	}
}

//坐下
func (room *RoomUserMgr) Sit(u *user.User, ChairID int, status int) int {
	if ChairID == INVALID_CHAIR {
		ChairID = room.GetChairId()
		if ChairID == -1 {
			return ErrRoomFull
		}
	}
	oldUser := room.GetUserByChairId(ChairID)
	if oldUser != nil {
		return ChairHasUser
	}

	if room.ChatRoomId == 0 {
		id, err := Chat.ChanRPC.Call1("createRoom", u.Agent)
		if err != nil {
			log.Error("create Chat Room faild")
			return ErrCreateRoomFaild
		}
		room.ChatRoomId = id.(int)
	}

	room.EnterRoom(ChairID, u, status)

	//把自己的信息推送给所有玩家
	room.NotifyUserInfo(u)

	room.SetUsetStatus(u, US_SIT)
	return 0
}

//广播某个玩家的信息
func (room *RoomUserMgr) NotifyUserInfo(u *user.User) {
	room.SendMsgAllNoSelf(u.Id, &msg.G2C_UserEnter{
		KindID:      u.KindID,      //游戏id
		UserID:      u.Id,          //用户 I D
		FaceID:      u.FaceID,      //头像索引
		CustomID:    u.CustomID,    //自定标识
		Gender:      u.Gender,      //用户性别
		MemberOrder: u.MemberOrder, //会员等级
		TableID:     u.RoomId,      //桌子索引
		ChairID:     u.ChairId,     //椅子索引
		UserStatus:  u.Status,      //用户状态
		Score:       u.Score,       //用户分数
		WinCount:    u.WinCount,    //胜利盘数
		LostCount:   u.LostCount,   //失败盘数
		DrawCount:   u.DrawCount,   //和局盘数
		FleeCount:   u.FleeCount,   //逃跑盘数
		Experience:  u.Experience,  //用户经验
		NickName:    u.NickName,    //昵称
		HeaderUrl:   u.HeadImgUrl,  //头像
	})
}

func (room *RoomUserMgr) SendUserInfoToSelf(u *user.User) {
	room.ForEachUser(func(eachuser *user.User) {
		if eachuser.Id == u.Id {
			return
		}
		u.WriteMsg(&msg.G2C_UserEnter{
			KindID:      u.KindID,             //游戏id
			UserID:      eachuser.Id,          //用户 I D
			FaceID:      eachuser.FaceID,      //头像索引
			CustomID:    eachuser.CustomID,    //自定标识
			Gender:      eachuser.Gender,      //用户性别
			MemberOrder: eachuser.MemberOrder, //会员等级
			TableID:     eachuser.RoomId,      //桌子索引
			ChairID:     eachuser.ChairId,     //椅子索引
			UserStatus:  eachuser.Status,      //用户状态
			Score:       eachuser.Score,       //用户分数
			WinCount:    eachuser.WinCount,    //胜利盘数
			LostCount:   eachuser.LostCount,   //失败盘数
			DrawCount:   eachuser.DrawCount,   //和局盘数
			FleeCount:   eachuser.FleeCount,   //逃跑盘数
			Experience:  eachuser.Experience,  //用户经验
			NickName:    eachuser.NickName,    //昵称
			HeaderUrl:   eachuser.HeadImgUrl,  //头像
		})
	})
}

//起立
func (room *RoomUserMgr) Standup(u *user.User) bool {
	//room.LeaveRoom(u, 1)
	return true
}

func (room *RoomUserMgr) SetUsetStatus(u *user.User, stu int) {
	u.Status = stu
	room.SendMsgAll(&msg.G2C_UserStatus{
		UserID: u.Id,
		UserStatus: &msg.UserStu{
			TableID:    room.id,
			ChairID:    u.ChairId,
			UserStatus: u.Status,
		},
	})
}

//通知房间解散
func (room *RoomUserMgr) RoomDissume(Reason int) {

	room.SendMsgAll(&msg.G2C_CancelTable{})
	room.SendMsgAll(&msg.G2C_PersonalTableEnd{})
	room.SendMsgAll(&msg.G2C_RoomDissume{Reason: Reason})

	for _, u := range room.Users {
		if u != nil {
			u.ChanRPC().Go("LeaveRoom")

			err := model.GamescorelockerOp.UpdateWithMap(u.Id, map[string]interface{}{
				"GameNodeID": 0,
				"EnterIP":    "",
			})
			if err != nil {
				log.Error("at RoomDissume  updaye .Gamescorelocker error:%s", err.Error())
			}
		}
	}

	if room.ChatRoomId != 0 {
		Chat.ChanRPC.Go("closeChatRoom", room.ChatRoomId)
	}
}

func (room *RoomUserMgr) IsAllReady() bool {
	PlayerCount := 0
	for _, u := range room.Users {
		if u == nil {
			continue
		}

		if u.Status != US_READY {
			return false
		}
		PlayerCount++
	}
	if PlayerCount < room.MinUserCount || PlayerCount > room.UserCnt {
		return false
	}

	return true
}

func (room *RoomUserMgr) ReLogin(u *user.User, Status int) {
	room.Users[u.ChairId] = u
	if Status == RoomStatusStarting {
		room.SetUsetStatus(u, US_PLAYING)
	} else {
		if u.Status == US_OFFLINE {
			room.SetUsetStatus(u, US_SIT)
		}
	}

	Chat.ChanRPC.Go("addRoomMember", room.ChatRoomId, u.Agent)
}

func (room *RoomUserMgr) SendDataToHallUser(chiairID int, data interface{}) {
	u := room.GetUserByChairId(chiairID)
	if u == nil {
		return
	}

	cluster.SendMsgToHallUser(u.HallNodeId, u.Id, data)
}

func (room *RoomUserMgr) SendMsgToHallServerAll(data interface{}) {
	for _, u := range room.Users {
		if u == nil {
			continue
		}
		cluster.SendMsgToHallUser(u.HallNodeId, u.Id, data)
	}
}

//检测房间是否该返还房主钱
func (room *RoomUserMgr) CheckRoomReturnMoney(roomStatus, CreatorNodeId, roomId int, creatorId int64) {
	//全付的房间，并且没开始过游戏
	if room.PayType != SELF_PAY_TYPE && roomStatus != RoomStatusReady {
		return
	}
	//要求房主没在房间内才在这边返还，否则走的是其他逻辑返还
	isCreatorInRoom := false
	room.ForEachUser(func(u *user.User) {
		if u.Id == creatorId {
			isCreatorInRoom = true
		}
	})
	log.Debug("################ CheckRoomReturnMoney isCreatorInRoom=%v", isCreatorInRoom)
	if !isCreatorInRoom {
		cluster.SendMsgToHallUser(CreatorNodeId, creatorId, &msg.RoomReturnMoney{RoomId: roomId, CreatorUid: creatorId})
	}
}
