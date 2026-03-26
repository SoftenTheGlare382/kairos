package moderation

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"time"

	ark "github.com/sashabaranov/go-openai"
)

type Decision string

const (
	DecisionAllow   Decision = "allow"
	DecisionBlock   Decision = "block"
	DecisionSuspect Decision = "suspect"
)

type Result struct {
	Decision   Decision `json:"decision"`
	Categories []string `json:"categories"`
	Reason     string   `json:"reason"`
}

type ArkClient struct {
	client *ark.Client
	model  string
}

type ArkConfig struct {
	APIKey  string
	BaseURL string
	Model   string
}

func NewArkClient(cfg ArkConfig) (*ArkClient, error) {
	if strings.TrimSpace(cfg.APIKey) == "" {
		return nil, errors.New("ARK_API_KEY is required")
	}
	c := ark.DefaultConfig(cfg.APIKey)
	if cfg.BaseURL != "" {
		c.BaseURL = cfg.BaseURL
	}
	cl := ark.NewClientWithConfig(c)
	model := cfg.Model
	if model == "" {
		model = "doubao-seed-1-6-lite-251015"
	}
	return &ArkClient{client: cl, model: model}, nil
}

func (c *ArkClient) AuditComment(ctx context.Context, content string) (Result, error) {
	// 强制 JSON 输出，便于解析；不使用 stream
	system := `你是内容安全审核助手。请对用户评论进行合规审核，只输出严格 JSON，不要输出其它文本。
要求：
1) decision 只能是 allow / block / suspect
2) 若 decision="allow"，则 categories 必须是空数组 []（表示安全通过），reason 给出简短理由
3) 若 decision!="allow"，则 categories 必须从以下集合中选择（可多选，至少 1 个）：["涉政","色情","辱骂","广告","引流","赌博","其他"]
4) 输出必须是严格 JSON，且只输出一行
JSON schema:
{"decision":"allow|block|suspect","categories":["涉政|色情|辱骂|广告|引流|赌博|其他"],"reason":"简短原因"}`
	user := "评论内容：" + content

	ctx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()

	resp, err := c.client.CreateChatCompletion(ctx, ark.ChatCompletionRequest{
		Model: c.model,
		Messages: []ark.ChatCompletionMessage{
			{Role: ark.ChatMessageRoleSystem, Content: system},
			{Role: ark.ChatMessageRoleUser, Content: user},
		},
	})
	if err != nil {
		return Result{}, err
	}
	out := strings.TrimSpace(resp.Choices[0].Message.Content)
	var r Result
	if err := json.Unmarshal([]byte(out), &r); err != nil {
		return Result{}, err
	}
	switch r.Decision {
	case DecisionAllow, DecisionBlock, DecisionSuspect:
	default:
		r.Decision = DecisionSuspect
	}

	// 纠错：allow 时不应带违规分类；block/suspect 时必须至少有一个分类
	if r.Decision == DecisionAllow {
		r.Categories = nil
	} else if len(r.Categories) == 0 {
		r.Categories = []string{"其他"}
	}
	return r, nil
}

