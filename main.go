package main

import (
	_ "net/http/pprof"

	_ "code.huawei.com/5gcore/base/pub/cell/initializer"
	_ "code.huawei.com/5gcore/cp/lib/util/runtimeext/absgc"

	//这里顺序不能变

	"fmt"
	"runtime/debug"
	"strings"
	"time"

	"code.huawei.com/5gcore/base/data/ddf/common/pool"
	"code.huawei.com/5gcore/base/pub/cell/cell"
	"code.huawei.com/5gcore/base/pub/cell/conf"
	"code.huawei.com/5gcore/base/pub/cell/core"
	"code.huawei.com/5gcore/cp/lib/util/osext/env"
)

func main() {

	// ===== 新增：启动 GC 监控 =====
	stopCh := make(chan struct{})
	go startGCHookMonitor(time.Second, stopCh)

	// GC 触发采样：每次 GC 后打印聚合版，标记 reason=gc
	registerGCCallback(func(_ *debug.GCStats) {
		pool.DumpAllPoolsGCStatsAggWithReason("", "gc") // 空串 = 遍历全部 pool
	})

	// 定时采样：每 5 分钟一次完整版（聚合段在开头 + 逐 pool 明细，整体单次写出），标记 reason=tick
	pool.StartPoolStatsTickerMode(5*time.Minute, "", "tick", stopCh, pool.DumpModeFull)
	// ===== 新增结束 =====

	configPath := env.GetCfgjsnNameFromEnv2()
	fmt.Println("load configration file: ", configPath)
	content := env.GetCellType(configPath)                          //"cellType": "CELL_CTRL"
	celltype := strings.Trim(strings.Split(content, ":")[1], "\" ") //CELL_CTRL
	cellname := strings.ToLower(strings.Split(celltype, "_")[1])    //ctrl

	fmt.Println("cellname: ", cellname)
	if err := createCellByCfg(configPath, cellname); err != nil {
		panic(err)
	}

	// prevent the main goroutine existing
	messages := make(chan string)
	<-messages
}

// createCellByCfg creates a new cell by the specified config file.
func createCellByCfg(configFile string, cellName string) error {
	cfg, err := core.LoadConfig(configFile, &conf.Config{})
	if err != nil {
		return fmt.Errorf("load cell configure file fail:%v \n", err)
	}
	err = cell.New(cellName, cfg).Start()
	if err != nil {
		return fmt.Errorf("start cell failure:%v \n", err)
	}
	return nil
}
