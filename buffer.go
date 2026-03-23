package idgen

import (
	"sync"
	"sync/atomic"

	"github.com/simonalong/gole/goid"

	"github.com/simonalong/gole/logger"
)

// 第二buffer的刷新比率：0.4（暂定）
const refreshRatio = 0.4

const (
	// StateNone 无状态
	StateNone = iota
	// StateRefresh 准备刷新
	StateRefresh
	StateFinish
	// StateErr 刷新异常
	StateErr
)

type ModuleAllocator struct {
	idClient *IdClient
	// --------- 基础配置 ---------
	ServiceName  string
	ModuleConfig ModuleConfig

	// --------- 参数配置 ---------
	buffer1       *BufferConfig
	buffer2       *BufferConfig
	currentBuffer *BufferConfig
	refreshValue  int64

	currentSequence atomic.Int64
	refreshState    atomic.Int32
	refreshLocker   sync.Mutex
	refreshCond     *sync.Cond
}

type BufferConfig struct {
	StartIndex int64
}

type BufferAllocateConfig struct {
	ServiceName string `json:"service_name"`
	Module      string `json:"module"`
	BufferSize  int64  `json:"buffer_size"`
}

func (module *ModuleAllocator) Init() error {
	module.refreshLocker = sync.Mutex{}
	module.refreshCond = sync.NewCond(&module.refreshLocker)
	module.currentSequence = atomic.Int64{}
	module.refreshState = atomic.Int32{}
	module.buffer1 = &BufferConfig{}
	module.buffer2 = &BufferConfig{}
	module.currentBuffer = &BufferConfig{}

	// 注册服务或者分配首次初始id
	startIndex, err := module.idClient.allocateBuffer(module.ServiceName, module.ModuleConfig.Module, module.ModuleConfig.BufferSize)
	if err != nil {
		logger.Errorf("向发号器服务注册模块失败：%v", err)
		return ID_INIT_ERR
	}
	module.currentBuffer.StartIndex = startIndex
	module.buffer1.StartIndex = startIndex
	module.refreshValue = int64(float64(module.ModuleConfig.BufferSize) * refreshRatio)
	return nil
}

func (module *ModuleAllocator) getId() (int64, error) {
	for {
		if module.refreshState.Load() == StateErr {
			return 0, ID_ALLOCATE_ERR
		}

		// 判断是否到达阈值，到达阈值则启动二级buffer刷新处理
		if module.currentSequence.Load() >= module.refreshValue && module.refreshState.Load() == StateNone {
			if module.refreshState.Load() == StateRefresh || module.refreshState.Load() == StateFinish {
				continue
			}

			// 异常返回异常
			if module.refreshState.Load() == StateErr {
				return 0, ID_ALLOCATE_ERR
			}

			module.refreshLocker.Lock()
			if module.refreshState.Load() == StateRefresh || module.refreshState.Load() == StateFinish {
				module.refreshLocker.Unlock()
				continue
			}

			if module.refreshState.Load() == StateErr {
				module.refreshLocker.Unlock()
				return 0, ID_ALLOCATE_ERR
			}

			module.refreshState.Store(StateRefresh)
			module.refreshLocker.Unlock()

			// 刷新二级buffer
			module.refreshBuffer()
			continue
		}

		// 判断是否到达最后
		if module.currentSequence.Load()+1 > module.ModuleConfig.BufferSize {
			module.refreshLocker.Lock()
			if module.refreshState.Load() == StateNone {
				module.refreshLocker.Unlock()
				continue
			}

			if module.refreshState.Load() == StateRefresh {
				logger.Warnf("service=%v，module=%v对应的bufferSize【%v】太短了，建议改大", module.ServiceName, module.ModuleConfig.Module, module.ModuleConfig.BufferSize)
				module.refreshLocker.Unlock()
				continue
			}

			if module.refreshState.Load() == StateErr {
				module.refreshLocker.Unlock()
				return 0, ID_ALLOCATE_ERR
			}

			if module.refreshState.Load() == StateFinish {
				// 切换新的buffer
				if err := module.switchBuffer(); err != nil {
					module.refreshLocker.Unlock()
					return 0, err
				}

				module.refreshCond.Broadcast()
				module.refreshLocker.Unlock()
			}
		}

		seq := module.currentSequence.Load()
		if seq < module.ModuleConfig.BufferSize && module.currentSequence.CompareAndSwap(seq, seq+1) {
			return module.currentBuffer.StartIndex + seq, nil
		}
		continue
	}
}

func (module *ModuleAllocator) reachRefreshBuffer(sequence int64) bool {
	return sequence >= module.refreshValue && module.refreshState.Load() == StateNone
}

func (module *ModuleAllocator) refreshBuffer() {
	goid.Go(func() {
		module.refreshLocker.Lock()
		defer func() {
			module.refreshLocker.Unlock()
		}()

		startIndex, err := module.idClient.allocateBuffer(module.ServiceName, module.ModuleConfig.Module, module.ModuleConfig.BufferSize)
		if err != nil {
			logger.Errorf("【异步】调用发号器服务分配新的buffer起始点失败：%v", err)
			module.refreshState.Store(StateErr)
			return
		}
		module.setNewBuffer(startIndex)
		module.refreshState.Store(StateFinish)
	})
}

func (module *ModuleAllocator) switchBuffer() error {
	if module.currentBuffer.StartIndex == module.buffer1.StartIndex {
		module.currentBuffer.StartIndex = module.buffer2.StartIndex
	} else if module.currentBuffer.StartIndex == module.buffer2.StartIndex {
		module.currentBuffer.StartIndex = module.buffer1.StartIndex
	}

	module.currentSequence.Store(0)
	module.refreshState.Store(StateNone)
	return nil
}

func (module *ModuleAllocator) setNewBuffer(startIndex int64) {
	if module.currentBuffer.StartIndex == module.buffer1.StartIndex {
		module.buffer2.StartIndex = startIndex
	} else if module.currentBuffer.StartIndex == module.buffer2.StartIndex {
		module.buffer1.StartIndex = startIndex
	}
}
