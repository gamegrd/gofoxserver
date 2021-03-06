package internal

import (
	"encoding/json"
	"fmt"
	. "mj/common/cost"
	"mj/common/msg"
	"mj/common/register"
	"mj/common/utils"
	"mj/hallServer/common"
	"mj/hallServer/conf"
	"mj/hallServer/db/model"
	"mj/hallServer/db/model/account"
	"mj/hallServer/db/model/base"
	"mj/hallServer/game_list"
	"mj/hallServer/id_generate"
	datalog "mj/hallServer/log"
	"mj/hallServer/match_room"
	"mj/hallServer/user"
	"time"

	"github.com/lovelly/leaf/gate"
	"github.com/lovelly/leaf/log"
	"github.com/lovelly/leaf/nsq/cluster"
)

func RegisterHandler(m *UserModule) {
	reg := register.NewRegister(m.ChanRPC)
	//注册rpc 消息
	reg.RegisterRpc("handleMsgData", m.handleMsgData)
	reg.RegisterRpc("NewAgent", m.NewAgent)
	reg.RegisterRpc("CloseAgent", m.CloseAgent)
	reg.RegisterRpc("GetUser", m.GetUser)
	reg.RegisterRpc("SearchTableResult", m.SearchTableResult)
	reg.RegisterRpc("matchResult", m.matchResult)
	reg.RegisterRpc("LeaveRoom", m.leaveRoom)
	reg.RegisterRpc("JoinRoom", m.joinRoom)
	reg.RegisterRpc("GameStart", m.GameStart)
	reg.RegisterRpc("RoomEndInfo", m.RoomEndInfo)
	reg.RegisterRpc("RoomReturnMoney", m.RoomReturnMoney)
	reg.RegisterRpc("JoinRoomFaild", m.JoinRoomFaild)

	reg.RegisterRpc("Recharge", m.Recharge)
	reg.RegisterRpc("S2S_RenewalFeeResult", m.RenewalFeeResult)
	reg.RegisterRpc("S2S_OfflineHandler", m.HandlerOffilneEvent)
	reg.RegisterRpc("ForceClose", m.ForceClose)
	reg.RegisterRpc("SvrShutdown", m.SvrShutdown)
	reg.RegisterRpc("DeleteVaildIds", m.DeleteVaildIds)
	reg.RegisterRpc("AddRoomRecord", m.AddRoomRecord)
	reg.RegisterRpc("DelRoomRecord", m.DelRoomRecord)

	//c2s
	reg.RegisterC2S(&msg.C2L_Login{}, m.handleMBLogin)
	reg.RegisterC2S(&msg.C2L_ReConnect{}, m.handleReconnect)
	reg.RegisterC2S(&msg.C2L_Regist{}, m.handleMBRegist)
	reg.RegisterC2S(&msg.C2L_User_Individual{}, m.GetUserIndividual)
	reg.RegisterC2S(&msg.C2L_CreateTable{}, m.CreateRoom)
	reg.RegisterC2S(&msg.C2L_ReqCreatorRoomRecord{}, m.GetCreatorRecord)
	reg.RegisterC2S(&msg.C2L_ReqRoomPlayerBrief{}, m.GetRoomPlayerBreif)
	reg.RegisterC2S(&msg.C2L_DrawSahreAward{}, m.DrawSahreAward)
	reg.RegisterC2S(&msg.C2L_SetElect{}, m.SetElect)
	reg.RegisterC2S(&msg.C2L_DeleteRoom{}, m.DeleteRoom)
	reg.RegisterC2S(&msg.C2L_SetPhoneNumber{}, m.SetPhoneNumber)
	reg.RegisterC2S(&msg.C2L_DianZhan{}, m.DianZhan)
	reg.RegisterC2S(&msg.C2L_RenewalFees{}, m.RenewalFees)
	reg.RegisterC2S(&msg.C2L_ChangeUserName{}, m.ChangeUserName)
	reg.RegisterC2S(&msg.C2L_ChangeSign{}, m.ChangeSign)
	reg.RegisterC2S(&msg.C2L_ReqBindMaskCode{}, m.ReqBindMaskCode)
	reg.RegisterC2S(&msg.C2L_RechangerOk{}, m.RechangerOk)
	reg.RegisterC2S(&msg.C2L_ReqTimesInfo{}, m.ReqTimesInfo)
	reg.RegisterC2S(&msg.C2L_TimeSync{}, m.TimeSync)
	reg.RegisterC2S(&msg.C2L_GetRoomRecord{}, m.GetRoomRecord)
	reg.RegisterC2S(&msg.C2L_GetUserRecords{}, m.GetUserRecord)
}

//连接进来的通知
func (m *UserModule) NewAgent(args []interface{}) error {
	log.Debug("at hall NewAgent")
	return nil
}

//连接关闭的通知
func (m *UserModule) CloseAgent(args []interface{}) error {
	defer func() {
		m.closeCh <- true
	}()
	log.Debug("at hall CloseAgent")
	agent := args[0].(gate.Agent)
	Reason := args[1].(int)
	player, ok := agent.UserData().(*user.User)
	if !ok || player == nil {
		log.Error("at CloseAgent not foud user")
		return nil
	}

	m.UserOffline()
	if Reason != KickOutMsg { //重登踢出会覆盖， 所以这里不用删除
		DelUser(player.Id)
	}

	log.Debug("CloseAgent ok")
	return nil
}

