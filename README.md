# Rikka-Bot-WeChat

一个自用的简易微信机器人，基于Go

![rikka](./docs/img/rikka.jpg)

## 快速开始


## 配置项说明

```shell
./rikka-bot-wechat <custom reply msg> <target user ttl>
```

- custom reply msg: 自定义回复的消息
- target user ttl: 回复特定消息后，目标用户的下次触发间隔时间 （默认小时为单位）

For example:

```shell
./rikka-bot-wechat "hello this is rikka death!" 2
```

说明： 回复该条消息“hello this is rikka death!” 每间隔2小时刷新触发