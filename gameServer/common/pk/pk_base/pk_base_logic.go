package pk_base

import (
	"github.com/lovelly/leaf/log"
	"github.com/lovelly/leaf/util"
)

type BaseLogic struct {
	ConfigIdx int //配置索引
}

func NewBaseLogic(ConfigIdx int) *BaseLogic {
	bl := new(BaseLogic)
	bl.ConfigIdx = ConfigIdx
	return bl
}

func (lg *BaseLogic) GetCfg() *PK_CFG {
	return GetCfg(lg.ConfigIdx)
}


func (lg *BaseLogic) RandCardList(cbCardBuffer, OriDataArray []int) {

	//混乱准备
	cbBufferCount := int(len(cbCardBuffer))
	cbCardDataTemp := make([]int, cbBufferCount)
	util.DeepCopy(&cbCardDataTemp, &OriDataArray)

	//混乱扑克
	var cbRandCount int
	var cbPosition int
	for {
		if cbRandCount >= cbBufferCount {
			break
		}
		cbPosition = int(util.RandInterval(0, int(cbBufferCount-cbRandCount)))
		cbCardBuffer[cbRandCount] = cbCardDataTemp[cbPosition]
		cbRandCount++
		cbCardDataTemp[cbPosition] = cbCardDataTemp[cbBufferCount-cbRandCount]
	}

	return
}

//排列扑克
func (lg *BaseLogic) SortCardList(cardData []int, cardCount int) {
	var logicValue []int
	for i := 0; i < cardCount; i++ {
		logicValue[i] = lg.GetCardValue(cardData[i])
	}
	sorted := true
	last := cardCount - 1
	for {
		sorted = true
		for i := 0; i < last; i++ {
			if (logicValue[i] < logicValue[i+1]) || (logicValue[i] == logicValue[i+1] && (cardData[i] < cardData[i+1])) {
				tempData := cardData[i]
				cardData[i] = cardData[i+1]
				cardData[i+1] = tempData
				tempData = logicValue[i]
				logicValue[i] = logicValue[i+1]
				logicValue[i+1] = tempData
				sorted = false
			}
		}
		last--
		if sorted == true {
			break
		}
	}
}

//获取数值
func (lg *BaseLogic) GetCardValue(CardData int) int {
	return CardData & LOGIC_MASK_VALUE
}

//获取花色
func (lg *BaseLogic) GetCardColor(CardData int) int {
	return CardData & LOGIC_MASK_COLOR
}



//获取牛牛牌值
func (lg *BaseLogic) GetCardLogicValue(CardData int) int {
	//扑克属性
	//CardColor = GetCardColor(CardData)
	CardValue := lg.GetCardValue(CardData)

	//转换数值
	//return (CardValue>10)?(10):CardValue
	if CardValue > 10 {
		CardValue = 10
	}
	return CardValue
}

//获取牛牛牌型
func (lg *BaseLogic) NNGetCardType(CardData []int, CardCount int) int {

	if CardCount != lg.GetCfg().MaxCount {
		return 0
	}

	////炸弹牌型
	//SameCount := 0

	Temp := make([]int, lg.GetCfg().MaxCount)
	Sum := 0
	for i := 0; i < CardCount; i++ {
		Temp[i] = lg.GetCardLogicValue(CardData[i])
		log.Debug("%d", Temp[i])
		Sum += Temp[i]
	}
	log.Debug("%d", Sum)

	//王的数量
	KingCount := 0
	TenCount := 0

	for i := 0; i < CardCount; i++ {
		if lg.GetCardValue(CardData[i]) > 10 && CardData[i] != 0x4E && CardData[i] != 0x4F {
			KingCount++
		} else if lg.GetCardValue(CardData[i]) == 10 {
			TenCount++
		}
	}

	if KingCount == lg.GetCfg().MaxCount {
		return OX_FIVEKING //五花――5张牌都是10以上（不含10）的牌。。
	}

	Value := lg.GetCardLogicValue(CardData[3])
	Value += lg.GetCardLogicValue(CardData[4])

	if Value > 10 {
		if CardData[3] == 0x4E || CardData[4] == 0x4F || CardData[4] == 0x4E || CardData[3] == 0x4F {
			Value = 10
		} else {
			Value -= 10 //2.3
		}

	}

	return Value //OX_VALUE0
}