func (m *UserModule) handleMBLogin(args []interface{}) {
	recvMsg := args[0].(*msg.C2L_Login)
	retMsg := &msg.L2C_LogonSuccess{}
	agent := m.a
	retcode := 0
	log.Debug("enter mbLogin  user:%s", recvMsg.Accounts)

	defer func() {
		if retcode != 0 {
			str := fmt.Sprintf("登录失败, 错误码: %d", retcode)
			agent.WriteMsg(&msg.L2C_LogonFailure{ResultCode: retcode, DescribeString: str})
			m.Close(KickOutUnlawfulMsg)
		}
	}()

	if recvMsg.Accounts == "" {
		retcode = ParamError
		return
	}

	accountData, aerr := account.AccountsinfoOp.GetByMap(map[string]interface{}{
		"Accounts": recvMsg.Accounts,
	})

	if aerr != nil {
		log.Debug("error at AccountsinfoOp GetByMap %s", aerr.Error())
		retcode = LoadUserInfoError
		return
	}

	if accountData == nil {
		if conf.Test {
			retcode, _, accountData = RegistUser(&msg.C2L_Regist{
				LogonPass:    recvMsg.LogonPass,
				Accounts:     recvMsg.Accounts,
				ModuleID:     recvMsg.ModuleID,
				PlazaVersion: recvMsg.PlazaVersion,
				MachineID:    recvMsg.MachineID,
				MobilePhone:  recvMsg.MobilePhone,
				NickName:     recvMsg.Accounts,
			}, agent)
			if retcode != 0 {
				return
			}
		} else {
			retcode = NotFoudAccout
			return
		}
	}

	if accountData.LogonPass != recvMsg.LogonPass {
		retcode = ErrPasswd
		return
	}

	player := user.NewUser(accountData.UserID)
	player.Id = accountData.UserID
	lok := loadUser(player)
	if !lok {
		retcode = LoadUserInfoError
		return
	}

	if player.Roomid != 0 {
		_, have := game_list.ChanRPC.Call1("HaseRoom", player.Roomid)
		if have != nil {
			log.Debug("user :%d room %d is close ", player.Id, player.Roomid)
			player.DelGameLockInfo()
		}
	}

	oldUser := getUser(accountData.UserID)
	if oldUser != nil {
		log.Debug("old user ====== %d  %d ", oldUser.KindID, oldUser.Roomid)
		m.KickOutUser(oldUser)
	}

	player.Agent = agent
	AddUser(player.Id, player)
	agent.SetUserData(player)
	player.LoadTimes()
	player.HallNodeID = conf.Server.NodeId
	model.GamescorelockerOp.UpdateWithMap(player.Id, map[string]interface{}{
		"HallNodeID": conf.Server.NodeId,
	})

	game_list.ChanRPC.Call0("sendGameList", agent)
	BuildClientMsg(retMsg, player, accountData)
	agent.WriteMsg(retMsg)
	agent.WriteMsg(&msg.L2C_ServerListFinish{})

	m.Recharge(nil)

	//加载离线处理
	loadHandles(player)

	ids := player.GetRoomIds()
	if len(ids) > 0 {
		game_list.ChanRPC.Go("CheckVaildIds", ids, m.ChanRPC)
	}
}

//重连
func (m *UserModule) handleReconnect(args []interface{}) {
	recvMsg := args[0].(*msg.C2L_ReConnect)
	retMsg := &msg.L2C_ReConnectRsp{}
	agent := m.a
	retMsg.Code = -1

	log.Debug("enter mbLogin  user:%s", recvMsg.Accounts)

	defer func() {
		if retMsg.Code != 0 {
			m.Close(KickOutUnlawfulMsg)
			str := fmt.Sprintf("重连失败, 错误码: %d", retMsg.Code)
			agent.WriteMsg(&msg.L2C_LogonFailure{ResultCode: retMsg.Code, DescribeString: str})
		} else {
			agent.WriteMsg(retMsg)
		}
	}()

	if recvMsg.Accounts == "" {
		retMsg.Code = ParamError
		return
	}

	accountData, ok := account.AccountsinfoOp.GetByMap(map[string]interface{}{
		"Accounts": recvMsg.Accounts,
	})

	if ok != nil || accountData == nil {
		retMsg.Code = NotFoudAccout
		return
	}

	if accountData.LogonPass != recvMsg.LogonPass {
		retMsg.Code = ErrPasswd
		return
	}

	player := user.NewUser(accountData.UserID)
	player.Id = accountData.UserID
	lok := loadUser(player)
	if !lok {
		retMsg.Code = LoadUserInfoError
		return
	}

	if player.Roomid != 0 {
		_, have := game_list.ChanRPC.Call1("HaseRoom", player.Roomid)
		if have != nil {
			log.Debug("user :%d room %d is close ", player.Id, player.Roomid)
			player.DelGameLockInfo()
		}
	}

	oldUser := getUser(accountData.UserID)
	if oldUser != nil {
		log.Debug("ar handleReconnect old user ====== %d  %d ", oldUser.KindID, oldUser.Roomid)
		m.KickOutUser(oldUser)
	}

	player.Agent = agent
	AddUser(player.Id, player)
	agent.SetUserData(player)
	player.LoadTimes()
	player.HallNodeID = conf.Server.NodeId
	model.GamescorelockerOp.UpdateWithMap(player.Id, map[string]interface{}{
		"HallNodeID": conf.Server.NodeId,
	})
	infoMsg := &msg.L2C_LogonSuccess{}
	BuildClientMsg(infoMsg, player, accountData)
	player.WriteMsg(infoMsg)
	retMsg.Code = 0
}

//请求同步时间
func (m *UserModule) TimeSync(args []interface{}) {
	player := m.a.UserData().(*user.User)
	player.WriteMsg(&msg.L2C_TimeSync{ServerTime: time.Now().Unix()})
}

func (m *UserModule) handleMBRegist(args []interface{}) {
	retcode := 0
	recvMsg := args[0].(*msg.C2L_Regist)
	agent := args[1].(gate.Agent)
	retMsg := &msg.L2C_RegistResult{}
	var accountData *account.Accountsinfo
	defer func() {
		if retcode != 0 {
			account.AccountsinfoOp.DeleteByMap(map[string]interface{}{
				"Accounts": recvMsg.Accounts,
			})
			if accountData != nil {
				model.AccountsmemberOp.Delete(accountData.UserID)
				model.GamescorelockerOp.Delete(accountData.UserID)
				model.GamescoreinfoOp.Delete(accountData.UserID)
				model.UserattrOp.Delete(accountData.UserID)
				model.UserextrainfoOp.Delete(accountData.UserID)
				model.UsertokenOp.Delete(accountData.UserID)
			}
			agent.WriteMsg(RenderErrorMessage(retcode, "注册失败"))
		} else {
			agent.WriteMsg(retMsg)
		}
	}()

	retcode, _, _ = RegistUser(recvMsg, agent)
}

