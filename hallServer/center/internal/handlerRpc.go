package internal

import (
	"mj/hallServer/conf"
	"github.com/lovelly/leaf/cluster"
	"gopkg.in/mgo.v2/bson"
	"github.com/lovelly/leaf/gate"
	"github.com/lovelly/leaf/log"
)

var (
	clientCount     = 0
	accountAgentMap = map[bson.ObjectId]gate.Agent{}
	userAgentMap    = map[bson.ObjectId]gate.Agent{}
)

func handleRpc(id interface{}, f interface{}) {
	cluster.SetRoute(id, ChanRPC)
	skeleton.RegisterChanRPC(id, f)
}

func init() {
	skeleton.RegisterChanRPC("KickAccount", KickAccount)
	skeleton.RegisterChanRPC("AccountOnline", AccountOnline)
	skeleton.RegisterChanRPC("AccountOffline", AccountOffline)
	skeleton.RegisterChanRPC("UserOnline", UserOnline)
	skeleton.RegisterChanRPC("UserOffline", UserOffline)

	handleRpc("GetFrontInfo", GetFrontInfo)
	handleRpc("AddClusterClient", AddClusterClient)
	handleRpc("RemoveClusterClient", RemoveClusterClient)
}

func KickAccount(args []interface{}) {
	accountId := args[0].(bson.ObjectId)
	oldAgent, ok := accountAgentMap[accountId]
	if ok {
		oldAgent.Destroy()
	}
}

func AccountOnline(args []interface{}) (interface{}, error) {
	accountId := args[0].(bson.ObjectId)
	agent := args[1].(gate.Agent)
	if oldAgent, ok := accountAgentMap[accountId]; ok {
		oldAgent.Destroy()
		return false, nil
	} else {
		accountAgentMap[accountId] = agent

		clientCount += 1
		cluster.Go("world", "UpdateFrontInfo", conf.ServerName(), clientCount)

		log.Debug("%v account is online", accountId)
		return true, nil
	}
}

func AccountOffline(args []interface{}) {
	accountId := args[0].(bson.ObjectId)
	agent := args[1].(gate.Agent)
	oldAgent, ok := accountAgentMap[accountId]
	if ok && agent == oldAgent {
		delete(accountAgentMap, accountId)

		clientCount -= 1
		cluster.Go("world", "UpdateFrontInfo", conf.ServerName(), clientCount)

		log.Debug("%v account is offline", accountId)
	}
}

func UserOnline(args []interface{}) {
	userId := args[0].(bson.ObjectId)
	agent := args[1].(gate.Agent)
	userAgentMap[userId] = agent
	log.Debug("%v user is online", userId)
}

func UserOffline(args []interface{}) {
	userId := args[0].(bson.ObjectId)
	agent := args[1].(gate.Agent)
	oldAgent, ok := userAgentMap[userId]
	if ok && agent == oldAgent {
		delete(userAgentMap, userId)
		log.Debug("%v user is offline", userId)
	}
}

func GetFrontInfo(args []interface{}) ([]interface{}, error) {
	return []interface{}{clientCount, conf.Server.MaxConnNum, conf.Server.TCPAddr}, nil
}

func AddClusterClient(args []interface{}) {
	serverInfoMap := args[0].(map[string]string)
	for serverName, addr := range serverInfoMap {
		cluster.AddClient(serverName, addr)
	}
}

func RemoveClusterClient(args []interface{}) {
	serverName := args[0].(string)
	cluster.RemoveClient(serverName)
}
