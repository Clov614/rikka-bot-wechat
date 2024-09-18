<div style="text-align: center;">

# Rikka-Bot-WeChat

✨一个简易的微信机器人框架，基于GoLang✨

![](https://img.shields.io/github/go-mod/go-version/Clov614/rikka-bot-wechat "语言")
![](https://img.shields.io/github/stars/Clov614/rikka-bot-wechat?style=flat&color=yellow)
[![](https://img.shields.io/github/actions/workflow/status/Clov614/rikka-bot-wechat/golangci-lint.yml?branch=main)](https://github.com/Clov614/rikka-bot-wechat/actions/workflows/golangci-lint.yml "代码分析")
[![](https://github.com/Clov614/rikka-bot-wechat/actions/workflows/release.yml/badge.svg)](https://github.com/Clov614/rikka-bot-wechat/actions/workflows/release.yml "go-releaser")
[![](https://img.shields.io/github/contributors/Clov614/rikka-bot-wechat)](https://github.com/Clov614/rikka-bot-wechat/graphs/contributors "贡献者")
[![](https://img.shields.io/github/license/Clov614/rikka-bot-wechat)](https://github.com/Clov614/rikka-bot-wechat/blob/main/LICENSE "许可协议")
</div>

## [OneBot V12 标准](https://12.onebot.dev/)

- [x] HTTP
- [x] Http WebHook
- [ ] 正向WebSocket
- [ ] 反向WebSocket

### OneBot客户端快速使用

1. 前往 [Release](https://github.com/Clov614/rikka-bot-wechat/releases) 页面下载对应系统版本的可执行文件

2. [查看配置和启动说明](docs/onebot/README.md)

## 功能介绍

1.
    - [x] 支持规则校验
2.
    - [x] 持久化设置
3.
    - [x] 插件化调用对话（普通对话/长对话）
4.
    - [x] 权限管理
5.
    - [x] cron定时任务
6.
    - [x] OneBot 标准客户端

![cmd_run](/docs/img/product.png)
![http_post](/docs/img/product01.png)

## 机器人快速开始

前往 [Release](https://github.com/Clov614/rikka-bot-wechat/releases) 页面下载对应系统版本的可执行文件

Linux运行

```bash
./rikka-bot-wechat 
```

Win运行(不推荐直接点击exe运行、可以将如下内容写进run.bat双击运行或者直接运行如下命令)

```bash
start cmd /K rikka-bot-wechat.exe
```

## 功能模块

1.
    - [x] [管理权限](docs/plugin/admin/README.md)
2.
    - [x] [链接解析](docs/plugin/bilibili/README.md)
3.
    - [ ] 热更新

## 如何开发插件

todo (
暂未完善文档，可以阅读一下 [/rikka/plugins](https://github.com/Clov614/rikka-bot-wechat/tree/main/rikkabot/plugins))

## 相关链接

| 地址                                                                                      | 简介                            |
|-----------------------------------------------------------------------------------------|-------------------------------|
| [eatmoreapple/openwechat](https://github.com/eatmoreapple/openwechat)                   | golang微信SDK                   |
| [code-innovator-zyx/wechat-gptbot](https://github.com/code-innovator-zyx/wechat-gptbot) | 一个很好的微信机器人项目（微信刷步数功能参考了该项目实现） |