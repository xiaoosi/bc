package main

// 该示例毫无意义

import (
	"strconv"

	"github.com/xuperchain/xuperchain/core/contractsdk/go/code"
	"github.com/xuperchain/xuperchain/core/contractsdk/go/driver"
)

// 合约接口实现
type counter struct{}

func (c *counter) Initialize(ctx code.Context) code.Response {
	// 获取创建者
	creator, ok := ctx.Args()["creator"]
	if !ok {
		return code.Errors("missing creator")
	}
	// 写入创建者
	err := ctx.PutObject([]byte("creator"), creator)
	if err != nil {
		return code.Error(err)
	}
	return code.OK(nil)
}

// 传入key key++
func (c *counter) Increase(ctx code.Context) code.Response {
	key, ok := ctx.Args()["key"]
	if !ok {
		return code.Errors("missing key")
	}
	value, err := ctx.GetObject(key)
	cnt := 0
	if err == nil {
		cnt, _ = strconv.Atoi(string(value))
	}
	ctx.Logf("get value %s -> %d", key, cnt)

	cntstr := strconv.Itoa(cnt + 1)

	err = ctx.PutObject(key, []byte(cntstr))
	if err != nil {
		return code.Error(err)
	}
	ctx.EmitJSONEvent("increase", map[string]interface{}{
		"key":   string(key),
		"value": cntstr,
	})
	return code.OK([]byte(cntstr))
}

func (c *counter) Get(ctx code.Context) code.Response {
	key, ok := ctx.Args()["key"]
	if !ok {
		return code.Errors("missing key")
	}
	value, err := ctx.GetObject(key)
	if err != nil {
		return code.Error(err)
	}
	return code.OK(value)
}

func main() {
	driver.Serve(new(counter))
}