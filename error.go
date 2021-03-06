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

//错误的定义，
type ErrConfigObject struct {
	//错误说明
	expect string
	//存储任意的错误类型
	got interface{}
}

//错误日志类
func (err ErrConfigObject) Error() string {
	return fmt.Sprintf("config object is not an instance of %s, instead got '%T'", err.expect, err.got)
}

//日志不可用的错误
type ErrInvalidLevel struct{}

func (err ErrInvalidLevel) Error() string {
	return "input level is not one of: TRACE, INFO, WARN, ERROR or FATAL"
}
