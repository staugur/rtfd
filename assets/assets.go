/*
   Copyright 2021 Hiroshi.tao

   Licensed under the Apache License, Version 2.0 (the "License");
   you may not use this file except in compliance with the License.
   You may obtain a copy of the License at

       http://www.apache.org/licenses/LICENSE-2.0

   Unless required by applicable law or agreed to in writing, software
   distributed under the License is distributed on an "AS IS" BASIS,
   WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
   See the License for the specific language governing permissions and
   limitations under the License.
*/

package assets

import _ "embed" // embed static file

//RtfdCFG 配置文件示例内容
//go:embed rtfd.cfg
var RtfdCFG []byte

//RtfdJS 文档站点运行时引入的JS内容
//go:embed rtfd.js
var RtfdJS []byte

//BuiderSH 构建脚本内容
//go:embed builder.sh
var BuiderSH []byte

//AppVersion 程序版本号
//go:embed VERSION
var AppVersion string
