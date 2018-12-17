/* 抽奖大转盘：
 * 抽奖前，用户已知全部奖品信息；
 * 后端设置各个奖品的中奖概率和数量限制；
 * 更新奖品库存时存在并发安全性问题；
 */
package main

import (
	"fmt"
	"github.com/kataras/iris"
	"github.com/kataras/iris/mvc"
	"math/rand"
	"strings"
	"time"
)

// 奖品中奖概率
type Prate struct {
	Rate      int // 万分之N的中奖概率
	Total     int // 总数量限制，0表示无限量
	CodeStart int // 中奖概率起始编码(包含)
	CodeEnd   int // 中奖概率终止编码(包含)
	Left      int // 剩余数量
}

// 奖品列表
var prizeList = []string{
	"一等奖，火星单程船票",
	"二等奖，凉飕飕南极之旅",
	"三等奖，iPhonex一部",
	"", // 没有中奖
}

// 奖品中奖概率设置，与上面的prizeList对应的设置
var rateList = []Prate{
	{1, 2, 0, 0, 1},
	{2, 2, 1, 2, 2},
	{5, 10, 3, 5, 10},
	{100, 0, 0, 9999, 0},
}

// 抽奖控制器
type lotteryController struct {
	Ctx iris.Context
}

func newApp() *iris.Application {
	app := iris.New()
	mvc.New(app.Party("/")).Handle(&lotteryController{})
	return app
}

func main() {
	app := newApp()
	// http://localhost:8080
	app.Run(iris.Addr(":8080"))
}

// http://localhost:8080
func (c *lotteryController) Get() string {
	c.Ctx.Header("Content-Type", "text/html")
	return fmt.Sprintf("大转盘奖品列表:<br>%s", strings.Join(prizeList, "</br>\n"))
}

//
func (c *lotteryController) GetDebug() string {
	return fmt.Sprintf("中奖概率:%v\n", rateList)
}

func (c *lotteryController) GetPrize() string {
	// 根据随机数匹配奖品
	seed := time.Now().UnixNano()
	r := rand.New(rand.NewSource(seed))
	code := r.Intn(10000)
	var myPrize string
	var prizeRate *Prate

	for i, prize := range prizeList { // 从奖品列表中匹配是否中奖
		rate := &rateList[i]
		if code >= rate.CodeStart && code <= rate.CodeEnd { // 中奖
			myPrize = prize
			prizeRate = rate
			break
		}
	}
	// 没有中奖
	if myPrize == "" {
		myPrize = "很遗憾您没有中奖"
		return myPrize
	}
	// 中奖了，发奖
	if prizeRate.Total == 0 { // 无限量奖品
		return myPrize
	} else if prizeRate.Left > 0 { // 限量奖品
		prizeRate.Left -= 1
		prizeRate.Total -= 1
		return myPrize
	} else {
		myPrize = "很遗憾您没有中奖"
		return myPrize
	}
}
