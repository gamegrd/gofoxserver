package msg

import (
	//"gopkg.in/mgo.v2/bson"
	"github.com/lovelly/leaf/network/json"
)

var (
	Processor = json.NewProcessor()
)

func init() {
	Processor.Register(&C2L_Login{})
	Processor.Register(&ShowErrCode{})
	Processor.Register(&C2L_Regist{})
	Processor.Register(&CMD_GP_LogonFailure{})
	Processor.Register(&CMD_MB_LogonSuccess{})
	Processor.Register(&C2G_CreateRoom{})
	Processor.Register(&C2G_HZOutCard{})
	Processor.Register(&L2C_ServerList{})

}

type ShowErrCode struct {
	ErrCode int
}

/////////////////////// login server msg ////////////////////////////
//登录消息
type C2L_Login struct {
	ModuleID  int
	PlazaVersion int
	DeviceType  int
	LogonPass string
	Accounts string
	MachineID string
	MobilePhone string
}

//注册消息
type C2L_Regist struct {
	ModuleID int//模块标识
	PlazaVersion int//广场版本
	DeviceType string //设备类型
	//密码变量
	LogonPass string //登录密码
	InsurePass string //银行密码

	//注册信息
	FaceID int8 //头像标识
	Gender int8 //用户性别
	Accounts string //登录帐号
	NickName string//用户昵称

	//连接信息
	MachineID string //机器标识
	MobilePhone string //电话号码  //默认不获取本机号码
}

//登录失败
type  CMD_GP_LogonFailure struct{
	ResultCode int						//错误代码
	DescribeString string				//描述消息
};


//登录成功
type CMD_MB_LogonSuccess struct {
	FaceID int8		  	`json:"wFaceID"` 					//头像标识
	Gender int8			`json:"cbGender"`				//用户性别
	UserID int			`json:"dwUserID"`			//用户 I D
	Spreader int		`json:"szSpreader"`				//推荐人用户标识
	GameID int			`json:"dwGameID"`			//游戏 I D
	Experience int		`json:"dwExperience"`			//经验数值
	LoveLiness int		`json:"dwLoveLiness"`				//用户魅力
	NickName string		`json:"szNickName"`		//用户昵称

	//用户成绩
	UserScore int64		`json:"lUserScore"`			//用户欢乐豆
	UserInsure int64			`json:"lUserInsure"`		//用户银行
	Medal  int				`json:"dwMedal"`			//用户钻石
	UnderWrite string		`json:"szUnderWrite"`	//个性签名
	WinCount  int			`json:"dwWinCount"`			//赢局数
	LostCount int			`json:"dwLostCount"`			//输局数
	DrawCount int			`json:"dwDrawCount"`			//和局数
	FleeCount int				`json:"dwFleeCount"`		//跑局数
	RegisterDate *DateTime		`json:"RegisterDate"`				//注册时间

	//额外信息
	MbTicket int				`json:"dwMbTicket"`			//手机兑换券数量
	MbPayTotal int				`json:"dwMbPayTotal"`		//手机充值累计
	MbVipLevel int				`json:"dwMbVipLevel"`		//手机VIP等级
	PayMbVipUpgrade int			`json:"dwPayMbVipUpgrade"`		//手机VIP升级，所需充值数（vip最高级时该值为0）

	//约战房相关
 	RoomCard int				`json:"lRoomCard"`		//用户房卡
	LockServerID int			`json:"dwLockServerID"`		//锁定房间
	KindID int					`json:"dwKindID"`		//游戏类型
};

type L2C_ServerList []*TagGameServer

////////////// game sverver proto
//创建房间
type C2G_CreateRoom struct {

}

// 出牌
type C2G_HZOutCard struct {

}



