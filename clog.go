// Copyright 2017 Unknwon
//
// Licensed under the Apache License, Version 2.0 (the "License"): you may
// not use this file except in compliance with the License. You may obtain
// a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS, WITHOUT
// WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied. See the
// License for the specific language governing permissions and limitations
// under the License.

// Clog is a channel-based logging package for Go.
package clog

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"
)

//版本
const (
	_VERSION = "1.1.1"
)

//获取版本
// Version returns current version of the package.
func Version() string {
	return _VERSION
}

//得到模式和级别
type (
	MODE  string //定义模式
	LEVEL int    //定义变量
)

//日志参数常量
const (
	TRACE LEVEL = iota //0
	INFO               //1
	WARN               //2
	ERROR              //3
	FATAL              //4
)

//建立一个level和字符串的对应表
var formats = map[LEVEL]string{
	TRACE: "[TRACE] ",
	INFO:  "[ INFO] ",
	WARN:  "[ WARN] ",
	ERROR: "[ERROR] ",
	FATAL: "[FATAL] ",
}

//是否可用
// isValidLevel returns true if given level is in the valid range.
func isValidLevel(level LEVEL) bool {
	return level >= TRACE && level <= FATAL
}

//消息的类型
// Message represents a log message to be processed.
type Message struct {
	Level LEVEL  //级别
	Body  string //内容
}

func Write(level LEVEL, skip int, format string, v ...interface{}) {
	//新建一个msg
	msg := &Message{
		Level: level,
	}

	// Only error and fatal information needs locate position for debugging.
	// But if skip is 0 means caller doesn't care so we can skip.

	//如果Level == ERROR且存在skip
	if msg.Level >= ERROR && skip > 0 {
		pc, file, line, ok := runtime.Caller(skip)
		if ok {
			// Get caller function name.
			fn := runtime.FuncForPC(pc)
			var fnName string
			if fn == nil {
				fnName = "?()"
			} else {
				fnName = strings.TrimLeft(filepath.Ext(fn.Name()), ".") + "()"
			}

			if len(file) > 20 {
				file = "..." + file[len(file)-20:]
			}
			msg.Body = formats[level] + fmt.Sprintf("[%s:%d %s] ", file, line, fnName) + fmt.Sprintf(format, v...)
		}
	}
	//如果消息的body为空
	//获取消息内容
	if len(msg.Body) == 0 {
		msg.Body = formats[level] + fmt.Sprintf(format, v...)
	}

	//从消息的接收者里面
	for i := range receivers {
		//如果消费者的level大于当前日志的级别，则跳出
		if receivers[i].Level() > level {
			continue
		}
		//接收消息
		receivers[i].msgChan <- msg
	}
}

//trace日志
func Trace(format string, v ...interface{}) {
	Write(TRACE, 0, format, v...)
}

//info日志
func Info(format string, v ...interface{}) {
	Write(INFO, 0, format, v...)
}

//warn日志
func Warn(format string, v ...interface{}) {
	Write(WARN, 0, format, v...)
}

//error日志
func Error(skip int, format string, v ...interface{}) {
	Write(ERROR, skip, format, v...)
}

//fatal日志
func Fatal(skip int, format string, v ...interface{}) {
	Write(FATAL, skip, format, v...)
	//关闭
	Shutdown()
	//退出
	os.Exit(1)
}

func Shutdown() {
	//摧毁所有的消息接收者
	for i := range receivers {
		receivers[i].Destroy()
	}

	// Shutdown the error handling goroutine.
	//给quitChan发送数据
	quitChan <- struct{}{}
	for {
		//如果errorChan长度为0 退出
		if len(errorChan) == 0 {
			break
		}
		//发送数据
		fmt.Printf("clog: unable to write message: %v\n", <-errorChan)
	}
}
