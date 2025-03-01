# 基于OneBot V12 标准的 openwechat-sdk 实现

OneBot V12: [接口标准](https://12.onebot.dev/)

## 支持接口

- [x] HTTP
- [x] Http WebHook
- [ ] 正向WebSocket
- [ ] 反向WebSocket

## 快速开始

~~Linux运行~~ hook客户端不再支持Linux


Win运行(不推荐直接点击exe运行、可以将如下内容写进run.bat双击运行或者直接运行如下命令)
```bash
start cmd /K rikka-bot-wechat.exe -http -autoInject
```
> [!TIP] 
> 【后缀说明】  -bot 开启机器人功能 -http 开启http客户端 -autoInject 自动注入(自动唤起微信完成注入) -debug debug模式

> [!IMPORTANT]
> 【重要事项】首次启动客户端会自动生成配置文件以及拉取需要的文件，拉取失败可根据日志信息自行手动下载到对应目录
> 可通过删除原本的配置文件，启动会生成默认配置文件


## HTTP 正向

> [!TIP]
> HTTP正向 请参考  [接口请求API](https://apifox.com/apidoc/shared-a79a48e7-1352-483b-b9fc-3268bf88ae0d)

目前 `http正向` 仅实现了发送消息 `send_message` 这一个动作请求

更多标准请参考: [OneBot12 消息动作](https://12.onebot.dev/interface/message/actions/#:~:text=%E6%B6%88%E6%81%AF%E5%8A%A8%E4%BD%9C-,send_message%20%E5%8F%91%E9%80%81%E6%B6%88%E6%81%AF,-%E5%85%B3%E4%BA%8E%E6%89%A9%E5%B1%95%E6%AD%A4)


## HTTP 上报器（Post）

Http post 上报`onebot_event`

> [!TIP]
> 事件的格式 请参考  [事件](https://12.onebot.dev/connect/data-protocol/event/)

> [!IMPORTANT]
> 可以使用 `uuid` 作为发送消息的 `send_id` (Http正向中的端点: [send_message](https://apifox.com/apidoc/shared-a79a48e7-1352-483b-b9fc-3268bf88ae0d/api-197448307))
> 
> 需要注意的是，在消息事件中，一旦`uuid`返回不唯一，证明机器人账号好友或群聊重名(好友可以通过更改备注，群聊通过修改群名，使得uuid重新唯一)
> 
> `uuid`返回的不唯一代码: `That uuid is not unique in friends/groups! Error!`

### 关于设置

```yaml
# Http server config
http_server:
  # The Robot HTTP Address default to http://127.0.0.1:10614
  http_address: http://127.0.0.1:10614
  # 接口鉴权
  access_token: rikka-bot
  # 是否启用 get_latest_events 元动作 todo 尚未实现
  event_enabled: false
  # 事件缓冲区大小，超过该大小将会丢弃最旧的事件，0 表示不限大小
  event_buffer_size: 0
# Http 上报器，如不需要请注释掉
http_post:
    - # The httpapi post URL
      url: http://127.0.0.1:8000
      # The httpapi post Access Token
      secret: ""
      # The maximum number of retries
      max_retries: 3
      # 上报请求超时时间
      time_out: 5
    # 支持多个上报器创建
    - # The httpapi post URL
      url: http://127.0.0.1:8001
      # The httpapi post Access Token
      secret: ""
      # The maximum number of retries
      max_retries: 3
      # 上报请求超时时间
      time_out: 5
# 是否开启心跳
enable_heart_beat: true
# 心跳间隔
heart_beat_interval: 5
```


### 消息事件示例

**事件类型（type）**: `message`

```json
[
  {
    // 唯一id
    "id": "f0a06883-31b5-483b-83e4-2739dcd3cace",
    "time": 1721634924.443763,
    // 消息事件
    "type": "message",
    // 消息类型
    "detail_type": "group",
    "sub_type": "",
    "message": [
      {
        // 消息类型 0: 文本 1: 图片
        "msg_type": 0, 
        // 图片url (拼接一下使用，如：http://127.0.0.1:10614/chat_image/2024-07-31/1722358456_8e30abef6dc47bb0.png)
        "chat_img_url": "/chat_image/2024-07-31/1722358456_8e30abef6dc47bb0.png",
        // 文本内容
        "content": "@clover 呼唤1sdaf",
        // 群聊id
        "group_id": "813468086",
        // 群聊中发送者id （非好友无）
        "sender_id": "813468424",
        // 接收者（自己）
        "receiver_id": "2788092443",
        // 群成员 nickname
        "group_name_list": [
          "big号",
          "🗻",
          "clover"
        ],
        // 被艾特的群成员
        "group_at_name_list": [
          "clover"
        ],
        // 是否艾特机器人自己
        "is_at": true,
        // 是否群消息
        "is_group": true,
        // 是否私聊消息（好友消息） 
        "is_friend": false,
        // 是否机器人自己发送的消息
        "is_my_self": false,
        // 是否系统消息
        "is_system": false
      }
    ]
  },
  {
    "id": "dcdd8786-f71a-46b3-a0e7-e6de072a980c",
    "time": 1721634878.242186,
    "type": "message",
    // 私聊消息
    "detail_type": "private",
    "sub_type": "",
    "message": [
      {
        "msg_type": 0,
        "chat_img_url": "",
        "content": "123",
        // 私聊消息 群id 置空
        "group_id": "",
        "sender_id": "813468424",
        "receiver_id": "2788092443",
        "group_name_list": null,
        "group_at_name_list": null,
        "is_at": false,
        "is_group": false,
        "is_friend": true,
        "is_my_self": false,
        "is_system": false
      }
    ]
  }
]
```

### 登录二维码回调事件 (仅在第一次登录时触发)

**事件类型（type）**: `notice`

```json
{
    "id": "a1aaba75-45fb-4e2a-9957-00fe8878a17d",
    "time": 1721639096.389885,
    "type": "notice",
    "detail_type": "login_callback",
    "sub_type": "",
    "Data": {
        "login_url": "https://login.weixin.qq.com/qrcode/wfNzmIDtbw=="
    }
}
```

## TODO LIST

- [ ] 图片保存与本地，根据策略进行过期清理，通过向http server请求返回图片，消息事件不再直接返回图片的原始数据改为返回链接
