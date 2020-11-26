package main

import (
	"encoding/json"
	"github.com/xuperchain/xuperchain/core/contractsdk/go/code"
	"github.com/xuperchain/xuperchain/core/contractsdk/go/driver"
	"strconv"
)

type contract struct {
}

// key的头
const (
	USER_PRE string = "USER_"
	FILE_PRE string = "FILE_"
	UF_PRE   string = "USER~FILE_"
	VOUCHER  string = "VOUCHER_" //购入凭证
)

// 文件以hash作为索引的键
type File struct {
	Price  int    `json:"price"`  //文件出售的价格
	Belong string `json:"belong"` //文件的所有者
}

// 初始化合约，部署者直接获取初始的资金
func (c *contract) Initialize(ctx code.Context) code.Response {
	// initSupply为链中发行的初始基金
	initSupplyStr := string(ctx.Args()["initSupply"])
	if initSupplyStr == "" {
		return code.Errors("缺少初始资金")
	}
	initSupply, err := strconv.Atoi(initSupplyStr)
	if err != nil {
		return code.Errors("资金必须是数字")
	}
	if initSupply < 0 {
		return code.Errors("资金必须大于零")
	}
	caller := ctx.Caller()
	// 将资金存入部署者的账户中
	err = ctx.PutObject([]byte(USER_PRE+caller), []byte(strconv.Itoa(initSupply)))
	if err != nil {
		return code.Errors("初始化资金时失败")
	}
	// 部署成功
	return code.OK(nil)
}

// 上传文件
// 所需参数: 上传文件的hash、所需金币
func (c *contract) UploadFile(ctx code.Context) code.Response {
	// 参数校验
	hash := string(ctx.Args()["hash"])
	if hash == "" {
		return code.Errors("没有提供文件的hash")
	}
	priceStr := string(ctx.Args()["price"])
	if priceStr == "" {
		return code.Errors("没有提供文件的价格")
	}
	price, err := strconv.Atoi(priceStr)
	if err != nil {
		return code.Errors("价格必须是数字")
	}
	if price < 0 {
		return code.Errors("资金必须大于零")
	}
	// 文件验重
	fileByte, err := ctx.GetObject([]byte(FILE_PRE + hash))
	if err == nil && fileByte != nil {
		//	文件存在
		return code.Errors("该文件已经被人上传了")
	}
	// 新建文件
	file := File{
		Price:  price,
		Belong: ctx.Caller(),
	}
	fileByte, err = json.Marshal(file)
	if err != nil {
		return code.Errors("编码json时出错：" + err.Error())
	}
	// 写入区块链
	err = ctx.PutObject([]byte(FILE_PRE+hash), fileByte)
	if err != nil {
		return code.Errors("文件写入区块链时出错：" + err.Error())
	}
	// 创建复合键 确保用户可以查询到自己的文件
	err = ctx.PutObject([]byte(UF_PRE+ctx.Caller()+"-"+hash), []byte("")) //传入空串是害怕传nil会报错
	if err != nil {
		return code.Errors("写入复合键时出错：" + err.Error())
	}
	return code.OK(nil)
}

// 购买文件
// 所需参数: 文件hash， 时间戳ts
func (c *contract) BuyFile(ctx code.Context) code.Response {
	// 参数校验
	hash := string(ctx.Args()["hash"])
	if hash == "" {
		return code.Errors("没有提供文件的hash")
	}
	ts := string(ctx.Args()["ts"])
	if ts == "" {
		return code.Errors("没有提供交易的时间戳")
	}
	// 资产校验
	caller := ctx.Caller()
	assetByte, err := ctx.GetObject([]byte(USER_PRE + caller))
	if err != nil || assetByte == nil {
		//用户不存在时 初始化用户并把资产置0
		err = ctx.PutObject([]byte(USER_PRE+caller), []byte("0"))
		if err != nil {
			return code.Errors("设置用户金币失败")
		}
		return code.Errors("您的金币不足")
	}
	asset, err := strconv.Atoi(string(assetByte))
	if err != nil {
		return code.Errors("解码资金失败")
	}
	//获取文件所需的资金
	fileByte, err := ctx.GetObject([]byte(FILE_PRE + hash))
	if err != nil || fileByte == nil {
		return code.Errors("获取文件失败,该文件可能不存在")
	}
	file := File{}
	err = json.Unmarshal(fileByte, &file)
	if err != nil {
		return code.Errors("解码文件失败")
	}
	if asset < file.Price {
		return code.Errors("您的金币不足")
	}
	//金币转移
	// 文件获取者金币减少
	err = ctx.PutObject([]byte(USER_PRE+caller), []byte(strconv.Itoa(asset-file.Price)))
	if err != nil {
		return code.Errors("金币扣除失败")
	}
	//拿到文件所有者的金币
	fileBelongAssetByte, err := ctx.GetObject([]byte(USER_PRE + file.Belong))
	if err != nil {
		return code.Errors("获取文件所有者信息失败")
	}
	fileBelongAsset, err := strconv.Atoi(string(fileBelongAssetByte))
	// 文件所有者金币增加
	err = ctx.PutObject([]byte(USER_PRE+file.Belong), []byte(strconv.Itoa(fileBelongAsset+file.Price)))
	// 生成交易凭证
	// 返回文件的信息
	err = ctx.PutObject([]byte(VOUCHER+caller+"-"+hash), []byte(ts))
	if err != nil {
		return code.Errors("生成交易凭证失败")
	}
	return code.OK(fileByte)

}

