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
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"
)

//基本的slackAttachment数据
type slackAttachment struct {
	Text  string `json:"text"`
	Color string `json:"color"`
}

//slackAttachment的slice
type slackPayload struct {
	Attachments []slackAttachment `json:"attachments"`
}

//基本的类型
const (
	SLACK = "slack"
)

//各个级别的日志的颜色
var slackColors = []string{
	"",        // Trace
	"#3aa3e3", // Info
	"warning", // Warn
	"danger",  // Error
	"#ff0200", // Fatal
}

//slack的配置
type SlackConfig struct {
	// Minimum level of messages to be processed.
	Level LEVEL //日志的级别
	// Buffer size defines how many messages can be queued before hangs.
	BufferSize int64 //buffer的长度
	// Slack webhook URL.
	URL string //定义url
}

//基本的日志，主要针对url？
type slack struct {
	Adapter

	url string
}

//新建一个slack日志对象
func newSlack() Logger {
	return &slack{
		Adapter: Adapter{
			quitChan: make(chan struct{}),
		},
	}
}

//获取级别
func (s *slack) Level() LEVEL { return s.level }

//初始化
func (s *slack) Init(v interface{}) error {
	//配置错误
	cfg, ok := v.(SlackConfig)
	if !ok {
		return ErrConfigObject{"SlackConfig", v}
	}
	//不可用的级别
	if !isValidLevel(cfg.Level) {
		return ErrInvalidLevel{}
	}
	s.level = cfg.Level

	//url不能为空
	if len(cfg.URL) == 0 {
		return errors.New("URL cannot be empty")
	}

	//配置url
	s.url = cfg.URL

	//新建一个msgChan
	s.msgChan = make(chan *Message, cfg.BufferSize)
	return nil
}

//返回当前的msg lever
func (s *slack) ExchangeChans(errorChan chan<- error) chan *Message {
	s.errorChan = errorChan
	return s.msgChan
}

/**
1 对消息的处理
2 对message 进行了json_encode

**/
func buildSlackPayload(msg *Message) (string, error) {
	payload := slackPayload{
		Attachments: []slackAttachment{
			{
				Text:  msg.Body,
				Color: slackColors[msg.Level],
			},
		},
	}
	p, err := json.Marshal(&payload)
	if err != nil {
		return "", err
	}
	return string(p), nil
}

//写日志
func (s *slack) write(msg *Message) {
	//对消息进行处理
	payload, err := buildSlackPayload(msg)

	//消息处理失败
	if err != nil {
		s.errorChan <- fmt.Errorf("slack.buildSlackPayload: %v", err)
		return
	}
	//发送日志信息
	resp, err := http.Post(s.url, "application/json", bytes.NewReader([]byte(payload)))
	if err != nil {
		s.errorChan <- fmt.Errorf("slack: %v", err)
	}
	//关闭请求
	defer resp.Body.Close()

	//如果状态码不是200，发送错误
	if resp.StatusCode/100 != 2 {
		data, _ := ioutil.ReadAll(resp.Body)
		s.errorChan <- fmt.Errorf("slack: %s", data)
	}
}

//开始处理消息
func (s *slack) Start() {
LOOP:
	for {
		select {
		case msg := <-s.msgChan:
			s.write(msg)
		case <-s.quitChan:
			break LOOP
		}
	}

	for {
		if len(s.msgChan) == 0 {
			break
		}

		s.write(<-s.msgChan)
	}
	s.quitChan <- struct{}{} // Notify the cleanup is done.
}

//关闭记录日志
func (s *slack) Destroy() {
	s.quitChan <- struct{}{}
	<-s.quitChan

	close(s.msgChan)
	close(s.quitChan)
}

//注册stack日志类
func init() {
	Register(SLACK, newSlack)
}
