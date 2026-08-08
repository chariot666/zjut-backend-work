package controller

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"net/http"
	"os"
	"strings"
	"time"

	openaimodel "github.com/cloudwego/eino-ext/components/model/openai"
	einomodel "github.com/cloudwego/eino/components/model"
	einotool "github.com/cloudwego/eino/components/tool"
	"github.com/cloudwego/eino/components/tool/utils"
	"github.com/cloudwego/eino/compose"
	"github.com/cloudwego/eino/flow/agent/react"
	"github.com/cloudwego/eino/schema"
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"

	"forum-backend/database"
	forumModel "forum-backend/model"
	response "forum-backend/utils"
)

const agentSystemPrompt = `你是校园论坛后端的智能助手。
必须遵守：
1. 用户查询帖子、评论、热榜、详情等只读信息时，必须调用后端提供的 tools，不要自己编造数据。
2. tools 返回的是后端真实查询结果，你要把结果整理成简短中文回答。
3. 用户想发布、创建、写一条帖子时，不要调用任何写入 tool，也不要说已经发布，只能辅助生成草稿。
4. 真正发布帖子必须等待用户下一次请求携带 confirm_draft_id，由服务端执行。`

type agentChatRequest struct {
	SessionID      string `json:"session_id" binding:"required,min=1,max=128"`
	Message        string `json:"message" binding:"required,min=1,max=4000"`
	ConfirmDraftID string `json:"confirm_draft_id"`
	EditDraftID    string `json:"edit_draft_id"`
}

func AgentChat(c *gin.Context) {
	var req agentChatRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, http.StatusBadRequest, "参数错误")
		return
	}
	if req.ConfirmDraftID != "" && req.EditDraftID != "" {
		response.Error(c, http.StatusBadRequest, "confirm_draft_id 和 edit_draft_id 不能同时传")
		return
	}

	userID, exists := c.Get("user_id")
	if !exists {
		response.Error(c, http.StatusUnauthorized, "未认证")
		return
	}
	uid := userID.(uint)

	if err := saveAgentMessage(req.SessionID, uid, "user", req.Message); err != nil {
		response.Error(c, http.StatusInternalServerError, "保存会话失败")
		return
	}

	if req.ConfirmDraftID != "" {
		handleDraftConfirm(c, req.SessionID, uid, req.ConfirmDraftID)
		return
	}
	if req.EditDraftID != "" {
		handleDraftEdit(c, req.SessionID, uid, req.EditDraftID, req.Message)
		return
	}

	reply, pendingAction, err := buildAgentReply(c.Request.Context(), req.SessionID, uid, req.Message)
	if err != nil {
		response.Error(c, http.StatusInternalServerError, "Agent处理失败: "+err.Error())
		return
	}
	if err := saveAgentMessage(req.SessionID, uid, "assistant", reply); err != nil {
		response.Error(c, http.StatusInternalServerError, "保存会话失败")
		return
	}

	response.Success(c, gin.H{
		"session_id":     req.SessionID,
		"reply":          reply,
		"pending_action": pendingAction,
	})
}

func handleDraftConfirm(c *gin.Context, sessionID string, userID uint, draftID string) {
	var result gin.H
	err := database.DB.Transaction(func(tx *gorm.DB) error {
		var draft forumModel.AgentDraft
		if err := findPendingDraft(tx, sessionID, userID, draftID, &draft); err != nil {
			return err
		}
		if draft.Action != "create_post" {
			return errors.New("不支持的草稿操作")
		}

		post := forumModel.Post{
			Content: draft.Content,
			UserID:  userID,
		}
		if err := tx.Create(&post).Error; err != nil {
			return err
		}

		draft.Status = "confirmed"
		if err := tx.Save(&draft).Error; err != nil {
			return err
		}

		reply := fmt.Sprintf("已确认并发布帖子，帖子 ID 是 %d。", post.ID)
		if err := tx.Create(&forumModel.AgentMessage{
			SessionID: sessionID,
			UserID:    userID,
			Role:      "assistant",
			Content:   reply,
		}).Error; err != nil {
			return err
		}

		result = gin.H{
			"session_id":     sessionID,
			"reply":          reply,
			"pending_action": nil,
		}
		return nil
	})
	if err != nil {
		writeDraftError(c, err)
		return
	}

	response.Success(c, result)
}

