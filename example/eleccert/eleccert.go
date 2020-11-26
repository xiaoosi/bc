package main

import (
	"encoding/json"
	"fmt"
	"strconv"

	"github.com/xuperchain/xuperchain/core/contractsdk/go/code"
	"github.com/xuperchain/xuperchain/core/contractsdk/go/driver"
)


// 该链码的身份是传入的owner字串来决定的，问题很大

// 电子票务主体 实现了合约接口
type elecCert struct {
	User *User
}

// User TODO
// 所属用户
type User struct {
	Owner     string               //所属人
	UserFiles map[string]*UserFile // 所属的文件
}

// UserFile TODO
type UserFile struct {
	Timestamp int64  //时间戳
	Hashval   []byte //文件哈希
}

func newElecCert() *elecCert {
	return &elecCert{}
}

// 存储文件
// 传入user名字，文件hash，时间戳
func (e *elecCert) putFile(user string, filehash string, ts int64) *User {
	// 组件文件结果提
	userFile := &UserFile{
		Timestamp: ts,
		Hashval:   []byte(filehash),
	}

	// 这个user是传入的参数真是令人迷惑

	// 新建user 放入文件并返回
	if e.User != nil {
		e.User.Owner = user
		e.User.UserFiles[filehash] = userFile
		return e.User
	}

	u := &User{
		Owner:     user,
		UserFiles: map[string]*UserFile{},
	}
	u.UserFiles[filehash] = userFile

	e.User = u

	return u
}

// 取文件
func (e *elecCert) getFile(user string, filehash string) (*UserFile, error) {
	if e.User != nil {
		if userFile, ok := e.User.UserFiles[filehash]; ok {
			return userFile, nil
		}
		return nil, fmt.Errorf("User's file:%v no exist", filehash)
	}

	return nil, fmt.Errorf("User:%v no exist", user)
}

// 设置上下文
func (e *elecCert) setContext(ctx code.Context, user string) {
	// 获取链中存储的user
	value, err := ctx.GetObject([]byte(user))
	if err != nil {
	} else {
		userStruc := &User{}
		err = json.Unmarshal(value, userStruc)
		if err != nil {
		}
		e.User = userStruc
	}
}

// 链码初始化
func (e *elecCert) Initialize(ctx code.Context) code.Response {
	// 获取owner的文本
	user := string(ctx.Args()["owner"])
	if user == "" {
		return code.Errors("Missing key: owner")
	}

	e.setContext(ctx, user)

	return code.OK(nil)
}

func (e *elecCert) Save(ctx code.Context) code.Response {
	user := string(ctx.Args()["owner"])
	if user == "" {
		return code.Errors("Missing key: owner")
	}
	filehash := string(ctx.Args()["filehash"])
	if filehash == "" {
		return code.Errors("Missing key: filehash")
	}
	ts := string(ctx.Args()["timestamp"])
	if ts == "" {
		return code.Errors("Missing key: timestamp")
	}

	e.setContext(ctx, user)
	tsInt, _ := strconv.ParseInt(ts, 10, 64)
	userStruc := e.putFile(user, filehash, tsInt)
	userJSON, _ := json.Marshal(userStruc)

	err := ctx.PutObject([]byte(user), userJSON)
	if err != nil {
		return code.Errors("Invoke method PutObject error")
	}

	return code.OK(userJSON)
}

func (e *elecCert) Query(ctx code.Context) code.Response {
	user := string(ctx.Args()["owner"])
	if user == "" {
		return code.Errors("Missing key: owner")
	}
	filehash := string(ctx.Args()["filehash"])
	if filehash == "" {
		return code.Errors("Missing key: filehash")
	}

	e.setContext(ctx, user)

	userFile, err := e.getFile(user, filehash)
	if err != nil {
		return code.Errors("Query not exist")
	}

	userFileJSON, _ := json.Marshal(userFile)
	return code.OK(userFileJSON)
}


func main() {
	driver.Serve(newElecCert())
}