func RegistUser(recvMsg *msg.C2L_Regist, agent gate.Agent) (int, *user.User, *account.Accountsinfo) {
	accountData, _ := account.AccountsinfoOp.GetByMap(map[string]interface{}{
		"Accounts": recvMsg.Accounts,
	})
	if accountData != nil {
		return AlreadyExistsAccount, nil, nil
	}

	//todo 名字排重等等等 验证
	now := time.Now()
	accInfo := &account.Accountsinfo{
		UserID:           user.GetUUID(),
		Gender:           recvMsg.Gender,   //用户性别
		Accounts:         recvMsg.Accounts, //登录帐号
		LogonPass:        recvMsg.LogonPass,
		InsurePass:       recvMsg.InsurePass,
		NickName:         recvMsg.NickName, //用户昵称
		GameLogonTimes:   1,
		LastLogonIP:      agent.RemoteAddr().String(),
		LastLogonMobile:  recvMsg.MobilePhone,
		LastLogonMachine: recvMsg.MachineID,
		RegisterMobile:   recvMsg.MobilePhone,
		RegisterMachine:  recvMsg.MachineID,
		RegisterDate:     &now,
		RegisterIP:       agent.RemoteAddr().String(), //连接地址
	}

	_, err := account.AccountsinfoOp.Insert(accInfo)
	if err != nil {
		log.Error("RegistUser err :%s", err.Error())
		return InsertAccountError, nil, nil
	}

	player, cok := createUser(accInfo.UserID, accInfo)
	if !cok {
		return CreateUserError, nil, nil
	}
	return 0, player, accInfo
}

//获取个人信息
func (m *UserModule) GetUserIndividual(args []interface{}) {
	recvMsg := args[0].(*msg.C2L_User_Individual)
	agent := args[1].(gate.Agent)
	player, ok := agent.UserData().(*user.User)
	if !ok {
		log.Debug("not foud user data")
		return
	}

	var retmsg *msg.L2C_UserIndividual
	if recvMsg.UserId == player.Id || recvMsg.UserId == 0 {
		retmsg = &msg.L2C_UserIndividual{
			UserID:      player.Id,        //用户 I D
			NickName:    player.NickName,  //昵称
			WinCount:    player.WinCount,  //赢数
			LostCount:   player.LostCount, //输数
			DrawCount:   player.DrawCount, //平数
			Medal:       player.UserMedal,
			RoomCard:    player.Currency,    //房卡
			MemberOrder: player.MemberOrder, //会员等级
			Score:       player.Score,
			HeadImgUrl:  player.HeadImgUrl,
			Star:        player.Star,
			Sign:        player.Sign,
			PhomeNumber: player.PhomeNumber,
			ElectUid:    player.ElectUid,
		}
	} else {
		userAttr, ok := model.UserattrOp.Get(recvMsg.UserId)
		source, ok1 := model.GamescoreinfoOp.Get(recvMsg.UserId)
		if !ok || !ok1 {
			log.Error("not found user info :%d", recvMsg.UserId)
			return
		}
		retmsg = &msg.L2C_UserIndividual{
			UserID:      userAttr.UserID,   //用户 I D
			NickName:    userAttr.NickName, //昵称
			WinCount:    source.WinCount,   //赢数
			LostCount:   source.LostCount,  //输数
			DrawCount:   source.DrawCount,  //平数
			Medal:       userAttr.UserMedal,
			RoomCard:    0, //房卡
			MemberOrder: 0, //会员等级
			Score:       source.Score,
			HeadImgUrl:  userAttr.HeadImgUrl,
			Star:        userAttr.Star,
			Sign:        userAttr.Sign,
			PhomeNumber: "",
			ElectUid:    player.ElectUid,
		}
	}

	player.WriteMsg(retmsg)
}

func (m *UserModule) UserOffline() {

}

func (m *UserModule) CreateRoom(args []interface{}) {
	recvMsg := args[0].(*msg.C2L_CreateTable)
	retMsg := &msg.L2C_CreateTableSucess{}
	agent := args[1].(gate.Agent)
	retCode := -1
	defer func() {
		if retCode == 0 {
			agent.WriteMsg(retMsg)
		} else {
			agent.WriteMsg(&msg.L2C_CreateTableFailure{ErrorCode: retCode, DescribeString: "创建房间失败"})
		}
	}()
	template, ok := base.GameServiceOptionCache.Get(recvMsg.Kind, recvMsg.ServerId)
	if !ok {
		log.Debug("not foud template %d,%d", recvMsg.Kind, recvMsg.ServerId)
		retCode = NoFoudTemplate
		return
	}

	feeTemp, ok1 := base.PersonalTableFeeCache.Get(recvMsg.Kind, recvMsg.ServerId, recvMsg.DrawCountLimit)
	if !ok1 {
		log.Error("not foud PersonalTableFeeCache")
		retCode = NoFoudTemplate
		return
	}

	player := agent.UserData().(*user.User)
	if player.GetRoomCnt() >= common.GetGlobalVarInt(MAX_CREATOR_ROOM_CNT) {
		retCode = ErrMaxRoomCnt
		return
	}

	host, nodeId := game_list.GetSvrByKind(recvMsg.Kind)
	if host == "" {
		retCode = ErrNotFoudServer
		return
	}

	rid, iok := id_generate.GenerateRoomId(nodeId)
	if !iok {
		retCode = RandRoomIdError
		return
	}

	playerCnt := recvMsg.PlayerCnt
	if playerCnt < template.MinPlayer || playerCnt > template.MaxPlayer {
		log.Debug(" client player ctn invalid %d ", playerCnt)
		playerCnt = template.MaxPlayer
	}

	//检测是否有限时免费
	if !player.CheckFree() {
		//AA在加入的时候扣钱
		log.Debug(" kou qian ............ ")
		if recvMsg.PayType == SELF_PAY_TYPE {
			log.Debug(" kou qian ............ 111111111111  ")
			money := feeTemp.TableFee
			if !player.SubCurrency(money, recvMsg.PayType) {
				retCode = NotEnoughFee
				return
			}
			record := &model.TokenRecord{}
			record.UserId = player.Id
			record.RoomId = rid
			record.Amount = money
			record.TokenType = recvMsg.PayType
			record.KindID = recvMsg.Kind
			record.PlayCnt = recvMsg.DrawCountLimit
			if !player.AddRecord(record) {
				retCode = ErrServerError
				player.AddCurrency(money)
				return
			}
		}
	}

	//日志：创建房间数据
	data := &datalog.RoomLog{}
	data.AddCreateRoomLog(rid, player.UserId, recvMsg.RoomName, recvMsg.Kind, recvMsg.ServerId, nodeId, recvMsg.PayType, retCode)

	_, err := cluster.Call1GameSvr(nodeId, &msg.L2G_CreatorRoom{
		CreatorUid:    player.Id,
		PayType:       recvMsg.PayType,
		MaxPlayerCnt:  playerCnt,
		RoomID:        rid,
		PlayCnt:       recvMsg.DrawCountLimit,
		KindId:        recvMsg.Kind,
		ServiceId:     recvMsg.ServerId,
		OtherInfo:     recvMsg.OtherInfo,
		CreatorNodeId: conf.Server.NodeId,
	})
	if err != nil {
		retCode = ErrCreaterError
	}

	//记录创建房间信息
	info := &model.CreateRoomInfo{}
	info.UserId = player.Id
	info.PayType = recvMsg.PayType
	info.MaxPlayerCnt = template.MaxPlayer
	info.RoomId = rid
	info.NodeId = nodeId
	info.Num = recvMsg.DrawCountLimit
	info.KindId = recvMsg.Kind
	info.ServiceId = recvMsg.ServerId
	now := time.Now()
	info.CreateTime = &now
	if recvMsg.Public {
		info.Public = 1
	} else {
		info.Public = 0
	}

	by, err := json.Marshal(recvMsg.OtherInfo)
	if err != nil {
		log.Error("at CreateRoom json.Marshal(recvMsg.OtherInfo) error:%s", err.Error())
		retCode = ErrParamError
		return
	}
	info.OtherInfo = string(by)
	if recvMsg.RoomName != "" {
		info.RoomName = recvMsg.RoomName
	} else {
		info.RoomName = template.RoomName
	}

	player.AddRooms(info)

	roomInfo := &msg.RoomInfo{}
	roomInfo.KindID = info.KindId
	roomInfo.ServerID = info.ServiceId
	roomInfo.RoomID = info.RoomId
	roomInfo.NodeID = info.NodeId
	roomInfo.SvrHost = host
	roomInfo.PayType = info.PayType
	roomInfo.CreateTime = time.Now().Unix()
	roomInfo.CreateUserId = player.Id
	roomInfo.IsPublic = recvMsg.Public
	roomInfo.MachPlayer = make(map[int64]int64)
	roomInfo.Players = make(map[int64]*msg.PlayerBrief)
	roomInfo.MaxPlayerCnt = info.MaxPlayerCnt
	roomInfo.PayCnt = info.Num
	roomInfo.RoomPlayCnt = info.Num
	roomInfo.RoomName = info.RoomName
	game_list.ChanRPC.Go("addyNewRoom", roomInfo)

	//回给客户端的消息
	retMsg.TableID = rid
	retMsg.DrawCountLimit = info.Num
	retMsg.DrawTimeLimit = 0
	retMsg.Beans = feeTemp.TableFee
	retMsg.RoomCard = player.Currency
	retMsg.ServerIP = host
	retCode = 0
}

