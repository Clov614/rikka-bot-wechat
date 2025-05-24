// Package httpapi
// @Author Clover
// @Data 2024/7/20 下午9:37:00
// @Desc http and http webhook
package httpapi

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/Clov614/rikka-bot-wechat/rikkabot/config"
	"github.com/Clov614/rikka-bot-wechat/rikkabot/coreapi"
	"io"
	"math/rand"
	"net/http"
	"net/url"
	"strings"
	"time"

	wcf "github.com/Clov614/wcf-rpc-sdk"

	"github.com/Clov614/logging"

	"github.com/Clov614/rikka-bot-wechat/rikkabot"
	"github.com/Clov614/rikka-bot-wechat/rikkabot/groupmanager"
	"github.com/Clov614/rikka-bot-wechat/rikkabot/message"
	"github.com/Clov614/rikka-bot-wechat/rikkabot/onebot/dto/event"
	"github.com/Clov614/rikka-bot-wechat/rikkabot/onebot/oneboterr"
	"github.com/Clov614/rikka-bot-wechat/rikkabot/utils/imgutil"
	"github.com/Clov614/rikka-bot-wechat/rikkabot/utils/timeutil"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

// FriendRequestEvent 好友请求事件
// NOTE: 此结构体理想情况下应定义在 rikkabot/onebot/dto/event/event.go 中
type FriendRequestEvent struct {
	event.Event        // 内嵌通用事件字段: Id, Time, Type, DetailType, SubType 等
	UserId      string `json:"user_id"`      // 请求者 wxid
	Comment     string `json:"comment"`      // 验证信息
	Flag        string `json:"flag"`         // 加好友请求的 flag (通常是 ticket_v4)
	DisplayName string `json:"display_name"` // 请求者昵称
	// 以下字段为微信特有，传递给动作处理器以便调用SDK
	TicketV3 string `json:"_ticket_v3,omitempty"` // v3 ticket
	Scene    int64  `json:"_scene,omitempty"`     // 场景值
}

// HandleFriendRequestParams 定义了 /handle_friend_request 接口的请求参数
// NOTE: 此结构体理想情况下也应与 ActionRequest 一起定义在更合适的位置，例如 dto/action 或 dto/params
type HandleFriendRequestParams struct {
	Flag     string `json:"flag" binding:"required"`      // 来自 FriendRequestEvent 的 Flag (ticket_v4)
	Approve  bool   `json:"approve"`                      // true 同意, false 拒绝
	Remark   string `json:"remark,omitempty"`             // 同意时可选的备注名
	TicketV3 string `json:"ticket_v3" binding:"required"` // 来自 FriendRequestEvent 的 _ticket_v3
	Scene    int64  `json:"scene" binding:"required"`     // 来自 FriendRequestEvent 的 _scene
}

// CreateGroupParams 定义了 /create_group 接口的请求参数
type CreateGroupParams struct {
	GroupName string `json:"group_name" binding:"required"` // 要创建的分组名称
}

// DeleteGroupParams 定义了 /delete_group 接口的请求参数
type DeleteGroupParams struct {
	GroupID   string `json:"group_id,omitempty"`   // 要删除的分组ID (可选)
	GroupName string `json:"group_name,omitempty"` // 要删除的分组名称 (可选，如果 group_id 未提供)
}

// RenameGroupParams 定义了 /rename_group 接口的请求参数
type RenameGroupParams struct {
	GroupID string `json:"group_id" binding:"required"` // 要重命名的分组ID
	NewName string `json:"new_name" binding:"required"` // 分组的新名称
}

// GetGroupsParams (空结构体，因为 get_groups 不需要额外参数，但 ActionRequest 需要一个类型)
type GetGroupsParams struct{}

// AddMemberToGroupParams 定义了 /add_member_to_group 接口的请求参数
type AddMemberToGroupParams struct {
	GroupID  string `json:"group_id" binding:"required"`  // 目标分组ID
	MemberID string `json:"member_id" binding:"required"` // 要添加的成员ID (wxid 或 room_id)
}

// RemoveMemberFromGroupParams 定义了 /remove_member_from_group 接口的请求参数
type RemoveMemberFromGroupParams struct {
	GroupID  string `json:"group_id" binding:"required"`  // 目标分组ID
	MemberID string `json:"member_id" binding:"required"` // 要移除的成员ID (wxid 或 room_id)
}

// GetGroupMembersParams 定义了 /get_group_members 接口的请求参数
type GetGroupMembersParams struct {
	GroupID   string `form:"group_id" json:"group_id,omitempty"`     // <--- 修改：添加 form tag
	GroupName string `form:"group_name" json:"group_name,omitempty"` // <--- 修改：添加 form tag
}

// GetMemberGroupsParams 定义了 /get_member_groups 接口的请求参数
type GetMemberGroupsParams struct {
	MemberID string `form:"member_id" json:"member_id" binding:"required"` // <--- 修改：添加 form tag
}

// GetGroupIDByNameParams 定义了 /get_group_id_by_name 接口的请求参数
type GetGroupIDByNameParams struct {
	GroupName string `form:"group_name" json:"group_name" binding:"required"` // <--- 修改：添加 form tag
}

// GetFriendListParams (空结构体，因为 get_friend_list 不需要额外参数)
type GetFriendListParams struct{}

// GetGroupListParams (空结构体，因为 get_group_list 不需要额外参数)
type GetGroupListParams struct{}

// GetGHListParams (空结构体，因为 get_gh_list 不需要额外参数)
type GetGHListParams struct{}

// FriendInfo OneBot v12 好友信息结构
type FriendInfo struct {
	UserID   string `json:"user_id"`
	Nickname string `json:"nickname"`
	Remark   string `json:"remark,omitempty"` // 备注，可选
	// --- 微信特有字段 (自定义，非OneBot标准) ---
	Code   string `json:"_code,omitempty"`   // 微信号
	Gender int64  `json:"_gender,omitempty"` // 性别 (假设 GenderType 可以转换为 int)
}

// GroupInfo OneBot v12 群信息结构
type GroupInfo struct {
	GroupID     string `json:"group_id"`
	GroupName   string `json:"group_name"`
	MemberCount int    `json:"member_count"`
	Avatar      string `json:"avatar,omitempty"` // 群头像URL
}

// GHInfo OneBot v12 公众号信息结构 (类似 Guild)
type GHInfo struct {
	GuildID   string `json:"guild_id"`   // 使用 guild_id 作为公众号的唯一标识
	GuildName string `json:"guild_name"` // 使用 guild_name 作为公众号的名称
	// --- 微信特有字段 (自定义，非OneBot标准) ---
	Code   string `json:"_code,omitempty"`   // 公众号原始ID或微信号
	Gender int64  `json:"_gender,omitempty"` // 性别 (通常公众号无性别，但结构体有)
}

// HttpServer http 服务
type HttpServer struct {
	HttpAddr    string
	AccessToken string // 鉴权
	botCfg      *config.CommonConfig
	core        *coreapi.Core
}

const (
	failedStatus  = "failed"
	successStatus = "ok"
)

