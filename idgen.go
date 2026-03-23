package idgen

import (
	"fmt"
	"sync"
	"sync/atomic"

	"github.com/simonalong/gole-boot/client/grpc"
	"github.com/simonalong/gole/config"
	"github.com/simonalong/gole/global"
	"github.com/simonalong/gole/logger"
	"github.com/simonalong/gole/util"
	"github.com/simonalong/idgen-client/api"
)

var loadLock sync.Mutex
var configLoaded = atomic.Bool{}

type IdClient struct {
	bufferClient        api.BufferClient
	serviceBufferConfig *ServiceBufferConfig
	moduleAllocatorMap  map[string]*ModuleAllocator
}

type ServiceBufferConfig struct {
	ServiceName string         `json:"service_name"`
	Modules     []ModuleConfig `json:"modules"`
}

type ModuleConfig struct {
	Module     string `json:"module"`
	BufferSize int64  `json:"buffer_size"`
}

func NewClient() (*IdClient, error) {
	loadLock.Lock()
	defer loadLock.Unlock()
	if !configLoaded.Load() {
		config.Load()
	}

	var serviceBufferConfig ServiceBufferConfig
	// 读取buffer分配的配置
	if err := config.GetValueObject("cbb.idgen", &serviceBufferConfig); err != nil {
		logger.Errorf("解析buffer的配置失败%v", err)
		return nil, ID_CONFIG_ERR.WithDetail(fmt.Sprintf("解析buffer的配置失败%v", err))
	}
	return NewClientWith(&serviceBufferConfig)
}

func NewClientWith(ServiceBufferConfig *ServiceBufferConfig) (*IdClient, error) {
	// 创建grpc客户端
	conn, err := grpc.NewClientConn("cbb-mid-srv-idgen")
	if err != nil {
		logger.Errorf("创建发号器客户端失败：%v", err)
		return nil, ID_INIT_ERR.WithDetail(fmt.Sprintf("创建发号器客户端失败：%v", err))
	}
	configLoaded.Store(true)

	idClient := &IdClient{
		bufferClient:        api.NewBufferClient(conn),
		serviceBufferConfig: ServiceBufferConfig,
	}

	// 初始化module的buffer分配器
	err = idClient.initModuleAllocator()
	if err != nil {
		logger.Errorf("初始化module的buffer分配器失败：%v", err)
		return nil, ID_ALLOCATE_NEW_BUFFER_ERR.WithDetail(fmt.Sprintf("初始化module的buffer分配器失败：%v", err))
	}
	return idClient, nil
}

func (idClient *IdClient) GetId(module string) (int64, error) {
	if module == "" {
		return 0, ID_MODULE_EMPTY_ERR
	}
	moduleAllocator, ok := idClient.moduleAllocatorMap[module]
	if !ok {
		return 0, ID_MODULE_NOT_FOUND_ERR.WithMsg("模块名【" + module + "】没有配置")
	}
	return moduleAllocator.getId()
}

// GetIdOfBase62 获取base62编码的id，其中base62编码字符串最长是11个字符
// @param module 模块名
func (idClient *IdClient) GetIdOfBase62(module string) (string, error) {
	id, err := idClient.GetId(module)
	if err != nil {
		return "", err
	}
	return util.Int64ToBase62(id), nil
}

func (idClient *IdClient) allocateBuffer(serviceName, module string, bufferSize int64) (int64, error) {
	allocateReq := &api.BufferAllocateReq{ServiceName: serviceName, Module: module, BufferSize: bufferSize}
	// 注册服务或者分配本次的发号器id
	allocateRsp, err := idClient.bufferClient.AllocateNew(global.GetGlobalContext(), allocateReq)
	if err != nil {
		logger.Errorf("申请新的buffer失败：%v", err)
		return 0, err
	}
	return allocateRsp.StartIndex, nil
}

func (idClient *IdClient) initModuleAllocator() error {
	moduleAllocatorMap := map[string]*ModuleAllocator{}
	for _, moduleConfig := range idClient.serviceBufferConfig.Modules {
		allocator := &ModuleAllocator{
			idClient:     idClient,
			ServiceName:  idClient.serviceBufferConfig.ServiceName,
			ModuleConfig: moduleConfig,
		}
		err := allocator.Init()
		if err != nil {
			return err
		}
		moduleAllocatorMap[moduleConfig.Module] = allocator
	}
	idClient.moduleAllocatorMap = moduleAllocatorMap
	return nil
}