func handleDraftEdit(c *gin.Context, sessionID string, userID uint, draftID string, instruction string) {
	var result gin.H
	err := database.DB.Transaction(func(tx *gorm.DB) error {
		var draft forumModel.AgentDraft
		if err := findPendingDraft(tx, sessionID, userID, draftID, &draft); err != nil {
			return err
		}
		if draft.Action != "create_post" {
			return errors.New("不支持的草稿操作")
		}

		editedContent, err := reviseDraftContent(c.Request.Context(), sessionID, userID, draft.Content, instruction)
		if err != nil {
			return err
		}
		if editedContent == "" {
			return errors.New("修改后的草稿不能为空")
		}

		draft.Content = editedContent
		draft.ExpiresAt = time.Now().Add(30 * time.Minute)
		if err := tx.Save(&draft).Error; err != nil {
			return err
		}

		pendingAction := &forumModel.PendingActionResponse{
			DraftID:   draft.DraftID,
			Action:    draft.Action,
			Content:   draft.Content,
			ExpiresAt: draft.ExpiresAt,
		}
		reply := fmt.Sprintf("已根据你的要求修改草稿，草稿 ID 仍是 %s。确认无误后再传入 confirm_draft_id 发布。", draft.DraftID)
		if err := tx.Create(&forumModel.AgentMessage{
			SessionID: sessionID,
			UserID:    userID,
			Role:      "assistant",
			Content:   reply,
		}).Error; err != nil {
			return err
		}

		result = gin.H{
			"session_id":     sessionID,
			"reply":          reply,
			"pending_action": pendingAction,
		}
		return nil
	})
	if err != nil {
		writeDraftError(c, err)
		return
	}

	response.Success(c, result)
}

func findPendingDraft(tx *gorm.DB, sessionID string, userID uint, draftID string, draft *forumModel.AgentDraft) error {
	if err := tx.Where(
		"draft_id = ? AND session_id = ? AND user_id = ?",
		draftID,
		sessionID,
		userID,
	).First(draft).Error; err != nil {
		return err
	}
	if draft.Status != "pending" {
		return errors.New("草稿已经处理过")
	}
	if time.Now().After(draft.ExpiresAt) {
		draft.Status = "expired"
		tx.Save(draft)
		return errors.New("草稿已过期")
	}
	return nil
}

func writeDraftError(c *gin.Context, err error) {
	if errors.Is(err, gorm.ErrRecordNotFound) {
		response.Error(c, http.StatusNotFound, "草稿不存在")
		return
	}
	response.Error(c, http.StatusBadRequest, err.Error())
}

func buildAgentReply(ctx context.Context, sessionID string, userID uint, message string) (string, *forumModel.PendingActionResponse, error) {
	if isCreatePostIntent(message) {
		return buildPostDraft(ctx, sessionID, userID, message)
	}

	reply, usedEino, err := runEinoReadAgent(ctx, sessionID, userID)
	if err != nil {
		return "", nil, err
	}
	if usedEino {
		return reply, nil, nil
	}

	reply, err = buildLocalReadReply(sessionID, userID, message)
	return reply, nil, err
}

func buildPostDraft(ctx context.Context, sessionID string, userID uint, message string) (string, *forumModel.PendingActionResponse, error) {
	content := ""
	chatModel, ok, err := newEinoChatModel(ctx)
	if err != nil {
		return "", nil, err
	}
	if ok {
		messages, err := loadSessionMessages(sessionID, userID, 10)
		if err != nil {
			return "", nil, err
		}
		prompt := append([]*schema.Message{
			schema.SystemMessage("你是校园论坛帖子撰写助手。请根据用户需求生成一条可直接发布的帖子正文，只输出正文，不要说已经发布，长度控制在200字以内。"),
		}, messages...)
		modelReply, err := chatModel.Generate(ctx, prompt)
		if err != nil {
			return "", nil, err
		}
		content = strings.TrimSpace(modelReply.Content)
	}
	if content == "" {
		content = draftPostContent(message)
	}

	draftID, err := newDraftID()
	if err != nil {
		return "", nil, err
	}
	expiresAt := time.Now().Add(30 * time.Minute)
	draft := forumModel.AgentDraft{
		DraftID:   draftID,
		SessionID: sessionID,
		UserID:    userID,
		Action:    "create_post",
		Content:   content,
		Status:    "pending",
		ExpiresAt: expiresAt,
	}
	if err := database.DB.Create(&draft).Error; err != nil {
		return "", nil, err
	}

	pendingAction := &forumModel.PendingActionResponse{
		DraftID:   draftID,
		Action:    "create_post",
		Content:   content,
		ExpiresAt: expiresAt,
	}
	reply := fmt.Sprintf("我已生成帖子草稿，草稿 ID 是 %s。你可以传入 edit_draft_id 修改草稿，或传入 confirm_draft_id 确认发布。", draftID)
	return reply, pendingAction, nil
}

