package internal

import (
	. "mj/common/cost"
	"mj/common/msg"
	"mj/gameServer/conf"
	"mj/gameServer/db/model/base"
	"mj/gameServer/user"
	"reflect"

	"github.com/lovelly/leaf/chanrpc"
	"github.com/lovelly/leaf/cluster"
	"github.com/lovelly/leaf/gate"
	"github.com/lovelly/leaf/log"
)

////注册rpc 消息
func handleRpc(id interface{}, f interface{}) {
	cluster.SetRoute(id, ChanRPC)
	ChanRPC.Register(id, f)
}

//注册 客户端消息调用
func handlerC2S(m interface{}, h interface{}) {
	msg.Processor.SetRouter(m, ChanRPC)
	skeleton.RegisterChanRPC(reflect.TypeOf(m), h)
}

func init() {
	handleRpc("createRoom", createRoom)
	handleRpc("addRoomMember", addRoomMember)
	handleRpc("delRoomMember", delRoomMember)

	handlerC2S(&msg.C2G_GameChart_ToAll{},SendChatMsgToAll)
}

//发送给房间所有人
func SendChatMsgToAll(args []interface{})  {
	agent := args[1].(gate.Agent)
	user := agent.UserData().(*user.User)
	getData:=args[0].(*msg.C2G_GameChart_ToAll)

	var sendData msg.G2C_GameChart_ToAll
	sendData.ChatColor = getData.ChatColor
	sendData.SendUserID=user.Id
	sendData.TargetUserID=getData.SendUserID
	sendData.ChatString=getData.ChatString

	SendMsgToAll(user.ChatRoomId, sendData)
}

//发送给房间某人
func sendCharMsgToUser(args []interface{}){

}