// Copyright (c) Huawei Technologies Co., Ltd. 2012-2019. All rights reserved.
// GC 完成时的回调注册机制（与 main 同包）

package main

import (
	"runtime/debug"
	"sync"
	"time"
)

// gcCallback 是 GC 完成时的回调函数类型
type gcCallback func(stats *debug.GCStats)

var (
	gcMu        sync.RWMutex
	gcCallbacks []gcCallback
)

// registerGCCallback 注册 GC 回调函数
func registerGCCallback(cb gcCallback) {
	gcMu.Lock()
	defer gcMu.Unlock()
	gcCallbacks = append(gcCallbacks, cb)
}

// triggerGCCallbacks 触发所有已注册的回调（内部使用）
func triggerGCCallbacks(stats *debug.GCStats) {
	gcMu.RLock()
	cbs := gcCallbacks
	gcMu.RUnlock()

	for _, cb := range cbs {
		cb(stats)
	}
}

// startGCHookMonitor 启动后台协程，通过比较 NumGC 检测 GC 触发
// interval: 检查间隔，建议 500ms ~ 1s
// stopCh: 关闭信号，用于优雅退出
func startGCHookMonitor(interval time.Duration, stopCh <-chan struct{}) {
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	var stats debug.GCStats

	// 初始读取，记录基准 NumGC
	debug.ReadGCStats(&stats)
	lastNumGC := stats.NumGC

	for {
		select {
		case <-ticker.C:
			debug.ReadGCStats(&stats)
			if stats.NumGC > lastNumGC {
				triggerGCCallbacks(&stats)
				lastNumGC = stats.NumGC
			}
		case <-stopCh:
			return
		}
	}
}