func reviseDraftContent(ctx context.Context, sessionID string, userID uint, oldContent string, instruction string) (string, error) {
	if replacement := extractExplicitDraftReplacement(instruction); replacement != "" {
		return replacement, nil
	}

	chatModel, ok, err := newEinoChatModel(ctx)
	if err != nil {
		return "", err
	}
	if ok {
		messages, err := loadSessionMessages(sessionID, userID, 8)
		if err != nil {
			return "", err
		}
		prompt := append([]*schema.Message{
			schema.SystemMessage("你是校园论坛帖子草稿修改助手。根据旧草稿和用户修改要求，输出修改后的帖子正文。只输出正文，不要解释，不要说已经发布，长度控制在200字以内。"),
			schema.UserMessage("旧草稿：\n" + oldContent + "\n\n修改要求：\n" + instruction),
		}, messages...)
		modelReply, err := chatModel.Generate(ctx, prompt)
		if err != nil {
			return "", err
		}
		if content := strings.TrimSpace(modelReply.Content); content != "" {
			return content, nil
		}
	}

	return oldContent + "\n" + strings.TrimSpace(instruction), nil
}

func extractExplicitDraftReplacement(message string) string {
	for _, keyword := range []string{"改成：", "改成:", "修改为：", "修改为:", "换成：", "换成:"} {
		if index := strings.Index(message, keyword); index >= 0 {
			return strings.TrimSpace(message[index+len(keyword):])
		}
	}
	return ""
}

func runEinoReadAgent(ctx context.Context, sessionID string, userID uint) (string, bool, error) {
	chatModel, ok, err := newEinoChatModel(ctx)
	if err != nil {
		return "", false, err
	}
	if !ok {
		return "", false, nil
	}

	tools, err := buildAgentTools()
	if err != nil {
		return "", false, err
	}
	agent, err := react.NewAgent(ctx, &react.AgentConfig{
		ToolCallingModel: chatModel,
		ToolsConfig: compose.ToolsNodeConfig{
			Tools:               tools,
			ExecuteSequentially: true,
		},
		MessageModifier: react.NewPersonaModifier(agentSystemPrompt),
		MaxStep:         8,
	})
	if err != nil {
		return "", true, err
	}

	messages, err := loadSessionMessages(sessionID, userID, 12)
	if err != nil {
		return "", true, err
	}
	out, err := agent.Generate(ctx, messages)
	if err != nil {
		return "", true, err
	}
	reply := strings.TrimSpace(out.Content)
	if reply == "" {
		reply = "我已经通过工具查询了后端数据，但模型没有生成最终文本，请换一种问法再试一次。"
	}
	return reply, true, nil
}

func newEinoChatModel(ctx context.Context) (einomodel.ToolCallingChatModel, bool, error) {
	apiKey := strings.TrimSpace(os.Getenv("OPENAI_API_KEY"))
	modelName := strings.TrimSpace(os.Getenv("OPENAI_MODEL"))
	if apiKey == "" || modelName == "" {
		return nil, false, nil
	}

	maxTokens := 600
	temperature := float32(0.2)
	chatModel, err := openaimodel.NewChatModel(ctx, &openaimodel.ChatModelConfig{
		APIKey:      apiKey,
		Model:       modelName,
		BaseURL:     strings.TrimSpace(os.Getenv("OPENAI_BASE_URL")),
		Timeout:     30 * time.Second,
		MaxTokens:   &maxTokens,
		Temperature: &temperature,
	})
	if err != nil {
		return nil, false, err
	}
	return chatModel, true, nil
}

func loadSessionMessages(sessionID string, userID uint, limit int) ([]*schema.Message, error) {
	var stored []forumModel.AgentMessage
	err := database.DB.
		Where("session_id = ? AND user_id = ?", sessionID, userID).
		Order("created_at desc").
		Limit(limit).
		Find(&stored).Error
	if err != nil {
		return nil, err
	}

	messages := make([]*schema.Message, 0, len(stored))
	for i := len(stored) - 1; i >= 0; i-- {
		msg := stored[i]
		switch msg.Role {
		case "assistant":
			messages = append(messages, schema.AssistantMessage(msg.Content, nil))
		case "system":
			messages = append(messages, schema.SystemMessage(msg.Content))
		default:
			messages = append(messages, schema.UserMessage(msg.Content))
		}
	}
	return messages, nil
}

