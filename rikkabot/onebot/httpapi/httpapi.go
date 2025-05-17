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
	"io"
	"math/rand"
	"net/http"
	"net/url"
	"strings"
	"time"

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
	GroupID   string `json:"group_id,omitempty"`   // 目标分组ID (可选)
	GroupName string `json:"group_name,omitempty"` // 目标分组名称 (可选，如果 group_id 未提供)
}

// GetMemberGroupsParams 定义了 /get_member_groups 接口的请求参数
type GetMemberGroupsParams struct {
	MemberID string `json:"member_id" binding:"required"` // 成员ID (wxid 或 room_id)
}

// HttpServer http 服务
type HttpServer struct {
	HttpAddr    string
	AccessToken string // 鉴权
	bot         *rikkabot.RikkaBot
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
	data := s.bot.GetImgDataByPath(s.bot.GetFullFilePathFromRelativePath(relativePath))
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
	var err error
	// 注意: RikkaBot.SendMsg 不返回 messageId，我们需要生成一个或处理缺失的情况
	// 对于群组标签发送，我们将为每个成功发送的消息记录日志，但只返回第一个虚拟ID（或特定策略）
	virtualMessageId := uuid.NewString() // 为整个操作生成一个主虚拟ID

	if len(params.GroupIDs) > 0 {
		uniqueReceivers := make(map[string]groupmanager.MemberType)
		for _, groupID := range params.GroupIDs {
			members, errManager := s.bot.GroupManager.GetGroupMembers(groupID)
			if errManager != nil {
				logging.Warn("获取群组成员失败，跳过此群组", map[string]interface{}{"groupID": groupID, "error": errManager.Error()})
				continue
			}
			for _, member := range members {
				uniqueReceivers[member.ID] = member.Type // 使用 groupmanager 返回的类型
			}
		}

		if len(uniqueReceivers) == 0 {
			retErr(c, "未能从指定的群组标签中找到任何成员", oneboterr.BAD_PARAM, failedStatus)
			return
		}

		var lastError error
		sentCount := 0

		for receiverID, memberType := range uniqueReceivers {
			// 将 OneBot Message (segment array) 转换为 RikkaBot.SendMsg 所需的 msgType 和 data
			msgType, msgData, convErr := convertOneBotMessageToRikka(params.Message)
			if convErr != nil {
				logging.ErrorWithErr(convErr, "消息转换失败，无法向此成员发送", map[string]interface{}{"receiverID": receiverID})
				lastError = convErr // 记录转换错误
				continue
			}

			// 注意: RikkaBot.SendMsg 的 sendId 参数不区分用户或群组，它就是目标ID。
			// ContactType (friend/group) 的区分是在 RikkaBot.SendMessage 中，但我们现在用 SendMsg。
			// SendMsg 内部可能需要根据 sendId 的格式来判断是发给个人还是群，或者 wcf SDK 会处理。
			sendErr := s.bot.SendMsg(msgType, msgData, receiverID)
			if sendErr != nil {
				logging.ErrorWithErr(sendErr, "向群组标签成员发送消息失败", map[string]interface{}{
					"receiverID": receiverID,
					"memberType": string(memberType),
				})
				lastError = sendErr
			} else {
				sentCount++
				logging.Info("已向群组标签成员发送消息", map[string]interface{}{
					"receiverID":       receiverID,
					"memberType":       string(memberType),
					"virtualMessageId": virtualMessageId,
				})
			}
		}

		if sentCount == 0 && lastError != nil {
			retErr(c, fmt.Sprintf("未能向任何群组标签成员成功发送消息: %v", lastError), oneboterr.API_SEND_FAIL, failedStatus)
			return
		}
		if sentCount < len(uniqueReceivers) && lastError != nil {
			resp.Status = successStatus
			resp.Retcode = oneboterr.OK
			resp.Message = fmt.Sprintf("部分消息发送成功 (%d/%d)。最后遇到的错误: %v", sentCount, len(uniqueReceivers), lastError)
			resp.Data = event.MsgRespData{
				Time:      timeutil.GetTimeUnix(),
				MessageId: virtualMessageId,
			}
		} else {
			resp.Status = successStatus
			resp.Retcode = oneboterr.OK
			resp.Data = event.MsgRespData{
				Time:      timeutil.GetTimeUnix(),
				MessageId: virtualMessageId,
			}
		}

	} else { // 单独发送逻辑 (非 GroupIDs)
		var targetID string

		if params.MessageType == event.OneBotMessageTypePrivate && params.UserId != "" {
			targetID = params.UserId
		} else if params.MessageType == event.OneBotMessageTypeGroup && params.GroupId != "" {
			targetID = params.GroupId
		} else if params.SendId != "" {
			targetID = params.SendId
			// 对于 SendId，其类型（用户或群组）将由底层的 s.bot.SendMsg 或 wcf SDK 自行处理或推断
		} else {
			retErr(c, "缺少必要的发送参数 (group_ids, 或 message_type + user_id/group_id, 或有效的 send_id)", oneboterr.BAD_PARAM, failedStatus)
			return
		}

		msgType, msgData, convErr := convertOneBotMessageToRikka(params.Message)
		if convErr != nil {
			retErr(c, fmt.Sprintf("消息内容转换失败: %v", convErr), oneboterr.BAD_PARAM, failedStatus)
			return
		}

		err = s.bot.SendMsg(msgType, msgData, targetID)
		if err != nil {
			logging.Error("发送消息失败", map[string]interface{}{"targetID": targetID, "err": err.Error()})
			retErr(c, err.Error(), oneboterr.API_SEND_FAIL, failedStatus)
			return
		}

		resp.Status = successStatus
		resp.Retcode = oneboterr.OK
		resp.Data = event.MsgRespData{
			Time:      timeutil.GetTimeUnix(),
			MessageId: virtualMessageId,
		}
	}

	if req.Echo != "" {
		resp.Echo = req.Echo
	}

	c.JSON(http.StatusOK, resp)
}