func (m *UserModule) SearchTableResult(args []interface{}) {
	roomInfo := args[0].(*msg.RoomInfo)
	player := m.a.UserData().(*user.User)
	retMsg := &msg.L2C_SearchResult{}
	retcode := 0
	defer func() {
		if retcode != 0 {
			if roomInfo.CreateUserId == player.Id {
				//todo  delte room ???
			}
			match_room.ChanRPC.Go("delMatchPlayer", player.Id, roomInfo)
			player.WriteMsg(RenderErrorMessage(retcode))
		} else {
			player.WriteMsg(retMsg)
		}
	}()

	template, ok := base.GameServiceOptionCache.Get(roomInfo.KindID, roomInfo.ServerID)
	if !ok {
		retcode = ConfigError
		return
	}

	feeTemp, ok1 := base.PersonalTableFeeCache.Get(roomInfo.KindID, roomInfo.ServerID, roomInfo.PayCnt)
	if !ok1 {
		log.Error("not foud PersonalTableFeeCache kindId:%d, serverID:%d, payCnt:%d", roomInfo.KindID, roomInfo.ServerID, roomInfo.PayCnt)
		retcode = NoFoudTemplate
		return
	}

	host := game_list.GetSvrByNodeID(roomInfo.NodeID)
	if host == "" {
		retcode = ErrNotFoudServer
		return
	}

	//扣除费用
	if !player.CheckFree() {
		money := feeTemp.AATableFee
		//全付的费用在创建房间时扣除
		if money > 0 && roomInfo.PayType == AA_PAY_TYPE {
			if !player.SubCurrency(money, roomInfo.PayType) {
				retcode = NotEnoughFee
				return
			}
			if !player.HasRecord(roomInfo.RoomID) {
				record := &model.TokenRecord{}
				record.UserId = player.Id
				record.RoomId = roomInfo.RoomID
				record.Amount = money
				record.TokenType = roomInfo.PayType
				record.KindID = template.KindID
				record.PlayCnt = roomInfo.PayCnt
				if !player.AddRecord(record) {
					retcode = ErrServerError
					player.AddCurrency(money)
					return
				}
			}
		}
	}

	player.KindID = roomInfo.KindID
	player.ServerID = roomInfo.ServerID
	player.Roomid = roomInfo.RoomID
	player.GameNodeID = roomInfo.NodeID
	player.EnterIP = host

	model.GamescorelockerOp.UpdateWithMap(player.Id, map[string]interface{}{
		"KindID":     player.KindID,
		"ServerID":   player.ServerID,
		"GameNodeID": roomInfo.NodeID,
		"EnterIP":    host,
		"roomid":     roomInfo.RoomID,
	})

	retMsg.TableID = roomInfo.RoomID
	retMsg.ServerIP = host
	retMsg.KindID = roomInfo.KindID
	return
}

//获取自己创建的房间
func (m *UserModule) GetCreatorRecord(args []interface{}) {
	//recvMsg := args[0].(*msg.C2L_ReqCreatorRoomRecord)
	retMsg := &msg.L2C_CreatorRoomRecord{}
	u := m.a.UserData().(*user.User)
	retMsg.Records = u.GetRoomInfo()
	var ids []int
	for _, v := range retMsg.Records {
		ids = append(ids, v.RoomID)
	}

	//更新状态
	if len(ids) > 0 {
		ret, _ := game_list.ChanRPC.Call1("GetRoomsStatus", ids)
		m := ret.(map[int]int)
		for _, v := range retMsg.Records {
			status, ok := m[v.RoomID]
			if ok {
				v.Status = status
			}
		}
	}
	u.WriteMsg(retMsg)
}

