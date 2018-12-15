/*
 * 摇一摇
 * /lucky 抽奖接口
 * wrk -t10 -c10 -d5 http://localhost:8080/lucky
 */
package main

import (
	"fmt"
	"github.com/kataras/iris"
	"github.com/kataras/iris/mvc"
	"log"
	"math/rand"
	"os"
	"sync"
	"time"
)

// 奖品类型
const (
	giftTypeCoin      = iota // 虚拟币
	giftTypeCoupon           // 不同券
	giftTypeCouponFix        // 相同券
	giftTypeRealSmall        // 实物小奖
	giftTypeRealLarge        // 实物大奖
)

// 奖品
type gift struct {
	id       int      // ID
	name     string   // 名称
	pic      string   // 图片
	link     string   // 链接
	gType    int      // 类型
	data     string   // 数据（特定的配置信息）
	dataList []string // 数据集合
	total    int      // 总数（0表示无限）
	left     int      // 剩余数量
	inUse    bool     // 是否使用
	rate     int      // 中奖率，万分之N，0-9999
	rateMin  int      // 中奖最小值
	rateMax  int      // 中奖最大值
}

// 最大中奖号码
const rateMax = 10000

// 奖品列表
var giftList []*gift

// 日志
var logger *log.Logger
var mu sync.Mutex

type lotteryController struct {
	Ctx iris.Context
}

// 初始化日志
func initLog() {
	f, _ := os.Create("/var/log/shake.log")
	logger = log.New(f, "", log.Ldate|log.Lmicroseconds)
}

// 初始化奖品列表
func initGift() {
	giftList = make([]*gift, 5)
	g1 := gift{
		id:       1,
		name:     "手机大奖(iphoneXS Max512G)",
		pic:      "",
		link:     "",
		gType:    giftTypeRealLarge,
		data:     "",
		dataList: nil,
		total:    2,
		left:     2,
		inUse:    true,
		rate:     1,
		rateMin:  0,
		rateMax:  0,
	}
	giftList[0] = &g1
	g2 := gift{
		id:       2,
		name:     "充电宝",
		pic:      "",
		link:     "",
		gType:    giftTypeRealSmall,
		data:     "",
		dataList: nil,
		total:    5,
		left:     5,
		inUse:    true,
		rate:     10,
		rateMin:  0,
		rateMax:  0,
	}
	giftList[1] = &g2
	g3 := gift{
		id:       3,
		name:     "满200减50优惠券",
		pic:      "",
		link:     "",
		gType:    giftTypeCouponFix,
		data:     "mall-coupon-2018",
		dataList: nil,
		total:    50,
		left:     50,
		inUse:    true,
		rate:     500,
		rateMin:  0,
		rateMax:  0,
	}
	giftList[2] = &g3
	g4 := gift{
		id:       4,
		name:     "减50优惠券",
		pic:      "",
		link:     "",
		gType:    giftTypeCoupon,
		data:     "",
		dataList: []string{"c01", "c02", "c03", "c04", "c05"},
		total:    5,
		left:     5,
		inUse:    true,
		rate:     100,
		rateMin:  0,
		rateMax:  0,
	}
	giftList[3] = &g4
	g5 := gift{
		id:       5,
		name:     "金币",
		pic:      "",
		link:     "",
		gType:    giftTypeCoin,
		data:     "10金币",
		dataList: nil,
		total:    5,
		left:     5,
		inUse:    true,
		rate:     5000,
		rateMin:  0,
		rateMax:  0,
	}
	giftList[4] = &g5

	// 整理中奖区间
	rateStart := 0
	for _, data := range giftList {
		if !data.inUse {
			continue
		}
		data.rateMin = rateStart
		data.rateMax = rateStart + data.rate
		if data.rateMax >= rateMax {
			data.rateMax = rateMax
			rateStart = 0
		} else {
			rateStart += data.rate
		}
	}
}

//
func newApp() *iris.Application {
	app := iris.New()
	mvc.New(app.Party("/")).Handle(&lotteryController{})
	initLog()
	initGift()
	return app
}

func main() {
	app := newApp()
	app.Run(iris.Addr(":8080"))
}

// 奖品数量信息 GET http://localhost:8080/
func (c *lotteryController) Get() string {
	count := 0
	total := 0
	for _, data := range giftList {
		if data.inUse && (data.total == 0 || (data.total > 0 && data.left > 0)) {
			count++
			total += data.left
		}
	}
	return fmt.Sprintf("total quantity of effective prizes:%d, total quantity of limited prizes:%d\n", count, total)
}

// 抽奖 GET http://localhost:8080/lucky
func (c *lotteryController) GetLucky() map[string]interface{} {
	mu.Lock()
	defer mu.Unlock()
	code := luckyCode()
	ok := false
	result := make(map[string]interface{})
	result["success"] = ok

	for _, data := range giftList {
		if !data.inUse || (data.total > 0 && data.left <= 0) {
			continue
		}
		if data.rateMin <= int(code) && data.rateMax > int(code) { // 抽奖号码在奖品范围内，中奖
			// 发奖
			sendData := ""
			switch data.gType {
			case giftTypeCoin:
				ok, sendData = sendCoin(data)
			case giftTypeCoupon:
				ok, sendData = sendCoupon(data)
			case giftTypeCouponFix:
				ok, sendData = sendCouponFix(data)
			case giftTypeRealSmall:
				ok, sendData = sendRealSmall(data)
			case giftTypeRealLarge:
				ok, sendData = sendRealLarge(data)
			}
			if ok {
				// 中奖后得到奖品，生成中奖记录
				saveLuckyData(code, data.id, data.name, data.link, sendData, data.left)
				result["success"] = ok
				result["id"] = data.id
				result["name"] = data.name
				result["link"] = data.link
				result["data"] = sendData
				break
			}
		}
	}
	return result
}

// 记录用户的获奖信息
func saveLuckyData(code int32, id int, name, link, sendData string, left int) {
	logger.Printf("lucky, code=%d, gift=%d, name=%s, link=%s, data=%s, left=%d\n", code, id, name, link, sendData, left)
}

// 产生开奖号码
func luckyCode() int32 {
	seed := time.Now().UnixNano()
	code := rand.New(rand.NewSource(seed)).Int31n(int32(rateMax))
	return code
}

// 发金币
func sendCoin(data *gift) (bool, string) {
	if data.total == 0 { // 数量无限
		return true, data.data
	} else if data.left > 0 { // 有剩余
		data.left = data.left - 1
		return true, data.data
	} else {
		return false, "The prize has been finished"
	}
}

// 发优惠券
func sendCoupon(data *gift) (bool, string) {
	if data.left > 0 { // 有剩余
		left := data.left - 1
		data.left = left
		return true, data.dataList[left]
	} else {
		return false, "The prize has been finished"
	}
}

//
func sendCouponFix(data *gift) (bool, string) {
	if data.total == 0 {
		return true, data.data
	} else if data.left > 0 {
		data.left = data.left - 1
		return true, data.data
	} else {
		return false, "The prize has been finished"
	}
}

//
func sendRealSmall(data *gift) (bool, string) {
	if data.total == 0 {
		return true, data.data
	} else if data.left > 0 {
		data.left = data.left - 1
		return true, data.data
	} else {
		return false, "The prize has been finished"
	}
}

//
func sendRealLarge(data *gift) (bool, string) {
	if data.total == 0 {
		return true, data.data
	} else if data.left > 0 {
		data.left = data.left - 1
		return true, data.data
	} else {
		return false, "The prize has been finished"
	}
}
