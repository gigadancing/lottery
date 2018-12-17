/*
 * 1. 即开即得型
 * 2. 双色球自选型
 */
package main

import (
	"fmt"
	"github.com/kataras/iris"
	"github.com/kataras/iris/mvc"
	"math/rand"
	"time"
)

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
	app.Run(iris.Addr(":8080"))
}

// 即开即得型
func (c *lotteryController) Get() string {
	seed := time.Now().UnixNano()
	code := rand.New(rand.NewSource(seed)).Intn(10)
	var prize string
	switch {
	case code == 1:
		prize = "the first prize"
	case code == 2 || code == 3:
		prize = "the second prize"
	case code == 4 || code == 5 || code == 6:
		prize = "the third prize"
	default:
		return fmt.Sprintf("<h1><font color='green'>Unfortunately, you didn't win the prize. ( %d ).</font></h1>", code)
	}
	return fmt.Sprintf("<h1><font color='red'>Congratulations. You won %s. ( %d ).</font></h1>", prize, code)
}

// 双色球自选型
// http://localhost:8080/prize
func (c *lotteryController) GetPrize() string {
	seed := time.Now().UnixNano()
	r := rand.New(rand.NewSource(seed))
	var prize [7]int
	// 6个红色球，1-33
	for i := 0; i < 6; i++ {
		prize[i] = r.Intn(33) + 1
	}
	// 最后一位蓝色球
	prize[6] = r.Intn(16) + 1
	return fmt.Sprintf("Today prize numbers:%v\n", prize)
}
