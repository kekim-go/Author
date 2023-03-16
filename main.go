package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/go-redis/redis/v8"
	"github.com/kekim-go/Author/app"
	"github.com/kekim-go/Author/app/ctx"
	"github.com/kekim-go/Author/constant"
	"github.com/kekim-go/Author/model"
	"github.com/robfig/cron/v3"
	log "github.com/sirupsen/logrus"
)

var (
	network = flag.String("network", "tcp", `one of "tcp" or "unix". Must be consistent to -endpoint`)
	port    = flag.Int("service port", 9090, "listen port")
)

func main() {
	flag.Parse()

	ctx := context.Background()

	ballast := make([]byte, 10<<24)
	_ = ballast

	var (
		err error
		a   *app.Application
	)

	a, err = app.New(ctx)
	if err != nil {
		log.Fatal("error ", os.Args[0]+" initialization error: "+err.Error())
		os.Exit(1)
	}

	// 주기적 통계 데이터 저장 처리
	// go runCron(a.Ctx)

	a.Run(*network, fmt.Sprintf(":%d", *port))
}

func runCron(ctx *ctx.Context) {
	c := cron.New()
	c.AddFunc("* * * * *", func() {
		_, err := ctx.RedisDB.LPop(constant.KEY_TRAFFIC_QUEUE)
		if err != nil && err == redis.Nil {
			fmt.Println("Ignore stat ======= ")
			return
		}

		members, err := ctx.RedisDB.SMembers(constant.KEY_TRAFFIC_SET)
		if err != nil && err == redis.Nil {
			fmt.Println("no members ======= ")
			return
		}

		var histories []model.AppTokenHistory

		for _, appTokenKey := range members {
			cntInfo, err := ctx.RedisDB.Get(appTokenKey, "uint")
			if err == nil {
				count := cntInfo.(uint)
				if count > 0 {
					ctx.RedisDB.Delete(appTokenKey)
					temp1 := strings.Split(appTokenKey, ":")[1]
					temp, _ := strconv.Atoi(temp1)
					appTokenId := uint(temp)

					histories = append(histories, model.AppTokenHistory{
						AppTokenId:  appTokenId,
						CallTraffic: count,
					})
				}
			}
		}

		if len(histories) > 0 {
			ctx.Orm.Insert(&histories)
		}

		ctx.RedisDB.LPush(constant.KEY_TRAFFIC_QUEUE, "1")

		time.Sleep(10 * 1000 * time.Millisecond)
		fmt.Println("Run Every min: ", time.Now().String())
	})
	c.Start()
}