// Run HttpServer
func (s HttpServer) Run() {

	r := gin.Default()

	// 全局中间件
	r.Use(func(c *gin.Context) {
		start := time.Now()

		c.Next()

		duration := time.Since(start)
		logging.Info("Request details", map[string]interface{}{
			"latency": duration.String(),
			"method":  c.Request.Method,
			"path":    c.Request.URL.Path,
			"status":  c.Writer.Status(),
		})
	})
	r.Use(gin.Recovery())

	// 处理
	HttpApiGroup := r.Group("/")
	{
		HttpApiGroup.GET("/*filepath", s.globalHandler())
		HttpApiGroup.POST("/*filepath", s.globalHandler())
	}

	// 启动
	parsedURL, err := url.Parse(s.HttpAddr)
	if err != nil {
		logging.Debug("启动正向http致命错误!请检查地址是否正确", map[string]interface{}{"err": err})
		logging.Fatal("启动正向http致命错误!请检查地址是否正确", 4)
	}
	go func() {
		err = r.Run(parsedURL.Host)
		if err != nil {
			logging.Debug("启动正向http致命错误", map[string]interface{}{"err": err})
			logging.Fatal("启动正向http致命错误!", 4)
		}
	}()
	logging.Info(fmt.Sprintf("正向http启动成功,监听: %s 端口", s.HttpAddr))
}

func (s HttpServer) globalHandler() gin.HandlerFunc {
	return func(c *gin.Context) {
		accessToken := s.AccessToken
		if accessToken != "" {
			tokenHeader := strings.Replace(c.GetHeader("Authorization"), "Bearer ", "", 1)
			tokenQuery, _ := c.GetQuery("access_token")
			if (tokenHeader == "" || tokenHeader != accessToken) && (tokenQuery == "" || tokenQuery != accessToken) {
				logging.Warn("鉴权失败", map[string]interface{}{"path": c.Request.URL.Path, "reason": "token mismatch or missing"})
				c.JSON(http.StatusForbidden, gin.H{"error": "鉴权失败"})
				return
			}
		}

		logging.Debug("鉴权成功", map[string]interface{}{"path": c.Request.URL.Path})
		// 检查路径和处理对应的请求
		path := c.Request.URL.Path
		switch {
		case "/send_message" == path: // 发送消息
			s.handleSendMsg(c)
		case "/create_group" == path: // 新增：创建分组
			s.handleCreateGroup(c)
		case "/delete_group" == path: // 新增：删除分组
			s.handleDeleteGroup(c)
		case "/rename_group" == path: // 新增：重命名分组
			s.handleRenameGroup(c)
		case "/get_groups" == path: // 新增：获取所有分组
			s.handleGetGroups(c)
		case "/add_member_to_group" == path: // 新增：添加成员到分组
			s.handleAddMemberToGroup(c)
		case "/remove_member_from_group" == path: // 新增：从分组移除成员
			s.handleRemoveMemberFromGroup(c)
		case "/get_group_members" == path: // 新增：获取分组内成员
			s.handleGetGroupMembers(c)
		case "/get_member_groups" == path: // 新增：获取成员所在分组
			s.handleGetMemberGroups(c)
		case "/get_group_id_by_name" == path: // <--- 新增路由处理
			s.handleGetGroupIDByName(c)
		case "/handle_friend_request" == path: // 新增：处理好友请求
			s.handleFriendRequest(c)
		case "/get_friend_list" == path: // 新增：获取好友列表
			s.handleGetFriendList(c)
		case "/get_group_list" == path: // 新增：获取群列表
			s.handleGetGroupList(c)
		case "/get_gh_list" == path: // 新增：获取公众号列表 (作为 guild_list 的一种实现)
			s.handleGetGHList(c)
		//case "/login_callback" == path: // 获取登录回调
		case strings.HasPrefix(path, "/chat_image/"):
			s.handleChatImage(c, path)
		}

	}
}

func (s HttpServer) handleChatImage(c *gin.Context, path string) {
	// 处理 /chat_image/:date/:imgId 路径
	msgAttachIndex := strings.Index(path, "chat_image")
	relativePath := path[msgAttachIndex+len("chat_image"):]

	// 获取图片
	data := s.core.GetImgDataByPath(s.core.GetFullFilePathFromRelativePath(relativePath))
	if data == nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "image not found"})
		return
	}
	// 判断 图片类型
	fileType, err := imgutil.DetectFileType(data)
	if err != nil {
		logging.ErrorWithErr(err, "cannot detect file type of image", map[string]interface{}{"path": path})
		// 使用默认 图片类型 jpeg
		fileType = imgutil.JPEG
		//c.JSON(http.StatusInternalServerError, gin.H{"error": "cannot detect file type"})
	}
	c.Data(http.StatusOK, imgutil.GetMimeTypeByFileType(fileType), data)
}

//func (s HttpServer) handleLoginUrl(c *gin.Context) {
//	var req event.ActionRequest[any]
//	var resp event.ActionResponse
//	if c.Request.Method == http.MethodGet {
//		// 从URL查询解析参数
//
//		if err := c.ShouldBindQuery(&req); err != nil {
//			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
//			return
//		}
//	} else if c.Request.Method == http.MethodPost {
//		if err := c.ShouldBind(&req); err != nil {
//			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
//			return
//		}
//	}
//	logging.Debug("请求参数", map[string]interface{}{"action_request": req})
//	if req.Action != "login_callback" {
//		retErr(c, "/login_callback 端点只处理 action: login_callback",
//			oneboterr.UNSUPPORTED_ACTION, failedStatus)
//		return
//	}
//	var retData struct {
//		Type string `json:"type"`
//		Data string `json:"data"`
//	}
//	retData.Type = "url"
//	retData.Data = s.bot.GetloginUrl()
//	resp.Retcode = oneboterr.OK
//	resp.Status = successStatus
//	resp.Data = retData
//	respData, err := json.Marshal(resp)
//	if err != nil {
//		logging.Error("marshal response failed", map[string]interface{}{"err": err.Error()})
//		retErr(c, "marshal response failed", oneboterr.INTERNAL_HANDLER_ERROR, failedStatus)
//		return
//	}
//
//	c.Header("Content-Type", "application/json")
//	logging.Info(fmt.Sprintf("发送成功回执: %+v", string(respData)))
//	c.String(http.StatusOK, string(respData)) // 返回json字符串
//}