//获取牛牛倍数
func (lg *BaseLogic) NNGetTimes(cardData []int, cardCount int, niu int) int {
	if niu != 1 {
		return 1
	}
	if cardCount != lg.GetCfg().MaxCount {
		return 0
	}
	times := lg.NNGetCardType(cardData, lg.GetCfg().MaxCount)
	log.Debug("times %d", times)

	/*if(bTimes<7)return 1;
	else if(bTimes==7)return 1;
	else if(bTimes==8)return 2;
	else if(bTimes==9)return 3;
	else if(bTimes==10)return 4;*/
	//else if(bTimes==OX_THREE_SAME)return 5;
	//else if(bTimes==OX_FOUR_SAME)return 5;
	//else if(bTimes==OX_FOURKING)return 5;
	//else if(bTimes==OX_FIVEKING)return 5;

	if times < 7 {
		return 1
	} else if times >= 7 && times <= 10 {
		return times - 6
	} else if times == OX_FIVEKING {
		return 5
	}
	return 0
}

// 获取牛牛
func (lg *BaseLogic) NNGetOxCard(cardData []int, cardCount int) bool {
	if cardCount != lg.GetCfg().MaxCount {
		return false
	}

	temp := make([]int, lg.GetCfg().MaxCount)
	sum := 0
	for i := 0; i < lg.GetCfg().MaxCount; i++ {
		temp[i] = lg.GetCardLogicValue(cardData[i])
		sum += temp[i]
	}
	//王的数量
	kingCount := 0
	tenCount := 0

	for i := 0; i < lg.GetCfg().MaxCount; i++ {
		if cardData[i] == 0x4E || cardData[i] == 0x4F {
			kingCount++
		} else if lg.GetCardValue(cardData[i]) == 10 {
			tenCount++
		}
	}
	maxNiuZi := 0
	maxNiuPos := 0
	niuTemp := make([][]int, 30,lg.GetCfg().MaxCount)
	var isKingPai [30]bool

	niuCount := 0
	haveKing := false
	//查找牛牛
	for i := 0; i < cardCount-1; i++ {
		for j := 0; j < cardCount; j++ {
			haveKing = false
			left := (sum - temp[i] - temp[j]) % 10
			if left > 0 && kingCount > 0 {
				for k := 0; k < cardCount; k++ {
					if k != i && k != j {
						if cardData[k] == 0x4E || cardData[k] == 0x4F {
							haveKing = true
						}
					}
				}
			}
			if (sum-temp[i]-temp[j])%10 == 0 || haveKing { ////如果减去2个剩下3个是10的倍数
				count := 0
				for k := 0; k < cardCount; k++ {
					if k != i && k != j {
						niuTemp[niuCount][count] = cardData[k]
						count++
					}
				}
				if count != 3 {
					log.Debug("NNGetOxCard err not 3")
					return false
				}
				niuTemp[niuCount][count] = cardData[i]
				count++
				niuTemp[niuCount][count] = cardData[j]
				count++
				value := temp[i]
				value += temp[j]
				if value > 10 {
					if cardData[i] == 0x4E || cardData[j] == 0x4F || cardData[i] == 0x4F || cardData[j] == 0x4E {
						haveKing = true
						value = 10
					} else {
						value -= 10
					}
				}
				isKingPai[niuCount] = haveKing
				if value > maxNiuZi {
					maxNiuZi = value     //最大牛数量
					maxNiuPos = niuCount //记录最大牛牌的位置
				}
				niuCount++
				continue
			}
		}
	}
	if niuCount > 0 {
		for i := 0; i < cardCount; i++ {
			cardData[i] = niuTemp[maxNiuPos][i]
		}
		return true
	}
	return false
}

// 牛牛获取整数
func (lg *BaseLogic) NNIsIntValue(cardData []int, cardCount int) bool {
	sum := 0
	for i := 0; i < cardCount; i++ {
		sum += lg.NNGetCardLogicValue(cardData[i])
	}
	if !(sum > 0) {
		return false
	}
	return (sum%10 == 0)
}