// 资产交易
// 参数：收款人，amount
func (c *contract) Transfer(ctx code.Context) code.Response {
	// 验参
	to := string(ctx.Args()["to"])
	if to == "" {
		return code.Errors("没有提供收款人")
	}
	amountStr := string(ctx.Args()["amount"])
	if amountStr == "" {
		return code.Errors("没有提供交易金额")
	}
	amount, err := strconv.Atoi(amountStr)
	if err != nil {
		return code.Errors("交易金额必须是数字")
	}
	if amount <= 0 {
		return code.Errors("交易金额必须大于0")
	}
	// 查询余额是否充足
	from := ctx.Caller()
	fromAssetByte, err := ctx.GetObject([]byte(USER_PRE + from))
	if err != nil || fromAssetByte == nil {
		// 用户不存在
		return code.Errors("余额不足")
	}
	fromAsse, err := strconv.Atoi(string(fromAssetByte))
	if fromAsse < amount {
		return code.Errors("余额不足")
	}
	// 产生交易
	err = ctx.PutObject([]byte(USER_PRE+from), []byte(strconv.Itoa(fromAsse-amount)))
	if err != nil {
		return code.Errors("交易失败：" + err.Error())
	}
	// 获取接受者的余额
	toAssetByte, err := ctx.GetObject([]byte(USER_PRE + from))
	toAsset := 0
	if err == nil || toAssetByte != nil {
		toAsset, err = strconv.Atoi(string(toAssetByte))
		if err != nil {
			return code.Errors("获取接收者金额失败")
		}
	}
	err = ctx.PutObject([]byte(USER_PRE+to), []byte(strconv.Itoa(toAsset+amount)))
	return code.OK(nil)
}

// 查询自己的所有的文件
func (c *contract) GetFiles(ctx code.Context) code.Response {
	// 获取调用者
	caller := ctx.Caller()
	// 拿到复合键中存储的关系
	iter := ctx.NewIterator([]byte(UF_PRE+caller), []byte(UF_PRE+caller+"~"))
	defer func() {
		// 关闭迭代器
		iter.Close()
	}()
	// 文件列表
	out := make([]File, 0)
	for iter.Next() {
		composeKey := string(iter.Key())
		// just like "UF_PRExiaoosi-filehash"
		fileHash := composeKey[len(UF_PRE)+len(caller)+1:]
		// 通过文件hash 获取文件的详细信息
		fileByte, err := ctx.GetObject([]byte(FILE_PRE + fileHash))
		if err != nil {
			continue
		}
		file := File{}
		err = json.Unmarshal(fileByte, &file)
		if err != nil {
			continue
		}
		out = append(out, file)
	}
	outByte, err := json.Marshal(out)
	if err != nil {
		return code.Errors("编码结果json串时出错")
	}
	// 返回文件列表的json字串
	return code.OK(outByte)
}

// 查询文件凭证
// 链上公开，任何人都可以查询
// args: 所有者、文件hash
// 返回{"is_owned": ?, "time_stamp": 时间戳}
type Response struct {
	IsOwned   bool   `json:"is_owned"`
	TimeStamp string `json:"time_stamp"`
}

func (c *contract) GetVoucher(ctx code.Context) code.Response {
	// 验参
	user := string(ctx.Args()["user"])
	if user == "" {
		return code.Errors("没有提供文件所有人")
	}
	hash := string(ctx.Args()["hash"])
	if hash == "" {
		return code.Errors("没有提供文件哈希")
	}
	key := VOUCHER + user + "-" + hash
	ts, err := ctx.GetObject([]byte(key))
	out := Response{
		IsOwned:   true,
		TimeStamp: "",
	}
	if err != nil || ts == nil {
		out.IsOwned = false
	}
	out.TimeStamp = string(ts)
	outByte, err := json.Marshal(out)
	if err != nil {
		return code.Errors("编码json字串出错")
	}
	return code.OK(outByte)
}

// 查询自己的金币
func (c *contract) GetAsset(ctx code.Context) code.Response {
	caller := ctx.Caller()
	asset, err := ctx.GetObject([]byte(USER_PRE + caller))
	if err != nil || asset == nil {
		return code.OK([]byte("0"))
	}
	return code.OK(asset)
}

func main() {
	driver.Serve(&contract{})
}
