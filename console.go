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

import (
	"log"

	//颜色处理？
	"github.com/fatih/color"
)

//定义console名称
const CONSOLE MODE = "console"

//定义各种颜色的方法
//trace blue, info green, warn yellow , error red, fatal hired
// Console color set for different levels.
var consoleColors = []func(a ...interface{}) string{
	color.New(color.FgBlue).SprintFunc(),   // Trace
	color.New(color.FgGreen).SprintFunc(),  // Info
	color.New(color.FgYellow).SprintFunc(), // Warn
	color.New(color.FgRed).SprintFunc(),    // Error
	color.New(color.FgHiRed).SprintFunc(),  // Fatal
}

//基本的配置
type ConsoleConfig struct {
	// Minimum level of messages to be processed.
	Level LEVEL //日志的级别
	// Buffer size defines how many messages can be queued before hangs.
	BufferSize int64 // message的buf的大小
}

//Adapter: level, msg chan, quit chan, error chan<-
//

type console struct {
	*log.Logger //包含自带的日志
	Adapter     //包含Adapter
}

//新建一个console
func newConsole() Logger {
	return &console{
		//直接输出到命令行
		Logger: log.New(color.Output, "", log.Ldate|log.Ltime),
		//定义退出的chan
		Adapter: Adapter{
			quitChan: make(chan struct{}),
		},
	}
}

//返回级别
func (c *console) Level() LEVEL { return c.level }

//初始化
func (c *console) Init(v interface{}) error {
	//传入的基本的配置
	cfg, ok := v.(ConsoleConfig)
	if !ok {
		return ErrConfigObject{"ConsoleConfig", v}
	}
	//是否是可用的日志级别
	if !isValidLevel(cfg.Level) {
		return ErrInvalidLevel{}
	}
	//定义日志级别
	c.level = cfg.Level
	//定义chan的大小
	c.msgChan = make(chan *Message, cfg.BufferSize)
	return nil
}

//把error chan赋值到当前的chan里面
//返回当前的msgChan
func (c *console) ExchangeChans(errorChan chan<- error) chan *Message {
	c.errorChan = errorChan
	return c.msgChan
}

//按照级别显示日志，显示日志的时候有颜色
func (c *console) write(msg *Message) {
	c.Logger.Print(consoleColors[msg.Level](msg.Body))
}

//开始运行
func (c *console) Start() {
LOOP:
	for {
		//接收消息，打印消息
		//如果接收到quit chan退出当前的loop
		select {
		case msg := <-c.msgChan:
			c.write(msg)

		case <-c.quitChan: //从quit读到过数据
			break LOOP
		}
	}
	//msgChan处理完了跳出
	for {
		if len(c.msgChan) == 0 {
			break
		}

		c.write(<-c.msgChan)
	}
	//把数据发送给quitchan
	c.quitChan <- struct{}{} // Notify the cleanup is done.
}

//摧毁日志
func (c *console) Destroy() {
	//发送关闭，跳出接收日志的循环
	c.quitChan <- struct{}{}

	//等待处理剩余消息的完毕
	<-c.quitChan
	//关闭msgChan，关闭quitchan
	close(c.msgChan)
	close(c.quitChan)
}

//注册日志
func init() {
	Register(CONSOLE, newConsole)
}
