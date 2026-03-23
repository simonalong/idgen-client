package idgen

import "github.com/simonalong/gole-boot/errorx"

var ID_INIT_ERR = errorx.New("ID_ALLOCATE_ERR", "发号器初始化失败")
var ID_CONFIG_ERR = errorx.New("ID_CONFIG_ERR", "发号器配置异常")
var ID_ALLOCATE_ERR = errorx.New("ID_ALLOCATE_ERR", "分配id异常")
var ID_ALLOCATE_NEW_BUFFER_ERR = errorx.New("ID_ALLOCATE_NEW_BUFFER_ERR", "发号器客户端获取服务端新buffer异常")
var ID_MODULE_EMPTY_ERR = errorx.New("ID_MODULE_EMPTY_ERR", "module模块名不可为空")
var ID_MODULE_NOT_FOUND_ERR = errorx.New("ID_MODULE_NOT_FOUND_ERR", "module模块名没有配置")