type listPostsToolInput struct {
	Page     int    `json:"page" jsonschema:"description=页码，从1开始"`
	PageSize int    `json:"page_size" jsonschema:"description=每页数量，最大20"`
	Sort     string `json:"sort" jsonschema:"description=排序方式，new表示最新，hot表示热度"`
}

type listPostsToolOutput struct {
	Items []forumModel.PostResponse `json:"items"`
	Meta  map[string]int64          `json:"meta"`
}

type getPostDetailToolInput struct {
	PostID uint `json:"post_id" jsonschema:"description=帖子ID"`
}

type searchPostsToolInput struct {
	Keyword string `json:"keyword" jsonschema:"description=帖子内容关键词"`
	Limit   int    `json:"limit" jsonschema:"description=最多返回几条，最大10"`
}

func buildAgentTools() ([]einotool.BaseTool, error) {
	listPostsTool, err := utils.InferTool(
		"list_posts",
		"查询论坛帖子列表，可按最新或热度排序。",
		func(ctx context.Context, input listPostsToolInput) (listPostsToolOutput, error) {
			page := input.Page
			if page < 1 {
				page = 1
			}
			pageSize := input.PageSize
			if pageSize < 1 {
				pageSize = 5
			}
			if pageSize > 20 {
				pageSize = 20
			}

			var total int64
			query := database.DB.Model(&forumModel.Post{})
			if err := query.Count(&total).Error; err != nil {
				return listPostsToolOutput{}, err
			}

			var posts []forumModel.Post
			if err := query.
				Preload("User").
				Order(postListOrder(input.Sort)).
				Limit(pageSize).
				Offset((page - 1) * pageSize).
				Find(&posts).Error; err != nil {
				return listPostsToolOutput{}, err
			}

			items := make([]forumModel.PostResponse, 0, len(posts))
			for _, post := range posts {
				items = append(items, buildPostResponse(post, countComments(post.ID)))
			}
			return listPostsToolOutput{
				Items: items,
				Meta: map[string]int64{
					"page":      int64(page),
					"page_size": int64(pageSize),
					"total":     total,
				},
			}, nil
		},
	)
	if err != nil {
		return nil, err
	}

	getPostDetailTool, err := utils.InferTool(
		"get_post_detail",
		"查询单个帖子的详情和评论。",
		func(ctx context.Context, input getPostDetailToolInput) (forumModel.PostDetailResponse, error) {
			var post forumModel.Post
			err := database.DB.
				Preload("User").
				Preload("Comments", func(db *gorm.DB) *gorm.DB {
					return db.Order("created_at asc")
				}).
				Preload("Comments.User").
				First(&post, input.PostID).Error
			if err != nil {
				return forumModel.PostDetailResponse{}, err
			}

			comments := make([]forumModel.CommentResponse, 0, len(post.Comments))
			for _, comment := range post.Comments {
				comments = append(comments, comment.ToResponse())
			}
			return forumModel.PostDetailResponse{
				PostResponse: buildPostResponse(post, int64(len(comments))),
				Comments:     comments,
			}, nil
		},
	)
	if err != nil {
		return nil, err
	}

	searchPostsTool, err := utils.InferTool(
		"search_posts",
		"按关键词搜索论坛帖子。",
		func(ctx context.Context, input searchPostsToolInput) (listPostsToolOutput, error) {
			keyword := strings.TrimSpace(input.Keyword)
			limit := input.Limit
			if limit < 1 {
				limit = 5
			}
			if limit > 10 {
				limit = 10
			}

			var posts []forumModel.Post
			query := database.DB.Model(&forumModel.Post{}).Preload("User")
			if keyword != "" {
				query = query.Where("content LIKE ?", "%"+keyword+"%")
			}
			if err := query.Order("created_at desc").Limit(limit).Find(&posts).Error; err != nil {
				return listPostsToolOutput{}, err
			}

			items := make([]forumModel.PostResponse, 0, len(posts))
			for _, post := range posts {
				items = append(items, buildPostResponse(post, countComments(post.ID)))
			}
			return listPostsToolOutput{
				Items: items,
				Meta: map[string]int64{
					"page":      1,
					"page_size": int64(limit),
					"total":     int64(len(items)),
				},
			}, nil
		},
	)
	if err != nil {
		return nil, err
	}

	return []einotool.BaseTool{listPostsTool, getPostDetailTool, searchPostsTool}, nil
}