//获取某个房间内的玩家信息
func (m *UserModule) GetRoomPlayerBreif(args []interface{}) {
	recvMsg := args[0].(*msg.C2L_ReqRoomPlayerBrief)
	u := m.a.UserData().(*user.User)
	r := u.GetRoom(recvMsg.RoomId)
	if r == nil {
		u.WriteMsg(&msg.L2C_RoomPlayerBrief{})
	} else {
		game_list.ChanRPC.Go("SendPlayerBrief", recvMsg.RoomId, u)
	}
}

///////
func loadUser(u *user.User) bool {
	ainfo, aok := model.AccountsmemberOp.Get(u.Id)
	if !aok {
		log.Error("at loadUser not foud AccountsmemberOp by user", u.Id)
		return false
	}

	log.Debug("load user : == %v", ainfo)
	u.Accountsmember = ainfo

	glInfo, glok := model.GamescorelockerOp.Get(u.Id)
	if !glok {
		log.Error("at loadUser not foud GamescoreinfoOp by user  %d", u.Id)
		//return false
		glInfo = &model.Gamescorelocker{
			UserID: u.Id,
		}
		model.GamescorelockerOp.Insert(glInfo)
	}
	u.Gamescorelocker = glInfo
	if u.Gamescorelocker.EnterIP != "" {
		log.Debug("check room ...............  %d ", u.Roomid)
		_, have := game_list.ChanRPC.Call1("HaseRoom", u.Roomid)
		if have != nil {
			log.Debug("check room  login room is close ...............  ")
			u.DelGameLockInfo()
		}
	}

	giInfom, giok := model.GamescoreinfoOp.Get(u.Id)
	if !giok {
		log.Error("at loadUser not foud GamescoreinfoOp by user  %d", u.Id)
		return false
	}
	u.Gamescoreinfo = giInfom

	ucInfo, uok := model.UserattrOp.Get(u.Id)
	if !uok {
		log.Error("at loadUser not foud UserroomcardOp by user  %d", u.Id)
		return false
	}
	u.Userattr = ucInfo

	uextInfo, ueok := model.UserextrainfoOp.Get(u.Id)
	if !ueok {
		log.Error("at loadUser not foud UserextrainfoOp by user  %d", u.Id)
		return false
	}
	u.Userextrainfo = uextInfo

	userToken, tok := model.UsertokenOp.Get(u.Id)
	if !tok {
		log.Error("at loadUser not foud UsertokenOp by user  %d", u.Id)
		return false
	}
	u.Usertoken = userToken

	rooms, err := model.CreateRoomInfoOp.QueryByMap(map[string]interface{}{
		"user_id": u.Id,
	})
	if err != nil {
		log.Error("at loadUser not foud CreateRoomInfoOp by user  %d", u.Id)
		return false
	}
	for _, v := range rooms {
		log.Debug("load creator room info ok ... %v", v)
		u.Rooms[v.RoomId] = v
	}

	tokenRecords, terr := model.TokenRecordOp.QueryByMap(map[string]interface{}{
		"user_id": u.Id,
	})
	if terr != nil {
		log.Error("at loadUser not foud CreateRoomInfoOp by user  %d", u.Id)
		return false
	}

	//加载扣钱记录
	now := time.Now().Unix()
	for _, v := range tokenRecords {
		temp, ok := base.GameServiceOptionCache.Get(v.KindID, v.ServerId)
		if ok {
			if v.CreatorTime.Unix()+int64(temp.TimeNotBeginGame+30) < now && v.Status == 0 { //没开始返回钱
				if u.AddCurrency(v.Amount) {
					model.TokenRecordOp.Delete(v.RoomId, v.UserId)
				}
			}

			if v.CreatorTime.Unix()+86400 < now { // 一天了还没删除？？？ 在这里删除。安全处理
				model.TokenRecordOp.Delete(v.RoomId, v.UserId)
			}
		}
		u.Records[v.RoomId] = v
	}

	return true
}

func createUser(UserID int64, accountData *account.Accountsinfo) (*user.User, bool) {
	U := user.NewUser(UserID)
	U.Accountsmember = &model.Accountsmember{
		UserID: UserID,
	}
	_, err := model.AccountsmemberOp.Insert(U.Accountsmember)
	if err != nil {
		log.Error("at createUser insert Accountsmember error")
		return nil, false
	}

	now := time.Now()
	U.Gamescoreinfo = &model.Gamescoreinfo{
		UserID:        UserID,
		LastLogonDate: &now,
	}
	_, err = model.GamescoreinfoOp.Insert(U.Gamescoreinfo)
	if err != nil {
		log.Error("at createUser insert Gamescoreinfo error")
		return nil, false
	}

	U.Userattr = &model.Userattr{
		UserID:     UserID,
		NickName:   accountData.NickName,
		Gender:     accountData.Gender,
		HeadImgUrl: accountData.HeadImgUrl,
	}
	_, err = model.UserattrOp.Insert(U.Userattr)
	if err != nil {
		log.Error("at createUser insert Userroomcard error")
		return nil, false
	}

	U.Userextrainfo = &model.Userextrainfo{
		UserId: UserID,
	}
	_, err = model.UserextrainfoOp.Insert(U.Userextrainfo)
	if err != nil {
		log.Error("at createUser insert Userroomcard error")
		return nil, false
	}

	U.Usertoken = &model.Usertoken{
		UserID: UserID,
	}

	if conf.Test {
		U.Usertoken.Currency = 9999999
		U.Usertoken.RoomCard = 1000000
	}

	U.Gamescorelocker = &model.Gamescorelocker{
		UserID: UserID,
	}
	_, err = model.GamescorelockerOp.Insert(U.Gamescorelocker)
	if err != nil {
		log.Error("at createUser insert Gamescorelocker error")
		return nil, false
	}

	_, err = model.UsertokenOp.Insert(U.Usertoken)
	if err != nil {
		log.Error("at createUser insert Userroomcard error")
		return nil, false
	}

	return U, true
}

