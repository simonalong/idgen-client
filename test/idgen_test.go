package test

import (
	"fmt"
	"sync"
	"testing"

	"github.com/berkaroad/detectlock-go"
	"github.com/simonalong/gole-boot/server/http/rsp"
	"github.com/simonalong/gole/maps"
	"github.com/stretchr/testify/assert"

	"github.com/gin-gonic/gin"
	"github.com/simonalong/gole-boot/server"
	httpServer "github.com/simonalong/gole-boot/server/http"

	cmap "github.com/orcaman/concurrent-map"
	"github.com/simonalong/gole/logger"
	"github.com/simonalong/gole/util"
	"github.com/simonalong/idgen-client"
)

func TestIdgen(t *testing.T) {
	idClient, err := idgen.NewClient()
	if err != nil {
		logger.Errorf("初始化异常：%v", err)
		return
	}

	id, err := idClient.GetId("test_module1")
	if err != nil {
		logger.Errorf("获取id失败：%v", err)
		return
	}
	fmt.Println(id)
}

func TestIdgen2(t *testing.T) {
	idClient, err := idgen.NewClient()
	if err != nil {
		logger.Errorf("初始化异常：%v", err)
		return
	}

	for i := 0; i < 10; i++ {
		id, err := idClient.GetId("test_module1")
		if err != nil {
			logger.Errorf("获取id失败：%v", err)
			return
		}
		fmt.Println(id)
	}
}

func TestIdgenStr(t *testing.T) {
	idClient, err := idgen.NewClient()
	if err != nil {
		logger.Errorf("初始化异常：%v", err)
		return
	}

	for i := 0; i < 10; i++ {
		id, err := idClient.GetIdOfBase62("test_module1")
		if err != nil {
			logger.Errorf("获取id失败：%v", err)
			return
		}
		fmt.Println(id)
	}
}

func TestIdgenBatch(t *testing.T) {
	idClient, _ := idgen.NewClient()

	bitmap := util.NewBitmap(200000000)
	success := true

	for i := 0; i < 10; i++ {
		id, _ := idClient.GetId("test_module1")
		fmt.Println(i, id)
		if bitmap.Check(uint64(id)) {
			logger.Errorf("该数据【%v】已经存在", id)
			success = false
		} else {
			bitmap.Set(uint64(id))
		}
	}

	if success {
		fmt.Println("结果：成功")
	} else {
		fmt.Println("结果：异常")
	}
}

func TestIdgenBatchGoroutine(t *testing.T) {
	idClient, err := idgen.NewClient()
	if err != nil {
		logger.Errorf("初始化异常：%v", err)
		return
	}

	bitmap := util.NewBitmap(200000000)
	countGroup := sync.WaitGroup{}
	success := true

	for i := 0; i < 200; i++ {
		countGroup.Add(1)
		go func() {
			id, err := idClient.GetId("test_module2_big")
			if err != nil {
				logger.Errorf("获取id异常：%v", err)
				countGroup.Done()
				return
			}
			if bitmap.Check(uint64(id)) {
				logger.Errorf("%v该数据【%v】已经存在", i, id)
				success = false
			} else {
				bitmap.Set(uint64(id))
			}
			countGroup.Done()
		}()
	}
	countGroup.Wait()
	assert.True(t, success)
}

func TestIdgenBatchGoroutine_print(t *testing.T) {
	idClient, err := idgen.NewClient()
	if err != nil {
		logger.Errorf("初始化异常：%v", err)
		return
	}

	bitmap := util.NewBitmap(200000000)
	countGroup := sync.WaitGroup{}
	success := true
	dataMap := cmap.New()

	for i := 0; i < 20; i++ {
		countGroup.Add(1)
		go func() {
			id, err := idClient.GetId("test_module2_big")
			if err != nil {
				logger.Errorf("获取发号器异常：%v", err)
				return
			}
			dataMap.Set(util.ToString(i), id)
			fmt.Println(i, id)
			if bitmap.Check(uint64(id)) {
				logger.Errorf("%v该数据【%v】已经存在", i, id)
				success = false
			} else {
				bitmap.Set(uint64(id))
			}
			countGroup.Done()
		}()
	}
	countGroup.Wait()
	if success {
		fmt.Println("结果：成功")
	} else {
		fmt.Println("结果：异常")
	}
	fmt.Println(util.ToJsonString(dataMap))
	assert.True(t, success)
}

var idClient *idgen.IdClient

func TestIdgenBatchGoroutine_demo(t *testing.T) {
	httpServer.Get("id", getId)
	httpServer.Get("check", check)
	detectlock.EnableDebug()
	_idClient, _ := idgen.NewClient()
	idClient = _idClient
	server.Run()
}

func check(c *gin.Context) (interface{}, error) {
	items := detectlock.Items()
	fmt.Println(detectlock.DetectAcquired(items))        // 检测获得锁的goroutine列表
	fmt.Println(detectlock.DetectReentry(items))         // 检测锁重入的goroutine列表
	fmt.Println(detectlock.DetectLockedEachOther(items)) // 检测互锁的goroutine列表

	dataMap := maps.Of()
	dataMap.Put("DetectAcquired", detectlock.DetectAcquired(items))
	dataMap.Put("DetectReentry", detectlock.DetectReentry(items))
	dataMap.Put("DetectLockedEachOther", detectlock.DetectLockedEachOther(items))

	rsp.Done(c, dataMap.ToMap())
	return dataMap.ToMap(), nil
}
func getId(*gin.Context) (any, error) {

	bitmap := util.NewBitmap(200000000)
	countGroup := sync.WaitGroup{}
	success := true
	dataMap := cmap.New()
	printNum := false
	printDone := false

	for i := 0; i < 40; i++ {
		countGroup.Add(1)
		go func() {
			id, _ := idClient.GetId("test_module1")
			dataMap.Set(util.ToString(i), id)
			if printNum {
				fmt.Println(i, id)
			}

			if bitmap.Check(uint64(id)) {
				logger.Errorf("%v该数据【%v】已经存在", i, id)
				success = false
			} else {
				bitmap.Set(uint64(id))
			}
			if printDone {
				fmt.Println(i, "done")
			}

			countGroup.Done()
		}()
	}
	countGroup.Wait()
	if success {
		fmt.Println("结果：成功")
	} else {
		fmt.Println("结果：异常")
	}
	fmt.Println(util.ToJsonString(dataMap))
	return nil, nil
}
