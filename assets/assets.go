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