func BuildClientMsg(retMsg *msg.L2C_LogonSuccess, user *user.User, acinfo *account.Accountsinfo) {
	retMsg.FaceID = user.FaceID //头像标识
	retMsg.Gender = user.Gender
	retMsg.UserID = user.Id
	retMsg.Spreader = acinfo.SpreaderID
	retMsg.Experience = user.Experience
	retMsg.LoveLiness = user.LoveLiness
	retMsg.NickName = user.NickName
	//用户成绩
	//retMsg.UserScore = user.Score
	retMsg.UserInsure = user.InsureScore
	retMsg.Medal = user.UserMedal
	retMsg.UnderWrite = user.UnderWrite
	retMsg.WinCount = user.WinCount
	retMsg.LostCount = user.LostCount
	retMsg.DrawCount = user.DrawCount
	retMsg.FleeCount = user.FleeCount
	log.Debug("node id === %v", conf.Server.NodeId)
	retMsg.HallNodeID = conf.Server.NodeId
	retMsg.RegisterDate = acinfo.RegisterDate.Unix()
	//额外信息
	retMsg.MbTicket = user.MbTicket
	retMsg.MbPayTotal = user.MbPayTotal
	retMsg.MbVipLevel = user.MbVipLevel
	retMsg.PayMbVipUpgrade = user.PayMbVipUpgrade
	//约战房相关
	retMsg.UserScore = user.Currency
	retMsg.ServerID = user.ServerID
	retMsg.KindID = user.KindID
	retMsg.ServerIP = user.EnterIP
}

func (m *UserModule) matchResult(args []interface{}) {
	ret := args[0].(bool)

	if ret {
		r := args[1].(*msg.RoomInfo)
		m.SearchTableResult([]interface{}{r})
	} else {
		retMsg := &msg.L2C_SearchResult{}
		retMsg.TableID = INVALID_TABLE
		u := m.a.UserData().(*user.User)
		u.WriteMsg(retMsg)
	}

}

//玩家离开游戏服房间
func (m *UserModule) leaveRoom(args []interface{}) {
	log.Debug("at leaveRoom ================= ")
	rmsg := args[0].(*msg.LeaveRoom)
	player := m.a.UserData().(*user.User)
	log.Debug("at hall server leaveRoom uid:%v", player.Id)

	//全付方式这边不返还
	if rmsg.PayType != SELF_PAY_TYPE {
		record := player.GetRecord(rmsg.RoomId)
		if record != nil {
			player.DelRecord(record.RoomId)
			//没开始就离开房间
			if rmsg.Status == RoomStatusReady {
				player.AddCurrency(record.Amount)
			}
		} else {
			log.Error("at restoreToken not foud record uid:%d", player.Id)
		}
	}
}

//玩家进入了游戏房间
func (m *UserModule) joinRoom(args []interface{}) {
	log.Debug("at joinRoom ================= ")
	rmsg := args[0].(*msg.JoinRoom)
	u := m.a.UserData().(*user.User)
	log.Debug("at hall server joinRoom uid:%v", u.Id)
	u.KindID = rmsg.Rinfo.KindID
	u.ServerID = rmsg.Rinfo.ServerID
	u.GameNodeID = rmsg.Rinfo.NodeID
	u.EnterIP = rmsg.Rinfo.SvrHost
}

//进入房间失败
func (m *UserModule) JoinRoomFaild(args []interface{}) {
	log.Debug("at JoinRoomFaild ================= ")
	rmsg := args[0].(*msg.JoinRoomFaild)
	player := m.a.UserData().(*user.User)
	record := player.GetRecord(rmsg.RoomID)
	if record != nil {
		player.DelRecord(rmsg.RoomID)
		player.AddCurrency(record.Amount)
	} else {
		log.Error("at JoinRoomFaild not foudn record")
	}
}

//游戏开始了
func (m *UserModule) GameStart(args []interface{}) {
	log.Debug("at GameStart ================= ")
	rmsg := args[0].(*msg.StartRoom)
	player := m.a.UserData().(*user.User)
	record := player.GetRecord(rmsg.RoomId)
	if record != nil {
		record.Status = 1
		model.TokenRecordOp.UpdateWithMap(record.RoomId, record.UserId, map[string]interface{}{
			"status": record.Status,
		})
	}
}

//房间结束了
func (m *UserModule) RoomEndInfo(args []interface{}) {
	info := args[0].(*msg.RoomEndInfo)
	player := m.a.UserData().(*user.User)
	log.Debug("at RoomEndInfo ================= roomid=%d, creator=%d", info.RoomId, player.Id)
	if info.Status == RoomStatusReady { //没开始就结束
		record := player.GetRecord(info.RoomId)
		if record != nil { //还原扣的钱
			err := player.DelRecord(record.RoomId)
			if err == nil {
				player.AddCurrency(record.Amount)
			} else {
				log.Error("at restoreToken not DelRecord error uid:%d", player.Id)
			}

		} else {
			log.Error("at restoreToken not foud record uid:%d", player.Id)
		}
	}
	player.DelRooms(info.RoomId)
	player.DelGameLockInfo()
	return
}

//客户端发来的充值
func (m *UserModule) RechargeById(OrderId int64) {
	player := m.a.UserData().(*user.User)
	retMsg := &msg.L2C_RechangerOk{}
	defer func() {
		player.WriteMsg(retMsg)
	}()

	order, err := account.OnlineorderOp.GetByMap(map[string]interface{}{
		"order_id": OrderId,
	})

	if err != nil {
		retMsg.Code = ErrNotFoudOrder
		return
	}

	if order.OrderStatus != 0 {
		retMsg.Code = ErrNotPay
		return
	}

	goods, ok := base.GoodsCache.Get(order.GoodsId)
	if !ok {
		retMsg.Code = ErrNotFoudTemlate
		return
	}

	qerr := account.OnlineorderOp.UpdateWithMap(order.OnLineId, map[string]interface{}{
		"order_status": 2,
	})

	if qerr != nil {
		retMsg.Code = ErrNotUpdateOrderFaild
		return
	}

	if !player.HasTimes(common.ActivityRechangeDay) {
		player.SetTimes(common.ActivityRechangeDay, 0)
	}
	log.Debug("11111111111111111111 ")
	player.AddCurrency(goods.Diamond)
	retMsg.Code = goods.Diamond
}