func (s HttpServer) handleSendMsg(c *gin.Context) {
	var req event.ActionRequest[event.SendMsgParams]
	var resp event.ActionResponse
	if c.Request.Method == http.MethodGet {
		if err := c.ShouldBindQuery(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
	} else if c.Request.Method == http.MethodPost {
		if err := c.ShouldBind(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
	}

	logging.Debug("请求参数", map[string]interface{}{"action_request": req})
	if req.Action != "send_message" {
		retErr(c, "/send_message 端点只处理 action: send_message",
			oneboterr.UNSUPPORTED_ACTION, failedStatus)
		return
	}

	params := req.Params
	virtualMessageId := uuid.NewString()

	// 获取随机延迟配置
	cfgDelayMin := 100 // 默认最小延迟 100ms
	cfgDelayMax := 300 // 默认最大延迟 300ms (示例值，原为1-3秒)

	if s.core != nil && s.botCfg != nil {
		// 假设配置中的单位是秒，我们需要转换为毫秒
		// 或者，如果配置已经是毫秒，则直接使用
		// 为了演示，我们假设配置是秒，并进行转换和限制
		cfgDelayMin = s.botCfg.AnswerDelayRandMin * 1000 // 秒转毫秒
		cfgDelayMax = s.botCfg.AnswerDelayRandMax * 1000 // 秒转毫秒
	}

	// 强制延迟上限为2000毫秒 (2秒)
	const maxAllowedDelayMs = 2000
	if cfgDelayMax > maxAllowedDelayMs {
		cfgDelayMax = maxAllowedDelayMs
		logging.Debug("Random send delay (max) capped at 2000ms", map[string]interface{}{"original_max_ms": s.botCfg.AnswerDelayRandMax * 1000})
	}

	// 确保 cfgDelayMin 不大于 cfgDelayMax，且不大于 maxAllowedDelayMs
	if cfgDelayMin > cfgDelayMax {
		cfgDelayMin = cfgDelayMax // 如果最小比最大还大，则设置为一样
	}
	if cfgDelayMin > maxAllowedDelayMs { // 进一步确保最小延迟也不超过上限
		cfgDelayMin = maxAllowedDelayMs
	}
	if cfgDelayMin < 0 { // 确保最小延迟不为负
		cfgDelayMin = 0
	}

	sentCount := 0
	var firstError error
	totalMessagesToSend := 0

	if len(params.GroupIDs) > 0 {
		// ... (GroupIDs 发送逻辑)
		uniqueReceivers := make(map[string]groupmanager.MemberType)
		for _, groupID := range params.GroupIDs {
			members, errManager := s.core.GroupManager.GetGroupMembers(groupID)
			if errManager != nil {
				logging.Warn("获取群组成员失败，跳过此群组", map[string]interface{}{"groupID": groupID, "error": errManager.Error()})
				continue
			}
			for _, member := range members {
				uniqueReceivers[member.ID] = member.Type
			}
		}

		if len(uniqueReceivers) == 0 {
			retErr(c, "未能从指定的群组标签中找到任何成员", oneboterr.BAD_PARAM, failedStatus)
			return
		}

		totalMessagesToSend = len(params.Message) * len(uniqueReceivers)
		var operationLastError error

		for receiverID, memberType := range uniqueReceivers {
			for i, segment := range params.Message {
				rnd := rand.New(rand.NewSource(time.Now().UnixNano()))
				var delayTimeMs int
				if cfgDelayMax >= cfgDelayMin {
					delayTimeMs = rnd.Intn(cfgDelayMax-cfgDelayMin+1) + cfgDelayMin
				} else {
					delayTimeMs = cfgDelayMin
					if delayTimeMs > maxAllowedDelayMs {
						delayTimeMs = maxAllowedDelayMs
					}
					if delayTimeMs < 0 {
						delayTimeMs = 0
					}
				}
				delay := time.Duration(delayTimeMs) * time.Millisecond
				time.Sleep(delay)
				logging.Debug("发送前的随机延迟", map[string]interface{}{"delay": delay.String(), "receiver": receiverID, "segment_index": i})

				msgType, msgData, convErr := convertOneBotMessageToRikka([]event.MessageSegment{segment})
				if convErr != nil {
					logging.ErrorWithErr(convErr, "消息段转换失败，无法向此成员发送此段", map[string]interface{}{"receiverID": receiverID, "segment_index": i, "segment": segment})
					if operationLastError == nil {
						operationLastError = convErr
					}
					continue
				}

				sendErr := s.core.SendMsg(msgType, msgData, receiverID)
				if sendErr != nil {
					logging.ErrorWithErr(sendErr, "向群组标签成员发送消息段失败", map[string]interface{}{
						"receiverID":    receiverID,
						"memberType":    string(memberType),
						"segment_index": i,
					})
					if operationLastError == nil {
						operationLastError = sendErr
					}
				} else {
					sentCount++
					logging.Info("已向群组标签成员发送消息段", map[string]interface{}{
						"receiverID":       receiverID,
						"memberType":       string(memberType),
						"segment_index":    i,
						"virtualMessageId": virtualMessageId,
					})
				}
			}
		}

		if sentCount == 0 && operationLastError != nil {
			retErr(c, fmt.Sprintf("未能向任何群组标签成员成功发送任何消息段: %v", operationLastError), oneboterr.API_SEND_FAIL, failedStatus)
			return
		}
		if sentCount < totalMessagesToSend && operationLastError != nil {
			resp.Message = fmt.Sprintf("部分消息段发送成功 (%d/%d)。最后遇到的错误: %v", sentCount, totalMessagesToSend, operationLastError)
		}

	} else {
		var targetID string
		if params.MessageType == event.OneBotMessageTypePrivate && params.UserId != "" {
			targetID = params.UserId
		} else if params.MessageType == event.OneBotMessageTypeGroup && params.GroupId != "" {
			targetID = params.GroupId
		} else if params.SendId != "" {
			targetID = params.SendId
		} else {
			retErr(c, "缺少必要的发送参数 (group_ids, 或 message_type + user_id/group_id, 或有效的 send_id)", oneboterr.BAD_PARAM, failedStatus)
			return
		}

		totalMessagesToSend = len(params.Message)
		if totalMessagesToSend == 0 {
			retErr(c, "消息数组为空，无可发送内容", oneboterr.BAD_PARAM, failedStatus)
			return
		}

		for i, segment := range params.Message {
			rnd := rand.New(rand.NewSource(time.Now().UnixNano()))
			var delayTimeMs int
			if cfgDelayMax >= cfgDelayMin {
				delayTimeMs = rnd.Intn(cfgDelayMax-cfgDelayMin+1) + cfgDelayMin
			} else {
				delayTimeMs = cfgDelayMin
				if delayTimeMs > maxAllowedDelayMs {
					delayTimeMs = maxAllowedDelayMs
				}
				if delayTimeMs < 0 {
					delayTimeMs = 0
				}
			}
			delay := time.Duration(delayTimeMs) * time.Millisecond
			time.Sleep(delay)
			logging.Debug("发送前的随机延迟", map[string]interface{}{"delay": delay.String(), "target": targetID, "segment_index": i})

			msgType, msgData, convErr := convertOneBotMessageToRikka([]event.MessageSegment{segment})
			if convErr != nil {
				logging.ErrorWithErr(convErr, "消息段转换失败", map[string]interface{}{"targetID": targetID, "segment_index": i, "segment": segment})
				if firstError == nil {
					firstError = convErr
				}
				continue
			}

			sendErr := s.core.SendMsg(msgType, msgData, targetID)
			if sendErr != nil {
				logging.ErrorWithErr(sendErr, "发送消息段失败", map[string]interface{}{"targetID": targetID, "segment_index": i})
				if firstError == nil {
					firstError = sendErr
				}
				continue
			} else {
				sentCount++
				logging.Info("消息段发送成功", map[string]interface{}{"targetID": targetID, "segment_index": i, "virtualMessageId": virtualMessageId})
			}
		}

		if sentCount == 0 && firstError != nil {
			retErr(c, fmt.Sprintf("未能成功发送任何消息段: %v", firstError), oneboterr.API_SEND_FAIL, failedStatus)
			return
		}
		if sentCount < totalMessagesToSend && firstError != nil {
			resp.Message = fmt.Sprintf("部分消息段发送成功 (%d/%d)。遇到的第一个错误: %v", sentCount, totalMessagesToSend, firstError)
		}
	}

	// ... (统一处理响应状态)
	resp.Status = successStatus
	resp.Retcode = oneboterr.OK
	resp.Data = event.MsgRespData{
		Time:      timeutil.GetTimeUnix(),
		MessageId: virtualMessageId,
	}

	if req.Echo != "" {
		resp.Echo = req.Echo
	}

	c.JSON(http.StatusOK, resp)
}

// convertOneBotMessageToRikka 将 OneBot 的 MessageSegment 数组转换为 RikkaBot.SendMsg 所需的类型和数据
// 修改：此函数现在被期望一次处理一个段（通过传入单元素切片），但其内部逻辑仍然是找到第一个就返回。
// 如果希望它能拼接文本或处理更复杂场景，其内部也需要修改。
// 目前，外部调用者通过循环并每次传递单元素切片来使用它。
func convertOneBotMessageToRikka(segments []event.MessageSegment) (message.MsgType, interface{}, error) {
	for _, seg := range segments { // 实际上因为外部调用方式，这个循环只会迭代一次
		if seg.Type == "text" && seg.Data != nil {
			if text, ok := seg.Data["text"]; ok {
				if textStr, okStr := text.(string); okStr {
					return message.MsgTypeText, textStr, nil
				}
			}
		}
		if seg.Type == "image" && seg.Data != nil {
			if url, ok := seg.Data["url"]; ok {
				if urlStr, okStr := url.(string); okStr {
					return message.MsgTypeImage, urlStr, nil
				}
			}
		}
		// 可以根据需要添加对其他类型 (如 at, reply 等) 的处理
		// 但请注意，如果这里返回错误，外部的循环会捕获它并跳过当前段
	}
	return 0, nil, fmt.Errorf("未能从提供的消息段中找到可处理的文本或图片内容，或者图片消息段缺少 'url' 字段")
}

// handleCreateGroup 处理创建分组的请求 (/create_group)
func (s *HttpServer) handleCreateGroup(c *gin.Context) {
	var req event.ActionRequest[CreateGroupParams]
	var resp event.ActionResponse

	if c.Request.Method != http.MethodPost {
		retErr(c, "/create_group endpoint only accepts POST requests", oneboterr.BAD_REQUEST, failedStatus)
		return
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		logging.Debug("参数绑定失败", map[string]interface{}{"err": err.Error()})
		retErr(c, fmt.Sprintf("参数绑定失败: %s。确保请求体是包含 'action' 和 'params': {'group_name': '名称'} 的 JSON。", err.Error()),
			oneboterr.BAD_PARAM, failedStatus)
		return
	}

	logging.Debug("创建分组请求参数", map[string]interface{}{"action_request": req})

	if req.Action != "create_group" {
		retErr(c, "/create_group 端点 action 必须是 'create_group'", oneboterr.UNSUPPORTED_ACTION, failedStatus)
		return
	}

	if req.Params.GroupName == "" {
		retErr(c, "参数 'group_name' 不能为空", oneboterr.BAD_PARAM, failedStatus)
		return
	}

	group, err := s.core.GroupManager.CreateGroup(req.Params.GroupName)
	if err != nil {
		logging.Error("创建分组失败", map[string]interface{}{"group_name": req.Params.GroupName, "err": err.Error()})
		if errors.Is(err, groupmanager.ErrGroupExistsWithName) {
			retErr(c, fmt.Sprintf("创建分组失败: 分组名称 '%s' 已存在", req.Params.GroupName), oneboterr.BAD_PARAM, failedStatus)
		} else {
			retErr(c, fmt.Sprintf("创建分组失败: %s", err.Error()), oneboterr.INTERNAL_HANDLER_ERROR, failedStatus)
		}
		return
	}

	resp.Echo = req.Echo
	resp.Retcode = oneboterr.OK
	resp.Status = successStatus
	resp.Data = group

	logging.Info("创建分组成功回执", map[string]interface{}{"response": resp})
	c.JSON(http.StatusOK, resp)
}

// handleGetGroups 处理获取所有分组的请求 (/get_groups)
func (s *HttpServer) handleGetGroups(c *gin.Context) {
	var req event.ActionRequest[GetGroupsParams]
	var resp event.ActionResponse

	if c.Request.Method == http.MethodGet {
		req.Action = "get_groups"
		if err := c.ShouldBindQuery(&req); err != nil {
			// Log or handle minor binding error if necessary, but continue for echo
		}
	} else if c.Request.Method == http.MethodPost {
		if err := c.ShouldBindJSON(&req); err != nil {
			retErr(c, fmt.Sprintf("参数绑定失败: %s", err.Error()), oneboterr.BAD_PARAM, failedStatus)
			return
		}
	} else {
		retErr(c, "/get_groups endpoint only accepts GET or POST requests", oneboterr.BAD_REQUEST, failedStatus)
		return
	}

	logging.Debug("获取所有分组请求", map[string]interface{}{"action_request": req})

	if req.Action != "get_groups" {
		retErr(c, "/get_groups 端点 action 必须是 'get_groups'", oneboterr.UNSUPPORTED_ACTION, failedStatus)
		return
	}

	groups, err := s.core.GroupManager.GetAllGroups()
	if err != nil {
		logging.Error("获取所有分组失败", map[string]interface{}{"err": err.Error()})
		retErr(c, fmt.Sprintf("获取所有分组失败: %s", err.Error()), oneboterr.INTERNAL_HANDLER_ERROR, failedStatus)
		return
	}

	resp.Echo = req.Echo
	resp.Retcode = oneboterr.OK
	resp.Status = successStatus
	resp.Data = groups

	logging.Info("获取所有分组成功回执", map[string]interface{}{"response": resp})
	c.JSON(http.StatusOK, resp)
}

// handleRenameGroup 处理重命名分组的请求 (/rename_group)
func (s *HttpServer) handleRenameGroup(c *gin.Context) {
	var req event.ActionRequest[RenameGroupParams]
	var resp event.ActionResponse

	if c.Request.Method != http.MethodPost {
		retErr(c, "/rename_group endpoint only accepts POST requests", oneboterr.BAD_REQUEST, failedStatus)
		return
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		retErr(c, fmt.Sprintf("参数绑定失败: %s", err.Error()), oneboterr.BAD_PARAM, failedStatus)
		return
	}

	logging.Debug("重命名分组请求参数", map[string]interface{}{"action_request": req})

	if req.Action != "rename_group" {
		retErr(c, "/rename_group 端点 action 必须是 'rename_group'", oneboterr.UNSUPPORTED_ACTION, failedStatus)
		return
	}

	err := s.core.GroupManager.RenameGroup(req.Params.GroupID, req.Params.NewName)
	if err != nil {
		logging.Error("重命名分组失败", map[string]interface{}{"params": req.Params, "err": err.Error()})
		if errors.Is(err, groupmanager.ErrGroupNotFound) {
			retErr(c, fmt.Sprintf("重命名分组失败: 分组 '%s' 未找到", req.Params.GroupID), oneboterr.BAD_PARAM, failedStatus)
		} else if errors.Is(err, groupmanager.ErrGroupExistsWithName) {
			retErr(c, fmt.Sprintf("重命名分组失败: 新名称 '%s' 已被其他分组使用", req.Params.NewName), oneboterr.BAD_PARAM, failedStatus)
		} else {
			retErr(c, fmt.Sprintf("重命名分组失败: %s", err.Error()), oneboterr.INTERNAL_HANDLER_ERROR, failedStatus)
		}
		return
	}

	resp.Echo = req.Echo
	resp.Retcode = oneboterr.OK
	resp.Status = successStatus
	resp.Data = gin.H{}

	logging.Info("重命名分组成功回执", map[string]interface{}{"response": resp})
	c.JSON(http.StatusOK, resp)
}

// handleDeleteGroup 处理删除分组的请求 (/delete_group)
func (s *HttpServer) handleDeleteGroup(c *gin.Context) {
	var req event.ActionRequest[DeleteGroupParams]
	var resp event.ActionResponse

	if c.Request.Method != http.MethodPost {
		retErr(c, "/delete_group endpoint only accepts POST requests", oneboterr.BAD_REQUEST, failedStatus)
		return
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		retErr(c, fmt.Sprintf("参数绑定失败: %s。确保请求体是包含 'action' 和 'params': {'group_id': 'id', 'group_name': '名称'} 的 JSON。", err.Error()),
			oneboterr.BAD_PARAM, failedStatus)
		return
	}

	logging.Debug("删除分组请求参数", map[string]interface{}{"action_request": req})

	if req.Action != "delete_group" {
		retErr(c, "/delete_group 端点 action 必须是 'delete_group'", oneboterr.UNSUPPORTED_ACTION, failedStatus)
		return
	}

	groupID := req.Params.GroupID
	groupName := req.Params.GroupName

	if groupID == "" && groupName == "" {
		retErr(c, "参数 'group_id' 或 'group_name' 必须提供一个", oneboterr.BAD_PARAM, failedStatus)
		return
	}

	if groupID == "" && groupName != "" {
		group, err := s.core.GroupManager.GetGroupByName(groupName)
		if err != nil {
			logging.Error("通过名称获取分组失败以便删除", map[string]interface{}{"group_name": groupName, "err": err.Error()})
			if errors.Is(err, groupmanager.ErrGroupNotFoundByName) {
				retErr(c, fmt.Sprintf("通过名称 '%s' 未找到分组", groupName), oneboterr.BAD_PARAM, failedStatus)
			} else {
				retErr(c, fmt.Sprintf("通过名称 '%s' 获取分组失败: %s", groupName, err.Error()), oneboterr.INTERNAL_HANDLER_ERROR, failedStatus)
			}
			return
		}
		if group == nil {
			retErr(c, fmt.Sprintf("通过名称 '%s' 未找到分组", groupName), oneboterr.BAD_PARAM, failedStatus)
			return
		}
		groupID = group.ID
	}

	err := s.core.GroupManager.DeleteGroup(groupID)
	if err != nil {
		logging.Error("删除分组失败", map[string]interface{}{"group_id": groupID, "err": err.Error()})
		if errors.Is(err, groupmanager.ErrGroupNotFound) {
			retErr(c, fmt.Sprintf("删除分组 '%s' 失败: 分组未找到", groupID), oneboterr.BAD_PARAM, failedStatus)
		} else {
			retErr(c, fmt.Sprintf("删除分组 '%s' 失败: %s", groupID, err.Error()), oneboterr.INTERNAL_HANDLER_ERROR, failedStatus)
		}
		return
	}

	resp.Echo = req.Echo
	resp.Retcode = oneboterr.OK
	resp.Status = successStatus
	resp.Data = gin.H{}

	logging.Info("删除分组成功回执", map[string]interface{}{"response": resp})
	c.JSON(http.StatusOK, resp)
}

// handleAddMemberToGroup 处理添加成员到分组的请求 (/add_member_to_group)
func (s *HttpServer) handleAddMemberToGroup(c *gin.Context) {
	var req event.ActionRequest[AddMemberToGroupParams]
	var resp event.ActionResponse

	if c.Request.Method != http.MethodPost {
		retErr(c, "/add_member_to_group endpoint only accepts POST requests", oneboterr.BAD_REQUEST, failedStatus)
		return
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		retErr(c, fmt.Sprintf("参数绑定失败: %s", err.Error()), oneboterr.BAD_PARAM, failedStatus)
		return
	}

	logging.Debug("添加成员到分组请求参数", map[string]interface{}{"action_request": req})

	if req.Action != "add_member_to_group" {
		retErr(c, "/add_member_to_group 端点 action 必须是 'add_member_to_group'", oneboterr.UNSUPPORTED_ACTION, failedStatus)
		return
	}

	params := req.Params
	err := s.core.GroupManager.AddMemberToGroup(params.GroupID, params.MemberID)
	if err != nil {
		logging.Error("添加成员到分组失败", map[string]interface{}{"params": params, "err": err.Error()})
		if errors.Is(err, groupmanager.ErrGroupNotFound) || errors.Is(err, groupmanager.ErrGroupDoesNotExist) {
			retErr(c, fmt.Sprintf("添加成员失败: 分组 '%s' 不存在", params.GroupID), oneboterr.BAD_PARAM, failedStatus)
		} else if errors.Is(err, groupmanager.ErrMemberAlreadyInGroup) {
			retErr(c, fmt.Sprintf("添加成员失败: 成员 '%s' 已在分组 '%s' 中", params.MemberID, params.GroupID), oneboterr.BAD_PARAM, failedStatus)
		} else if errors.Is(err, groupmanager.ErrMemberTypeInferenceFailed) {
			retErr(c, fmt.Sprintf("添加成员失败: 无法识别成员ID '%s' 的类型", params.MemberID), oneboterr.BAD_PARAM, failedStatus)
		} else {
			retErr(c, fmt.Sprintf("添加成员到分组失败: %s", err.Error()), oneboterr.INTERNAL_HANDLER_ERROR, failedStatus)
		}
		return
	}

	resp.Echo = req.Echo
	resp.Retcode = oneboterr.OK
	resp.Status = successStatus
	resp.Data = gin.H{}

	logging.Info("添加成员到分组成功回执", map[string]interface{}{"response": resp})
	c.JSON(http.StatusOK, resp)
}

// handleRemoveMemberFromGroup 处理从分组移除成员的请求 (/remove_member_from_group)
func (s *HttpServer) handleRemoveMemberFromGroup(c *gin.Context) {
	var req event.ActionRequest[RemoveMemberFromGroupParams]
	var resp event.ActionResponse

	if c.Request.Method != http.MethodPost {
		retErr(c, "/remove_member_from_group endpoint only accepts POST requests", oneboterr.BAD_REQUEST, failedStatus)
		return
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		retErr(c, fmt.Sprintf("参数绑定失败: %s", err.Error()), oneboterr.BAD_PARAM, failedStatus)
		return
	}

	logging.Debug("从分组移除成员请求参数", map[string]interface{}{"action_request": req})

	if req.Action != "remove_member_from_group" {
		retErr(c, "/remove_member_from_group 端点 action 必须是 'remove_member_from_group'", oneboterr.UNSUPPORTED_ACTION, failedStatus)
		return
	}

	params := req.Params
	err := s.core.GroupManager.RemoveMemberFromGroup(params.GroupID, params.MemberID)
	if err != nil {
		logging.Error("从分组移除成员失败", map[string]interface{}{"params": params, "err": err.Error()})
		if errors.Is(err, groupmanager.ErrGroupNotFound) || errors.Is(err, groupmanager.ErrGroupDoesNotExist) {
			retErr(c, fmt.Sprintf("移除成员失败: 分组 '%s' 不存在", params.GroupID), oneboterr.BAD_PARAM, failedStatus)
		} else if errors.Is(err, groupmanager.ErrMemberNotInGroup) {
			retErr(c, fmt.Sprintf("移除成员失败: 成员 '%s' 不在分组 '%s' 中", params.MemberID, params.GroupID), oneboterr.BAD_PARAM, failedStatus)
		} else if errors.Is(err, groupmanager.ErrMemberTypeInferenceFailed) {
			retErr(c, fmt.Sprintf("移除成员失败: 无法识别成员ID '%s' 的类型", params.MemberID), oneboterr.BAD_PARAM, failedStatus)
		} else {
			retErr(c, fmt.Sprintf("从分组移除成员失败: %s", err.Error()), oneboterr.INTERNAL_HANDLER_ERROR, failedStatus)
		}
		return
	}

	resp.Echo = req.Echo
	resp.Retcode = oneboterr.OK
	resp.Status = successStatus
	resp.Data = gin.H{}

	logging.Info("从分组移除成员成功回执", map[string]interface{}{"response": resp})
	c.JSON(http.StatusOK, resp)
}

// handleGetGroupMembers 处理获取分组内成员的请求 (/get_group_members)
func (s *HttpServer) handleGetGroupMembers(c *gin.Context) {
	var req event.ActionRequest[GetGroupMembersParams]
	var resp event.ActionResponse

	if c.Request.Method == http.MethodGet {
		req.Action = "get_group_members"
		if err := c.ShouldBindQuery(&req.Params); err != nil {
			retErr(c, fmt.Sprintf("GET 请求参数绑定失败: %s", err.Error()), oneboterr.BAD_PARAM, failedStatus)
			return
		}
		_ = c.ShouldBindQuery(&req)
	} else if c.Request.Method == http.MethodPost {
		if err := c.ShouldBindJSON(&req); err != nil {
			retErr(c, fmt.Sprintf("POST 请求参数绑定失败: %s", err.Error()), oneboterr.BAD_PARAM, failedStatus)
			return
		}
	} else {
		retErr(c, "/get_group_members endpoint only accepts GET or POST requests", oneboterr.BAD_REQUEST, failedStatus)
		return
	}

	logging.Debug("获取分组内成员请求参数", map[string]interface{}{"action_request": req})

	if req.Action != "get_group_members" {
		retErr(c, "/get_group_members 端点 action 必须是 'get_group_members'", oneboterr.UNSUPPORTED_ACTION, failedStatus)
		return
	}

	groupID := req.Params.GroupID
	groupName := req.Params.GroupName

	if groupID == "" && groupName == "" {
		retErr(c, "参数 'group_id' 或 'group_name' 必须提供一个", oneboterr.BAD_PARAM, failedStatus)
		return
	}

	if groupID == "" && groupName != "" {
		group, err := s.core.GroupManager.GetGroupByName(groupName)
		if err != nil {
			logging.Error("通过名称获取分组失败 (GetGroupMembers)", map[string]interface{}{"group_name": groupName, "err": err.Error()})
			if errors.Is(err, groupmanager.ErrGroupNotFoundByName) {
				retErr(c, fmt.Sprintf("通过名称 '%s' 未找到分组", groupName), oneboterr.BAD_PARAM, failedStatus)
			} else {
				retErr(c, fmt.Sprintf("通过名称 '%s' 获取分组失败: %s", groupName, err.Error()), oneboterr.INTERNAL_HANDLER_ERROR, failedStatus)
			}
			return
		}
		if group == nil {
			retErr(c, fmt.Sprintf("通过名称 '%s' 未找到分组", groupName), oneboterr.BAD_PARAM, failedStatus)
			return
		}
		groupID = group.ID
	}

	members, err := s.core.GroupManager.GetGroupMembers(groupID)
	if err != nil {
		logging.Error("获取分组内成员失败", map[string]interface{}{"group_id": groupID, "err": err.Error()})
		retErr(c, fmt.Sprintf("获取分组 '%s' 内成员失败: %s", groupID, err.Error()), oneboterr.INTERNAL_HANDLER_ERROR, failedStatus)
		return
	}

	resp.Echo = req.Echo
	resp.Retcode = oneboterr.OK
	resp.Status = successStatus
	resp.Data = members

	logging.Info("获取分组内成员成功回执", map[string]interface{}{"response": resp})
	c.JSON(http.StatusOK, resp)
}

// handleGetMemberGroups 处理获取成员所在分组的请求 (/get_member_groups)
func (s *HttpServer) handleGetMemberGroups(c *gin.Context) {
	var req event.ActionRequest[GetMemberGroupsParams]
	var resp event.ActionResponse

	if c.Request.Method == http.MethodGet {
		req.Action = "get_member_groups"
		if err := c.ShouldBindQuery(&req.Params); err != nil {
			retErr(c, fmt.Sprintf("GET 请求参数绑定失败: %s", err.Error()), oneboterr.BAD_PARAM, failedStatus)
			return
		}
		_ = c.ShouldBindQuery(&req)
	} else if c.Request.Method == http.MethodPost {
		if err := c.ShouldBindJSON(&req); err != nil {
			retErr(c, fmt.Sprintf("POST 请求参数绑定失败: %s", err.Error()), oneboterr.BAD_PARAM, failedStatus)
			return
		}
	} else {
		retErr(c, "/get_member_groups endpoint only accepts GET or POST requests", oneboterr.BAD_REQUEST, failedStatus)
		return
	}

	logging.Debug("获取成员所在分组请求参数", map[string]interface{}{"action_request": req})

	if req.Action != "get_member_groups" {
		retErr(c, "/get_member_groups 端点 action 必须是 'get_member_groups'", oneboterr.UNSUPPORTED_ACTION, failedStatus)
		return
	}

	params := req.Params
	groups, err := s.core.GroupManager.GetMemberGroups(params.MemberID)
	if err != nil {
		logging.Error("获取成员所在分组失败", map[string]interface{}{"member_id": params.MemberID, "err": err.Error()})
		if errors.Is(err, groupmanager.ErrMemberTypeInferenceFailed) {
			retErr(c, fmt.Sprintf("获取成员分组失败: 无法识别成员ID '%s' 的类型", params.MemberID), oneboterr.BAD_PARAM, failedStatus)
		} else {
			retErr(c, fmt.Sprintf("获取成员 '%s' 所在分组失败: %s", params.MemberID, err.Error()), oneboterr.INTERNAL_HANDLER_ERROR, failedStatus)
		}
		return
	}

	resp.Echo = req.Echo
	resp.Retcode = oneboterr.OK
	resp.Status = successStatus
	resp.Data = groups

	logging.Info("获取成员所在分组成功回执", map[string]interface{}{"response": resp})
	c.JSON(http.StatusOK, resp)
}

// handleGetGroupIDByName 处理通过分组名称获取分组ID的请求 (/get_group_id_by_name)
func (s *HttpServer) handleGetGroupIDByName(c *gin.Context) {
	var req event.ActionRequest[GetGroupIDByNameParams]
	var resp event.ActionResponse

	if c.Request.Method == http.MethodGet {
		req.Action = "get_group_id_by_name" // 假设 action 名称
		if err := c.ShouldBindQuery(&req.Params); err != nil {
			retErr(c, fmt.Sprintf("GET 请求参数绑定失败: %s。确保提供了 'group_name'。", err.Error()), oneboterr.BAD_PARAM, failedStatus)
			return
		}
		_ = c.ShouldBindQuery(&req) // 绑定 echo 等公共字段
	} else if c.Request.Method == http.MethodPost {
		if err := c.ShouldBindJSON(&req); err != nil {
			retErr(c, fmt.Sprintf("POST 请求参数绑定失败: %s。确保请求体包含 'action' 和 'params': {'group_name': '名称'}", err.Error()), oneboterr.BAD_PARAM, failedStatus)
			return
		}
	} else {
		retErr(c, "/get_group_id_by_name endpoint only accepts GET or POST requests", oneboterr.BAD_REQUEST, failedStatus)
		return
	}

	logging.Debug("通过分组名获取ID请求参数", map[string]interface{}{"action_request": req})

	// 虽然上面已经尝试绑定，但 action 检查还是需要的，以确保请求意图明确
	if req.Action != "get_group_id_by_name" && req.Action != "" { // 允许 action 为空，如果直接通过路径调用
		retErr(c, "/get_group_id_by_name 端点 action 必须是 'get_group_id_by_name' (或通过路径直接调用时可省略)", oneboterr.UNSUPPORTED_ACTION, failedStatus)
		return
	}

	if req.Params.GroupName == "" {
		retErr(c, "参数 'group_name' 不能为空", oneboterr.BAD_PARAM, failedStatus)
		return
	}

	group, err := s.core.GroupManager.GetGroupByName(req.Params.GroupName)
	if err != nil {
		logging.Warn("通过名称获取分组失败 (GetGroupIDByName)", map[string]interface{}{"group_name": req.Params.GroupName, "err": err.Error()})
		if errors.Is(err, groupmanager.ErrGroupNotFoundByName) {
			retErr(c, fmt.Sprintf("通过名称 '%s' 未找到分组", req.Params.GroupName), oneboterr.BAD_PARAM, failedStatus) // 或更具体的 not_found code
		} else {
			retErr(c, fmt.Sprintf("通过名称 '%s' 获取分组时发生内部错误: %s", req.Params.GroupName, err.Error()), oneboterr.INTERNAL_HANDLER_ERROR, failedStatus)
		}
		return
	}
	if group == nil { // 再次确认，虽然 GetGroupByName 在未找到时应该返回 ErrGroupNotFoundByName
		retErr(c, fmt.Sprintf("通过名称 '%s' 未找到分组 (group is nil)", req.Params.GroupName), oneboterr.BAD_PARAM, failedStatus)
		return
	}

	resp.Echo = req.Echo
	resp.Retcode = oneboterr.OK
	resp.Status = successStatus
	resp.Data = gin.H{ // 返回包含 group_id 的对象
		"group_id": group.ID,
	}

	logging.Info("通过分组名获取ID成功回执", map[string]interface{}{"response": resp})
	c.JSON(http.StatusOK, resp)
}

// handleFriendRequest 处理好友请求的操作 (/handle_friend_request)
func (s *HttpServer) handleFriendRequest(c *gin.Context) {
	var req event.ActionRequest[HandleFriendRequestParams]
	var resp event.ActionResponse

	if c.Request.Method != http.MethodPost {
		retErr(c, "/handle_friend_request endpoint only accepts POST requests", oneboterr.BAD_REQUEST, failedStatus)
		return
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		logging.Debug("处理好友请求参数绑定失败", map[string]interface{}{"err": err.Error()})
		retErr(c, fmt.Sprintf("参数绑定失败: %s. 确保请求体是包含 'action' 和 'params': {...} 的 JSON。", err.Error()),
			oneboterr.BAD_PARAM, failedStatus)
		return
	}

	logging.Debug("处理好友请求参数", map[string]interface{}{"action_request": req})

	if req.Action != "handle_friend_request" { // OneBot v12 风格的动作名，可自定义
		retErr(c, "/handle_friend_request 端点 action 必须是 'handle_friend_request'", oneboterr.UNSUPPORTED_ACTION, failedStatus)
		return
	}

	params := req.Params

	// 调用 RikkaBot 核心逻辑处理好友请求
	// 假设 RikkaBot 实例有一个方法如 ProcessFriendRequest
	// err := s.bot.ProcessFriendRequest(params.Flag, params.Approve, params.Remark, params.TicketV3, params.Scene)
	// 由于 RikkaBot 的具体方法未知，这里暂时模拟 WCF SDK 调用
	// 实际应通过 s.bot 抽象层调用

	if params.Approve {
		logging.Info("尝试同意好友请求", map[string]interface{}{"flag(v4)": params.Flag, "v3": params.TicketV3, "scene": params.Scene, "remark": params.Remark})

		b := s.core.AcceptNewFriend(wcf.NewFriendReq{
			V3:    params.TicketV3,
			V4:    params.Flag,
			Scene: params.Scene,
		}) // 假设 RikkaBot 有此方法直接调用 WCF
		if !b {
			logging.Error("同意好友请求失败 (SDK调用)", map[string]interface{}{"params": params})
			retErr(c, fmt.Sprintf("同意好友请求失败: %v", b), oneboterr.API_SEND_FAIL, failedStatus) // 或者更具体的错误码
			return
		}
		logging.Info("好友请求已同意", map[string]interface{}{"flag": params.Flag})
	} else {
		//logging.Info("尝试拒绝好友请求", map[string]interface{}{"flag(v4)": params.Flag, "v3": params.TicketV3, "scene": params.Scene})
		// todo 尚未实现拒绝 临时 忽略好友请求
		logging.Info("好友请求已忽略", map[string]interface{}{"flag": params.Flag})
	}

	resp.Echo = req.Echo
	resp.Retcode = oneboterr.OK
	resp.Status = successStatus
	resp.Data = gin.H{} // 成功时通常不返回特定数据，或返回空对象

	logging.Info("处理好友请求成功回执", map[string]interface{}{"response": resp})
	c.JSON(http.StatusOK, resp)
}

// handleGetFriendList 处理获取好友列表的请求
func (s *HttpServer) handleGetFriendList(c *gin.Context) {
	var req event.ActionRequest[GetFriendListParams]
	var resp event.ActionResponse

	if err := c.ShouldBindJSON(&req); err != nil && err != io.EOF { // 允许空 body
		logging.Debug("获取好友列表参数绑定失败", map[string]interface{}{"err": err.Error()})
		retErr(c, fmt.Sprintf("参数绑定失败: %s. 确保请求体是包含 'action' 和 'params': {...} 的 JSON，或者为空。", err.Error()),
			oneboterr.BAD_PARAM, failedStatus)
		return
	}
	if req.Action != "get_friend_list" && req.Action != "" { // 兼容直接调用和通过 action 调用
		retErr(c, "/get_friend_list 端点 action 必须是 'get_friend_list' 或为空", oneboterr.UNSUPPORTED_ACTION, failedStatus)
		return
	}

	friends, ok := s.core.GetCtFriends()
	if !ok {
		retErr(c, "获取好友列表失败 (SDK 调用失败)", oneboterr.API_SEND_FAIL, failedStatus)
		return
	}

	onebotFriends := make([]FriendInfo, 0, len(friends))
	for _, f := range friends {
		onebotFriends = append(onebotFriends, FriendInfo{
			UserID:   f.Wxid,
			Nickname: f.Name,
			Remark:   f.Remark,
			Code:     f.Code,
			Gender:   int64(f.Gender), // 假设 f.Gender 是 wcf.GenderType，可以转换为 int64
		})
	}

	resp.Echo = req.Echo
	resp.Retcode = oneboterr.OK
	resp.Status = successStatus
	resp.Data = onebotFriends
	c.JSON(http.StatusOK, resp)
}

// handleGetGroupList 处理获取群列表的请求
func (s *HttpServer) handleGetGroupList(c *gin.Context) {
	var req event.ActionRequest[GetGroupListParams]
	var resp event.ActionResponse

	if err := c.ShouldBindJSON(&req); err != nil && err != io.EOF {
		logging.Debug("获取群列表参数绑定失败", map[string]interface{}{"err": err.Error()})
		retErr(c, fmt.Sprintf("参数绑定失败: %s. 确保请求体是包含 'action' 和 'params': {...} 的 JSON，或者为空。", err.Error()),
			oneboterr.BAD_PARAM, failedStatus)
		return
	}
	if req.Action != "get_group_list" && req.Action != "" {
		retErr(c, "/get_group_list 端点 action 必须是 'get_group_list' 或为空", oneboterr.UNSUPPORTED_ACTION, failedStatus)
		return
	}

	groups, ok := s.core.GetCtChatRooms()
	if !ok {
		retErr(c, "获取群列表失败 (SDK 调用失败)", oneboterr.API_SEND_FAIL, failedStatus)
		return
	}

	onebotGroups := make([]GroupInfo, 0, len(groups))
	for _, g := range groups {
		memberCount := 0
		if g.RoomData != nil && g.RoomData.Members != nil {
			memberCount = len(g.RoomData.Members)
		}
		avatarURL := ""
		if g.RoomHeadImgURL != nil {
			avatarURL = *g.RoomHeadImgURL
		}
		onebotGroups = append(onebotGroups, GroupInfo{
			GroupID:     g.RoomID,
			GroupName:   g.Name, // 群名来自 User 嵌套结构
			MemberCount: memberCount,
			Avatar:      avatarURL,
		})
	}

	resp.Echo = req.Echo
	resp.Retcode = oneboterr.OK
	resp.Status = successStatus
	resp.Data = onebotGroups
	c.JSON(http.StatusOK, resp)
}

// handleGetGHList 处理获取公众号列表的请求 (作为 get_guild_list 的一种实现)
func (s *HttpServer) handleGetGHList(c *gin.Context) {
	var req event.ActionRequest[GetGHListParams]
	var resp event.ActionResponse

	if err := c.ShouldBindJSON(&req); err != nil && err != io.EOF {
		logging.Debug("获取公众号列表参数绑定失败", map[string]interface{}{"err": err.Error()})
		retErr(c, fmt.Sprintf("参数绑定失败: %s. 确保请求体是包含 'action' 和 'params': {...} 的 JSON，或者为空。", err.Error()),
			oneboterr.BAD_PARAM, failedStatus)
		return
	}
	if req.Action != "get_gh_list" && req.Action != "get_guild_list" && req.Action != "" { // 兼容 get_gh_list 和 get_guild_list
		retErr(c, "/get_gh_list (或 /get_guild_list) 端点 action 必须是 'get_gh_list', 'get_guild_list' 或为空", oneboterr.UNSUPPORTED_ACTION, failedStatus)
		return
	}

	ghs, ok := s.core.GetCtGHs()
	if !ok {
		retErr(c, "获取公众号列表失败 (SDK 调用失败)", oneboterr.API_SEND_FAIL, failedStatus)
		return
	}

	onebotGHs := make([]GHInfo, 0, len(ghs))
	for _, gh := range ghs {
		onebotGHs = append(onebotGHs, GHInfo{
			GuildID:   gh.Wxid, // 使用 Wxid 作为公众号的唯一标识
			GuildName: gh.Name,
			Code:      gh.Code,
			Gender:    int64(gh.Gender), // 假设 gh.Gender 是 wcf.GenderType
		})
	}

	resp.Echo = req.Echo
	resp.Retcode = oneboterr.OK
	resp.Status = successStatus
	resp.Data = onebotGHs
	c.JSON(http.StatusOK, resp)
}

// 处理错误并返回 json
func retErr(c *gin.Context, errMsg string, retcode int64, status string) {
	var resp event.ActionResponse
	resp.Message = errMsg
	resp.Retcode = retcode
	resp.Status = status
	c.JSON(http.StatusOK, resp)
}

// HttpClient 反向http
type HttpClient struct {
	secret     string
	postUrl    string
	timeout    int
	client     *http.Client
	MaxRetries int
	InitDelay  time.Duration // 最初的重试间隔
	MaxDelay   time.Duration // 最大重试间隔
	bot        *rikkabot.RikkaBot
}

// RunHttp 启动 http、http上报器
func RunHttp(rbot *rikkabot.RikkaBot) {
	httpserverCfg := rbot.Config.HttpServer
	// http server
	HttpServer{
		HttpAddr:    httpserverCfg.HttpAddress,
		AccessToken: httpserverCfg.AccessToken,
		core:        coreapi.GetCore(),
	}.Run()

	// http上报器
	for _, post := range rbot.Config.HttpPost {
		HttpClient{
			secret:     post.Secret,
			postUrl:    post.Url,
			timeout:    post.TimeOut,
			MaxRetries: post.MaxRetries,
			InitDelay:  1500 * time.Millisecond,
			MaxDelay:   8 * time.Second,
			bot:        rbot,
		}.Run()
	}
	HandlerHeartBeat(rbot) // 处理心跳 推送心跳事件
}

func (c HttpClient) Run() {
	if c.timeout < 5 {
		c.timeout = 5
	}

	c.client = &http.Client{
		Timeout: time.Duration(c.timeout) * time.Second,
	}
	logging.Info("Http Post 上报器已启动！", map[string]interface{}{"url": c.postUrl})
	// 注册事件处理
	c.bot.OnEventPush(c.HandlerPostEvent)
}

// HandlerHeartBeat 心跳事件
func HandlerHeartBeat(bot *rikkabot.RikkaBot) {
	cfg := bot.Config
	if !cfg.EnableHeartBeat {
		logging.Warn("警告: 心跳功能已关闭，若非预期，请检查配置文件。")
		return
	}
	go func() {
		t := time.NewTicker(time.Duration(cfg.Interval) * time.Second)
		for {
			<-t.C
			err := bot.EventPool.AddEvent(event.HeartBeatEvent{
				Event: event.Event{
					Id:         uuid.New().String(),
					Time:       timeutil.GetTimeUnix(),
					Type:       "meta",
					DetailType: "heart_beat",
				},
				Interval: cfg.Interval,
			})
			if err != nil {
				logging.Warn("添加心跳事件到事件池失败", map[string]interface{}{"error": err.Error()})
			}
		}
	}()
	logging.Info(fmt.Sprintf("心跳事件启动！间隔：%d", cfg.Interval))
}

// HandlerPostEvent 处理 post 事件
func (c HttpClient) HandlerPostEvent(event event.IEvent) {
	var err error
	eventJSON, err := json.Marshal(event)
	if err != nil {
		logging.ErrorWithErr(err, "marshal event failed", map[string]interface{}{"event_type": fmt.Sprintf("%T", event)})
	}
	var req *http.Request
	var resp *http.Response
	for i := 0; i <= c.MaxRetries; i++ {
		req, err = http.NewRequest("POST", c.postUrl, bytes.NewBuffer(eventJSON))
		if err != nil {
			logHttpPostError(event, err, "request create failed")
			c.bot.ExitWithErr(1102, err.Error())
		}
		req.Header.Set("Content-Type", "application/json")
		//req.Header.Set("Authorization", "Bearer "+encrypt(c.secret))
		req.Header.Set("Authorization", "Bearer "+c.secret)

		resp, err = c.client.Do(req) // nolint:bodyclose
		if err == nil && resp != nil && resp.StatusCode == 200 {
			break
		}
		if i < c.MaxRetries {
			logging.Warn(fmt.Sprintf("上报 Event 数据到 %v 失败， 将进行第 %d 次重试", c.postUrl, i+1),
				map[string]interface{}{"error": err, "retry_count": i + 1})
		} else {
			logging.Warn(fmt.Sprintf("上报 Event 到 %v 失败, 停止上报：已达重试上限", c.postUrl),
				map[string]interface{}{"error": err, "event_type": fmt.Sprintf("%T", event)})
			return
		}
		delay := c.InitDelay << i
		if delay > c.MaxDelay {
			delay = c.MaxDelay
		}
		// 添加随机抖动
		jitter := time.Duration(rand.Int63n(int64(delay) / 2))
		delay += jitter
		time.Sleep(delay)
	}
	defer resp.Body.Close()

	logging.Debug("上报Event数据到 "+c.postUrl, map[string]interface{}{"event_type": fmt.Sprintf("%T", event)})
	if resp.Body == nil {
		logging.Warn("返回Body数据为空", map[string]interface{}{"url": c.postUrl, "status_code": resp.StatusCode})
		return
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		logHttpPostError(event, err, "response body read failed")
	}
	if resp.StatusCode != 200 {
		logHttpPostError(event, nil, fmt.Sprintf("response status code not 200: %d", resp.StatusCode))
	}

	logging.Debug("response body: "+string(body), map[string]interface{}{"event_type": fmt.Sprintf("%T", event)})
}

func logHttpPostError(event event.IEvent, err error, msg string) {
	var wrappedError error
	if err != nil {
		wrappedError = fmt.Errorf("%w: %w", oneboterr.ErrHttpPost, err)
	} else {
		wrappedError = oneboterr.ErrHttpPost
	}
	logging.ErrorWithErr(wrappedError, msg, map[string]interface{}{"event_type": fmt.Sprintf("%T", event)})
}

// encrypt http post 加密 secret
func encrypt(secret string) string {
	key := []byte(secret)
	hash := sha256.New()
	hash.Write(key)
	secreted := hash.Sum(nil)
	return hex.EncodeToString(secreted)
}
