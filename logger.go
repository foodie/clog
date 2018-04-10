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

package clog

import "fmt"

//日志的接口
// Logger is an interface for a logger adapter with specific mode and level.
type Logger interface {
	// Level returns minimum level of given logger.
	Level() LEVEL //级别
	// Init accepts a config struct specific for given logger and performs any necessary initialization.
	Init(interface{}) error //初始化
	// ExchangeChans accepts error channel, and returns message receive channel.
	ExchangeChans(chan<- error) chan *Message //改变chan
	// Start starts message processing.
	Start() //开始
	// Destroy releases all resources.
	Destroy() //摧毁
}

//通用的Adapter
// Adapter contains common fields for any logger adapter. This struct should be used as embedded struct.
type Adapter struct {
	level     LEVEL         //级别
	msgChan   chan *Message //消息
	quitChan  chan struct{} //退出的chan
	errorChan chan<- error  //接收数据的chan
}

/**
factories 注册日志工厂方法，返回logger

receivers 包含logger和chan msg 用来处理消息的

**/

//共存方法返回一个Logger
type Factory func() Logger

//定义方法集合
// factories keeps factory function of registered loggers.
var factories = map[MODE]Factory{}

//注册方法
func Register(mode MODE, f Factory) {
	if f == nil {
		panic("clog: register function is nil")
	}
	if factories[mode] != nil {
		panic("clog: register duplicated mode '" + mode + "'")
	}
	factories[mode] = f
}

type receiver struct {
	Logger                //日志接口
	msgChan chan *Message //消息chan
}

//定义两个chan
//1 errorChan	2 quitChan
var (

	//接收多个消息的receivers
	// receivers is a list of loggers with
	//their message channel for broadcasting.
	receivers []*receiver
	//错误chan
	errorChan = make(chan error, 5)
	//退出的chan
	quitChan = make(chan struct{})
)

//出事
func init() {
	// Start background error handling goroutine.
	//启动一个协成，用来监控errorChan
	//如果发生errorChan，调用quitChan
	//发生错误一直打印，如果出现错误就跳出
	go func() {
		for {
			select {
			case err := <-errorChan:
				fmt.Printf("clog: unable to write message: %v\n", err)
			case <-quitChan:
				return
			}
		}
	}()
}

//把logger和msg注册到receivers中

// New initializes and appends a new logger to the receiver list.
// Calling this function multiple times will overwrite previous logger with same mode.
func New(mode MODE, cfg interface{}) error {
	//获取一种消息
	factory, ok := factories[mode]
	if !ok {
		return fmt.Errorf("unknown mode '%s'", mode)
	}

	//得到消息
	logger := factory()
	//初始化消息
	if err := logger.Init(cfg); err != nil {
		return err
	}
	//接收errorChan，返回一个消息chan
	msgChan := logger.ExchangeChans(errorChan)

	// Check and replace previous logger.
	//是否找到
	hasFound := false
	for i := range receivers {
		//找到一种类型的消息处理器
		if receivers[i].mode == mode {
			hasFound = true

			//是否前一个logger
			// Release previous logger.
			receivers[i].Destroy()

			//定义日志和消息处理器
			// Update info to new one.
			receivers[i].Logger = logger
			receivers[i].msgChan = msgChan
			break
		}
	}
	if !hasFound {
		//如果没有找到
		//新建一个消息处理器
		receivers = append(receivers, &receiver{
			Logger:  logger,
			mode:    mode,
			msgChan: msgChan,
		})
	}
	//开始处理消息
	go logger.Start()
	return nil
}

//删除一种类型的日志处理器
//同时弥补空缺
// Delete removes logger from the receiver list.
func Delete(mode MODE) {
	foundIdx := -1
	for i := range receivers {
		if receivers[i].mode == mode {
			foundIdx = i
			receivers[i].Destroy()
		}
	}

	//拷贝receiver
	if foundIdx >= 0 {
		newList := make([]*receiver, len(receivers)-1)
		copy(newList, receivers[:foundIdx])
		copy(newList[foundIdx:], receivers[foundIdx+1:])
		receivers = newList
	}
}