//登录的时候的充值
func (m *UserModule) Recharge(args []interface{}) {
	player := m.a.UserData().(*user.User)
	orders, err := account.OnlineorderOp.QueryByMap(map[string]interface{}{
		"user_id":      player.Id,
		"order_status": 0,
	})
	if err != nil {
		log.Debug("at Recharge load orders error :%s", err.Error())
		return
	}

	var qerr error
	var code int
	var goods *base.Goods
	var ok bool
	for _, v := range orders {
		code = 0
		for {
			if v.OrderStatus != 0 {
				log.Error("at rechang OrderStatus != 0")
				code = 2
				break
			}

			goods, ok = base.GoodsCache.Get(v.GoodsId)
			if !ok {
				log.Error("at Recharge error :GoodsId%d", v.GoodsId)
				code = 3
				break
			}

			qerr = account.OnlineorderOp.UpdateWithMap(v.OnLineId, map[string]interface{}{
				"order_status": 2,
			})

			if qerr != nil {
				log.Error("at Recharge error :%s", qerr.Error())
				code = 4
				break
			}
			break
		}

		if code == 0 {
			if !player.HasTimes(common.ActivityRechangeDay) {
				player.SetTimes(common.ActivityRechangeDay, 0)
			}
			log.Debug("11111111111111111111 ")
			player.AddCurrency(goods.Diamond)
			player.WriteMsg(&msg.L2C_RechangerOk{Code: code, Gold: goods.Diamond})
			recharge := datalog.RechargeLog{}
			recharge.AddRechargeLogInfo(v.OnLineId, v.PayAmount, v.UserId, v.PayType, v.GoodsId)
		}
	}
}

//离线通知事件
func (m *UserModule) HandlerOffilneEvent(args []interface{}) {
	log.Debug("at HandlerOffilneEvent .................. === %v", args)
	recvMsg := args[0].(*msg.S2S_OfflineHandler)
	player := m.a.UserData().(*user.User)
	h, ok := model.UserOfflineHandlerOp.Get(recvMsg.EventID)
	if ok {
		handlerEventFunc(player, h)
	}
}

func (m *UserModule) KickOutUser(player *user.User) {
	player.ChanRPC().Go("ForceClose")
}

func (m *UserModule) ForceClose(args []interface{}) {
	log.Debug("at ForceClose ..... ")
	m.Close(KickOutMsg)
}

func (m *UserModule) SvrShutdown(args []interface{}) {
	log.Debug("at SvrShutdown ..... ")
	m.Close(ServerKick)
}

//重登的时候删除已经不存在的房间， 后期这些房间放在redis
func (m *UserModule) DeleteVaildIds(args []interface{}) {
	log.Debug("at DeleteVaildIds ................ ")
	ids := args[0].([]int)
	player := m.a.UserData().(*user.User)
	for _, id := range ids {
		log.Debug("at login DeleteVaildIds uid:%d, roomid : %d", player.Id, id)
		player.DelRooms(id)
		player.DelRecord(id)
	}
}

//删除自己创建的房间
func (m *UserModule) DeleteRoom(args []interface{}) {
	recvMsg := args[0].(*msg.C2L_DeleteRoom)
	player := m.a.UserData().(*user.User)

	info := player.GetRoom(recvMsg.RoomId)
	if info == nil {
		player.WriteMsg(&msg.L2C_DeleteRoomResult{Code: ErrNotFondCreatorRoom})
		return
	}

	cluster.AsynCallGame(info.NodeId, m.Skeleton.GetChanAsynRet(), &msg.S2S_CloseRoom{RoomID: recvMsg.RoomId}, func(data interface{}, err error) {
		if err != nil {
			player.WriteMsg(&msg.L2C_DeleteRoomResult{Code: ErrRoomIsStart})
		} else {
			player.DelRooms(recvMsg.RoomId)
			player.WriteMsg(&msg.L2C_DeleteRoomResult{})
		}
	})

}

//绑定电话号码
func (m *UserModule) SetPhoneNumber(args []interface{}) {
	recvMsg := args[0].(*msg.C2L_SetPhoneNumber)
	retMsg := &msg.L2C_SetPhoneNumberRsp{}
	player := m.a.UserData().(*user.User)

	defer func() {
		player.WriteMsg(retMsg)
	}()

	info, ok := model.UserMaskCodeOp.Get(player.Id)
	if !ok {
		retMsg.Code = ErrMaskCodeNotFoud
		return
	}

	if info.MaskCode != recvMsg.MaskCode {
		retMsg.Code = ErrMaskCodeError
		return
	}

	if !player.HasTimes(common.ActivityBindPhome) {
		player.SetTimes(common.ActivityBindPhome, 0)
	}

	model.UserMaskCodeOp.Delete(player.Id)
	player.PhomeNumber = info.PhomeNumber
	model.UserattrOp.UpdateWithMap(player.Id, map[string]interface{}{
		"phome_number": info.PhomeNumber,
	})
	retMsg.PhoneNumber = info.PhomeNumber
}

//点赞
func (m *UserModule) DianZhan(args []interface{}) {
	recvMsg := args[0].(*msg.C2L_DianZhan)
	player := m.a.UserData().(*user.User)
	if AddOfflineHandler(OfflineTypeDianZhan, recvMsg.UserID, nil, true) {
		player.WriteMsg(&msg.L2C_DianZhanRsp{})
	} else {
		player.WriteMsg(&msg.L2C_DianZhanRsp{Code: 1})
	}

}

//续费
func (m *UserModule) RenewalFees(args []interface{}) {
	//recvMsg := args[0].(*msg.C2L_RenewalFees)
	player := m.a.UserData().(*user.User)
	retCode := 0
	defer func() {
		if retCode != 0 {
			player.WriteMsg(&msg.L2C_RenewalFeesRsp{Code: retCode, UserID: player.Id})
		}
	}()
	if player.Roomid == 0 {
		retCode = ErrNotInRoom
		return
	}

	info, err := game_list.ChanRPC.TimeOutCall1("GetRoomByRoomId", 5*time.Second, player.Roomid)
	if err != nil {
		retCode = ErrFindRoomError
		return
	}

	room := info.(*msg.RoomInfo)
	feeTemp, ok := base.PersonalTableFeeCache.Get(room.KindID, room.ServerID, room.PayCnt)
	if !ok {
		retCode = ErrConfigError
		return
	}

	money := feeTemp.TableFee
	if room.PayType == AA_PAY_TYPE {
		money = feeTemp.AATableFee
	}

	if !player.SubCurrency(money, room.PayType) {
		retCode = NotEnoughFee
		return
	}
	record := player.GetRecord(room.RoomID)
	if record == nil {
		log.Error("at RenewalFees not foud old TokenRecord ")
		record = &model.TokenRecord{}
		record.UserId = player.Id
		record.RoomId = room.RoomID
		record.Amount = money
		record.TokenType = room.PayType
		record.KindID = room.KindID
		record.PlayCnt = room.PayCnt
		if !player.AddRecord(record) {
			retCode = ErrServerError
			player.AddCurrency(money)
			return
		}
	} else {
		record.PlayCnt += feeTemp.DrawCountLimit
		if record.Status != 1 {
			log.Error("not foud status ")
		}
		err := model.TokenRecordOp.UpdateWithMap(record.RoomId, record.UserId, map[string]interface{}{
			"play_cnt": record.PlayCnt,
		})
		if err != nil {
			retCode = ErrRenewalFeesFaild
			return
		}
	}

	cluster.SendMsgToGame(room.NodeID, &msg.S2S_RenewalFee{RoomID: room.RoomID, AddCnt: feeTemp.DrawCountLimit,
		HallNodeID: conf.Server.NodeId, UserId: player.UserId})
}

