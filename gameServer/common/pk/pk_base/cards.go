package pk_base

import "github.com/lovelly/leaf/util"

const (
	IDX_TBNN = 0 // 通比牛牛牌型
	IDX_DDZ  = 1 //斗地主配置索引
	IDX_SSS  = 2 //十三水配置索引
)

var cards = [][]int{
	0: []int{ // 通比牛牛牌型
		0x01, 0x02, 0x03, 0x04, 0x05, 0x06, 0x07, 0x08, 0x09, 0x0A, 0x0B, 0x0C, 0x0D, //方块 A - K
		0x11, 0x12, 0x13, 0x14, 0x15, 0x16, 0x17, 0x18, 0x19, 0x1A, 0x1B, 0x1C, 0x1D, //梅花 A - K
		0x21, 0x22, 0x23, 0x24, 0x25, 0x26, 0x27, 0x28, 0x29, 0x2A, 0x2B, 0x2C, 0x2D, //红桃 A - K
		0x31, 0x32, 0x33, 0x34, 0x35, 0x36, 0x37, 0x38, 0x39, 0x3A, 0x3B, 0x3C, 0x3D, //黑桃 A - K 3 13 54
	},
}

func getCardByIdx(idx int) []int {
	card := make([]int, len(cards[idx]))
	oldcard := cards[idx]
	util.DeepCopy(&card, &oldcard)
	return card
}

type PK_CFG struct {
	PublicCardCount int //
	MaxCount        int //最大手牌数目
	MaxRepertory    int //最多存放多少张牌
}

var cfg = []*PK_CFG{
	IDX_TBNN: &PK_CFG{
		PublicCardCount: 2,
		MaxCount:        5,
		MaxRepertory:    52,
	},
}

func GetCfg(idx int) *PK_CFG {
	return cfg[idx]
}