func buildLocalReadReply(sessionID string, userID uint, message string) (string, error) {
	normalizedMessage := strings.ToLower(message)
	if strings.Contains(message, "查询") ||
		strings.Contains(message, "列表") ||
		strings.Contains(message, "查看") ||
		strings.Contains(message, "帖子") ||
		strings.Contains(normalizedMessage, "post") {
		return queryRecentPosts()
	}
	if strings.Contains(normalizedMessage, "history") ||
		strings.Contains(message, "历史") ||
		strings.Contains(message, "上下文") {
		return summarizeSession(sessionID, userID)
	}
	return "我可以帮你查询帖子列表、查看帖子详情，或者起草一条帖子。发布类操作会先生成草稿，等你确认后才真正执行。", nil
}

func isCreatePostIntent(message string) bool {
	normalizedMessage := strings.ToLower(message)
	if strings.Contains(normalizedMessage, "draft") ||
		strings.Contains(message, "起草") ||
		strings.Contains(message, "写一条") ||
		strings.Contains(message, "创建帖子") ||
		strings.Contains(message, "发布帖子") ||
		strings.Contains(message, "发帖") {
		return true
	}
	return false
}

func draftPostContent(message string) string {
	if strings.Contains(message, "招新") {
		return "技术部暑期招新开始啦！欢迎对后端、前端、项目开发感兴趣的同学报名，一起学习、实践和成长。"
	}
	if strings.Contains(message, "寻物") || strings.Contains(message, "丢") || strings.Contains(message, "遗失") {
		return "寻物启事：本人有物品遗失，如有同学捡到或看到相关线索，麻烦在评论区联系我，非常感谢！"
	}
	if strings.Contains(message, "失物招领") || strings.Contains(message, "捡到") || strings.Contains(message, "拾到") {
		return "失物招领：本人捡到一件物品，请失主在评论区说明物品特征后联系认领。"
	}
	if strings.Contains(message, "活动") {
		return "新的校园活动即将开始，欢迎大家关注并积极参与！"
	}
	if strings.Contains(message, "二手") || strings.Contains(message, "转让") {
		return "二手转让：有一件闲置物品想转让，感兴趣的同学可以在评论区联系我了解详情。"
	}
	if strings.Contains(message, "求助") || strings.Contains(message, "请教") || strings.Contains(message, "问题") {
		return "求助帖：我遇到了一些问题，想请教一下大家。如果有了解的同学，欢迎在评论区给我建议。"
	}

	topic := extractDraftTopic(message)
	return fmt.Sprintf("关于“%s”的帖子：欢迎大家在评论区交流看法。", topic)
}

func extractDraftTopic(message string) string {
	topic := strings.TrimSpace(message)
	replacer := strings.NewReplacer(
		"帮我", "",
		"请帮我", "",
		"起草", "",
		"写", "",
		"发布", "",
		"一条", "",
		"一个", "",
		"帖子", "",
		"的", "",
	)
	topic = strings.TrimSpace(replacer.Replace(topic))
	if topic == "" {
		return "校园交流"
	}
	return topic
}

func queryRecentPosts() (string, error) {
	var posts []forumModel.Post
	if err := database.DB.Order("created_at desc").Limit(3).Find(&posts).Error; err != nil {
		return "", err
	}
	if len(posts) == 0 {
		return "目前还没有帖子。", nil
	}

	lines := make([]string, 0, len(posts))
	for _, post := range posts {
		lines = append(lines, fmt.Sprintf("ID %d：%s", post.ID, post.Content))
	}
	return "最近的帖子有：" + strings.Join(lines, "；"), nil
}

func summarizeSession(sessionID string, userID uint) (string, error) {
	var count int64
	if err := database.DB.Model(&forumModel.AgentMessage{}).
		Where("session_id = ? AND user_id = ?", sessionID, userID).
		Count(&count).Error; err != nil {
		return "", err
	}
	return fmt.Sprintf("当前会话已经保存了 %d 条消息。", count), nil
}

func saveAgentMessage(sessionID string, userID uint, role string, content string) error {
	return database.DB.Create(&forumModel.AgentMessage{
		SessionID: sessionID,
		UserID:    userID,
		Role:      role,
		Content:   content,
	}).Error
}

func newDraftID() (string, error) {
	bytes := make([]byte, 6)
	if _, err := rand.Read(bytes); err != nil {
		return "", err
	}
	return "draft_" + hex.EncodeToString(bytes), nil
}