//续费结果
func (m *UserModule) RenewalFeeResult(args []interface{}) {
	log.Debug("=============at RenewalFeeResult")
	recvMsg := args[0].(*msg.S2S_RenewalFeeResult)
	player := m.a.UserData().(*user.User)

	//续费失败返还钱
	if recvMsg.ResultId != 0 {
		record := player.GetRecord(recvMsg.RoomId)
		if record != nil {
			player.AddCurrency(record.Amount)
			player.DelRecord(record.RoomId)
		}
		//通知客户端玩家续费失败
		retCode := 0
		switch recvMsg.ResultId {
		case 1, 2:
			retCode = ErrFindRoomError
		case 3:
			retCode = ErrRenewalFeeRepeat
		}
		player.WriteMsg(&msg.L2C_RenewalFeesRsp{Code: retCode, UserID: player.Id})
	} else {
		//成功了
		info, err := game_list.ChanRPC.TimeOutCall1("GetRoomByRoomId", 5*time.Second, recvMsg.RoomId)
		if err != nil {
			log.Error("RenewalFeeResult GetRoomByRoomId fail, RoomId=%d", recvMsg.RoomId)
			return
		}
		room := info.(*msg.RoomInfo)
		room.CurPayCnt = 0 //已玩局数重置
		//room.PayCnt += recvMsg.AddCount
		room.RenewalCnt++
	}
}

//改名字
func (m *UserModule) ChangeUserName(args []interface{}) {
	recvMsg := args[0].(*msg.C2L_ChangeUserName)
	player := m.a.UserData().(*user.User)
	player.NickName = recvMsg.NewName

	model.UserattrOp.UpdateWithMap(player.Id, map[string]interface{}{
		"NickName": player.NickName,
	})

	player.WriteMsg(&msg.L2C_ChangeUserNameRsp{Code: 0, NewName: player.NickName})
}

//改签名
func (m *UserModule) ChangeSign(args []interface{}) {
	recvMsg := args[0].(*msg.C2L_ChangeSign)
	player := m.a.UserData().(*user.User)

	player.Sign = recvMsg.Sign
	model.UserattrOp.UpdateWithMap(player.Id, map[string]interface{}{
		"Sign": player.Sign,
	})

	player.WriteMsg(&msg.L2C_ChangeSignRsp{Code: 0, NewSign: player.Sign})
}

//获取验证码
func (m *UserModule) ReqBindMaskCode(args []interface{}) {
	recvMsg := args[0].(*msg.C2L_ReqBindMaskCode)
	player := m.a.UserData().(*user.User)
	retCode := 0
	defer func() {
		player.WriteMsg(&msg.L2C_ReqBindMaskCodeRsp{Code: retCode})
	}()

	now := time.Now().Unix()
	if player.MacKCodeTime != 0 {
		if player.MacKCodeTime > now {
			retCode = ErrFrequentAccess
			return
		}
	}
	code, _ := utils.RandInt(100000, 1000000)

	player.MacKCodeTime = now + 60

	err := model.UserMaskCodeOp.InsertUpdate(&model.UserMaskCode{UserId: player.Id, PhomeNumber: recvMsg.PhoneNumber, MaskCode: code}, map[string]interface{}{
		"mask_code":    code,
		"phome_number": recvMsg.PhoneNumber,
	})
	if err != nil {
		log.Error("InsertUpdate error %s, ", err.Error())
		retCode = ErrRandMaskCodeError
		return
	}

	ret := VerifyCode(recvMsg.PhoneNumber, code)
	if ret != 0 {
		retCode = ErrFrequentAccess
		return
	}
}

func (m *UserModule) RechangerOk(args []interface{}) {
	recvMsg := args[0].(*msg.C2L_RechangerOk)
	m.RechargeById(recvMsg.OrderId)
}

func (m *UserModule) GetRoomRecord(args []interface{}) {
	recvMsg := args[0].(*msg.C2L_GetRoomRecord)
	retMsg := &msg.L2C_RoomRecord{}
	player := m.a.UserData().(*user.User)
	info, ok := model.RoomRecordOp.Get(recvMsg.RecordID)
	if ok {
		retMsg.Start = info.StartInfo
		retMsg.Playing = info.PlayInfo
		retMsg.End = info.EndInfo
	}
	player.WriteMsg(retMsg)
}

func (m *UserModule) GetUserRecord(args []interface{}) {
	recvMsg := args[0].(*msg.C2L_GetUserRecords)
	retMsg := &msg.L2C_GetUserRecords{}
	player := m.a.UserData().(*user.User)
	infos, _ := model.UserRoomRecordOp.QueryByMap(map[string]interface{}{
		"user_id": recvMsg.UserID,
	})

	for _, v := range infos {
		rcd := &msg.UserRoomRecord{}
		rcd.KindId = v.KindId
		rcd.RecordId = v.RecordId
		rcd.StartTime = v.CreateTime.Unix()
		retMsg.Data = append(retMsg.Data, rcd)
	}

	player.WriteMsg(retMsg)
}

func (m *UserModule) AddRoomRecord(args []interface{}) {
	log.Debug("at AddRoomRecord ................ ")
	roomInfo := args[0].(*model.CreateRoomInfo)
	player := m.a.UserData().(*user.User)
	player.ChangeRoomInfo(roomInfo)
}

func (m *UserModule) DelRoomRecord(args []interface{}) {
	log.Debug("at DelRoomRecord ................ ")
	recvMsg := args[0].(*msg.DelRoomRecord)
	player := m.a.UserData().(*user.User)
	player.DelRooms(recvMsg.RoomId)
}