// convertOneBotMessageToRikka 将 OneBot 的 MessageSegment 数组转换为 RikkaBot.SendMsg 所需的类型和数据
// 简化处理：优先取第一个 text 或 image 类型的 segment
func convertOneBotMessageToRikka(segments []event.MessageSegment) (message.MsgType, interface{}, error) {
	for _, seg := range segments {
		if seg.Type == "text" && seg.Data != nil {
			if text, ok := seg.Data["text"]; ok {
				if textStr, okStr := text.(string); okStr {
					return message.MsgTypeText, textStr, nil
				}
			}
		}
		if seg.Type == "image" && seg.Data != nil {
			// 仅支持通过 url 字段获取图片
			if url, ok := seg.Data["url"]; ok {
				if urlStr, okStr := url.(string); okStr {
					return message.MsgTypeImage, urlStr, nil
				}
			}
		}
		// 可以根据需要添加对其他类型 (如 at, reply 等) 的处理
	}
	return 0, nil, fmt.Errorf("未能从 OneBot Message 中找到可处理的文本或图片内容，或者图片消息段缺少 'url' 字段")
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

	group, err := s.bot.GroupManager.CreateGroup(req.Params.GroupName)
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

	groups, err := s.bot.GroupManager.GetAllGroups()
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

	err := s.bot.GroupManager.RenameGroup(req.Params.GroupID, req.Params.NewName)
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
		group, err := s.bot.GroupManager.GetGroupByName(groupName)
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

	err := s.bot.GroupManager.DeleteGroup(groupID)
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
	err := s.bot.GroupManager.AddMemberToGroup(params.GroupID, params.MemberID)
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
	err := s.bot.GroupManager.RemoveMemberFromGroup(params.GroupID, params.MemberID)
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
		group, err := s.bot.GroupManager.GetGroupByName(groupName)
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

	members, err := s.bot.GroupManager.GetGroupMembers(groupID)
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
	groups, err := s.bot.GroupManager.GetMemberGroups(params.MemberID)
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
		bot:         rbot,
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
		req.Header.Set("Authorization", "Bearer "+encrypt(c.secret))

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
