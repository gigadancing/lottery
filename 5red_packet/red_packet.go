/*
 * 设置红包
 * curl "http://localhost:8080/set?uid=1&money=100&num=20"
 * 抢红包
 * cur "http://localhost:8080/grab?uid=1&id=2229985533"
 * 并发压力测试
 * wrk -t10 -c10 -d5 "http://localhost:8080/set?uid=1&money=100&num=20"
 */
package main

import (
	"fmt"
	"github.com/kataras/iris"
	"github.com/kataras/iris/mvc"
	"math/rand"
	"sync"
	"time"
)

type task struct {
	id       uint32
	callback chan uint
}

// 红包列表
// var packetList = make(map[uint32][]uint)
var packetList = new(sync.Map)

var chTasks = make(chan task)

//
type lotteryController struct {
	Ctx iris.Context
}

func newApp() *iris.Application {
	app := iris.New()
	mvc.New(app.Party("/")).Handle(&lotteryController{})
	go fetchPacketListMoney()
	return app
}

func main() {
	app := newApp()
	app.Run(iris.Addr(":8080"))

}

// 返回全部红包地址 http://localhost:8080/
func (c *lotteryController) Get() map[uint32][2]int {
	res := make(map[uint32][2]int)
	packetList.Range(func(key, value interface{}) bool {
		id := key.(uint32)
		list := value.([]uint)
		var money int
		for _, v := range list {
			money += int(v)
		}
		res[id] = [2]int{len(list), money}
		return true
	})
	return res
}

// 发红包 http://localhost:8080/set?uid=1&money=100&num=100
func (c *lotteryController) GetSet() string {
	// 获得红包参数
	uid, errUid := c.Ctx.URLParamInt("uid")
	money, errMoney := c.Ctx.URLParamFloat64("money")
	num, errNum := c.Ctx.URLParamInt("num")
	if errUid != nil || errMoney != nil || errNum != nil {
		return fmt.Sprintf("invalid params, errUid=%d, errMoney=%d, errNum=%d", errUid, errMoney, errNum)
	}
	// 将元转为最小单位
	totalMoney := int(money * 100)
	if uid < 1 || totalMoney < num || num < 1 {
		return fmt.Sprintf("invalid params, uid=%d, money=%f,num=%d\n", uid, money, num)
	}
	// 金额分配
	r := rand.New(rand.NewSource(time.Now().UnixNano()))
	rMax := 0.55 // 随机分配最大值(某人抢到的金额占红包剩余金额的比例的最大值)
	if num > 1000 {
		rMax = 0.01
	} else if num >= 100 {
		rMax = 0.1
	} else if num >= 10 {
		rMax = 0.3
	}
	list := make([]uint, num) // 红包列表
	leftMoney := totalMoney   // 剩余红包金额
	leftNum := num            // 剩余红包数量
	for leftNum > 0 {
		if leftNum == 1 { // 最后一个红色，剩余钱都给它
			list[num-1] = uint(leftMoney)
			break
		}
		if leftMoney == leftNum { // 剩余红包数量和剩余个数相等，那么平分
			for i := num - leftNum; i < num; i++ {
				list[i] = 1
			}
			break
		}
		// 用剩余金额减去剩余红包个数，保证每个人都有一份最小单位的金额
		rMoney := int(float64(leftMoney-leftNum) * rMax)
		m := r.Intn(rMoney) // 红包的金额
		if m < 1 {          // 红包最小单位，不能再分
			m = 1
		}
		list[num-leftNum] = uint(m)
		leftMoney -= m
		leftNum--
	}
	// 红包的唯一ID
	id := r.Uint32()
	// packetList[id] = list
	packetList.Store(id, list)
	// 返回抢红包的URL
	return fmt.Sprintf("/get?id=%d&uid=%d&num=%d", id, uid, num)
}

// 抢红包 http://localhost:8080/grab?id=1&uid=1
func (c *lotteryController) GetGrab() string {
	uid, errUid := c.Ctx.URLParamInt("uid")
	id, errId := c.Ctx.URLParamInt("id")
	if errUid != nil || errId != nil {
		return fmt.Sprintf("")
	}
	if uid < 1 || id < 1 {
		return fmt.Sprintf("")
	}
	list1, ok := packetList.Load(uint32(id))
	if !ok || list1 == nil {
		return fmt.Sprintf("red packect not exist, id=%d\n", id)
	}

	// 构造抢红包任务
	callback := make(chan uint)
	t := task{id: uint32(id), callback: callback}
	// 发送任务
	chTasks <- t
	// 接收返回结果
	money := <-callback
	if money <= 0 {
		return "Unfortunately, no red packet was snatched."
	} else {
		return fmt.Sprintf("Congratulations on grabbing a red packet for %d.", money)
	}
}

// 抢红包服务
func fetchPacketListMoney() {
	for {
		t := <-chTasks
		id := t.id
		l, ok := packetList.Load(id)
		if ok && l != nil {
			list := l.([]uint)
			// 产生随机数(抢第几个红包)
			r := rand.New(rand.NewSource(time.Now().UnixNano()))
			index := r.Intn(len(list))
			money := list[index]
			// 更新红包列表(删除已经被抢的红包)
			if len(list) > 1 {
				if index == len(list)-1 { // 最后一个位置的红包
					packetList.Store(uint32(id), list[:index])
				} else if index == 0 { // 第一个位置的红包
					packetList.Store(uint32(id), list[1:])
				} else { // 中间位置的红包
					packetList.Store(uint32(id), append(list[:index], list[index+1:]...))
				}
			} else {
				packetList.Delete(uint32(id))
			}
			t.callback <- money
		} else {
			t.callback <- 0
		}
	}
}
