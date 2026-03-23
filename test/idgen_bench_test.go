package test

import (
	"testing"

	"github.com/simonalong/gole/logger"
	"github.com/simonalong/idgen-client"
)

var idClientOfBench *idgen.IdClient

func init() {
	_idClient, err := idgen.NewClient()
	idClientOfBench = _idClient
	if err != nil {
		logger.Errorf("初始化异常：%v", err)
		return
	}
}
func BenchmarkSprintf(b *testing.B) {
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := idClientOfBench.GetId("test_module1")
		if err != nil {
			logger.Errorf("获取id失败：%v", err)
			return
		}
	}
}