// 牛牛比牌
func (lg *BaseLogic) NNCompareCard(firstData []int, nextData []int, cardCount int, firstOX bool, nextOX bool) bool {
	if firstOX != nextOX {
		if firstOX {
			return true
		} else {
			return false
		}
	}
	if lg.NNGetCardType(firstData, cardCount) == OX_FIVEKING && lg.NNGetCardType(nextData, cardCount) != OX_FIVEKING {
		return true
	}
	if lg.NNGetCardType(firstData, cardCount) != OX_FIVEKING && lg.NNGetCardType(nextData, cardCount) == OX_FIVEKING {
		return false
	}
	//比较牛大小
	if firstOX == true {
		//获取点数
		firstType := 0
		nextType := 0

		value := lg.NNGetCardLogicValue(nextData[3])
		value += lg.NNGetCardLogicValue(nextData[4])

		firstKing := false
		nextKing := false

		firstDa := false
		nextDa := false //nextDa是判断4,5有没有利用大王的

		if value > 10 {
			if nextData[3] == 0x4E || nextData[4] == 0x4F || nextData[4] == 0x4E || nextData[3] == 0x4F {
				left := 0
				value = 0
				for i := 3; i < 5; i++ {
					value += lg.NNGetCardLogicValue(nextData[i])
				}
				left = value % 10
				if left > 0 {
					nextDa = true
				}
				value = 10
			} else {
				value -= 10
			}
		}
		nextType = value
		kingCount := 0
		for i := 0; i < 3; i++ {
			if nextData[i] == 0x4E || nextData[i] == 0x4F {
				kingCount++
			}
		}
		if kingCount > 0 {
			value = 0
			left := 0
			for i := 0; i < 3; i++ {
				value += lg.NNGetCardLogicValue(nextData[i])
			}
			left = value % 10
			if left > 10 {
				nextKing = true
			}
		}
		value = 0
		value = lg.GetCardLogicValue(firstData[3])
		value += lg.GetCardLogicValue(firstData[4])
		if value > 10 {
			if firstData[3] == 0x4E || firstData[4] == 0x4F || firstData[4] == 0x4E || firstData[3] == 0x4F {
				left := 0
				value = 0
				for i := 3; i < 5; i++ {
					value += lg.GetCardLogicValue(firstData[i])
				}
				left = value % 10
				if left > 0 {
					firstDa = true
				}
				value = 10
			} else {
				value -= 10
			}
		}
		firstType = value
		kingCount = 0
		for i := 0; i < 3; i++ {
			if firstData[i] == 0x4E || firstData[i] == 0x4F {
				kingCount++
			}
		}
		if kingCount > 0 {
			value = 0
			left := 0
			for i := 0; i < 3; i++ {
				value += lg.GetCardLogicValue(firstData[i])
			}
			left = value % 10
			if left > 0 {
				firstKing = true
			}
		}
		if firstType == nextType {
			//同点数大王>小王>...
			firstKingPoint := 10
			nextKingPoint := 10
			for i := 0; i < 5; i++ {
				if firstData[i] == 0x4E {
					firstKingPoint = 11
				} else if firstData[i] == 0x4F {
					firstKingPoint = 12
				}
				if nextData[i] == 0x4E {
					nextKingPoint = 11
				} else if nextData[i] == 0x4F {
					nextKingPoint = 12
				}
			}
			if firstKingPoint != nextKingPoint {
				return (firstKingPoint > nextKingPoint)
			}
			if firstKing || firstDa {
				return true
			} else if nextKing || nextDa {
				return false
			}
		}
		//点数判断
		if firstType != nextType {
			return (firstType > nextType)
		}
	}
	//排序大小
	var firstTemp []int
	var nextTemp []int
	util.DeepCopy(firstTemp, firstData)
	util.DeepCopy(nextTemp, nextData)
	lg.SortCardList(firstTemp, cardCount)
	lg.SortCardList(nextTemp, cardCount)
	//比较数值
	nextMaxValue := lg.GetCardValue(nextTemp[0])
	firstMaxValue := lg.GetCardValue(firstTemp[0])
	if nextMaxValue != firstMaxValue {
		return (firstMaxValue > nextMaxValue)
	}
	//比较颜色
	return (lg.GetCardColor(firstTemp[0]) > lg.GetCardColor(nextTemp[0]))

	return false
}
