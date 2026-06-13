package handlers

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"strings"
	"time"

	"github.com/cloudwego/hertz/pkg/app"
	"github.com/cloudwego/hertz/pkg/protocol/consts"
)

type modelListRequest struct {
	BaseURL string `json:"base_url"`
	APIKey  string `json:"api_key"`
}

type modelEntry struct {
	ID       string `json:"id"`
	Object   string `json:"object"`
	OwnedBy  string `json:"owned_by"`
}

type modelListResponse struct {
	Object string       `json:"object"`
	Data   []modelEntry `json:"data"`
}

// HandleModelList POST /api/models/list — 代理拉取 OpenAI 兼容接口的模型列表。
func (h *Handlers) HandleModelList(ctx context.Context, c *app.RequestContext) {
	var body modelListRequest
	if err := c.BindJSON(&body); err != nil {
		writeErrorKey(c, consts.StatusBadRequest, "api.common.invalidRequestWithDetail", "detail", err.Error())
		return
	}
	baseURL := strings.TrimSpace(body.BaseURL)
	apiKey := strings.TrimSpace(body.APIKey)
	if baseURL == "" {
		writeError(c, consts.StatusBadRequest, "base_url is required")
		return
	}
	if apiKey == "" {
		writeError(c, consts.StatusBadRequest, "api_key is required")
		return
	}

	baseURL = strings.TrimRight(baseURL, "/")
	modelsURL := baseURL + "/v1/models"

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, modelsURL, nil)
	if err != nil {
		log.Printf("[api] HandleModelList 构建请求失败 url=%s err=%v", modelsURL, err)
		writeError(c, consts.StatusInternalServerError, fmt.Sprintf("failed to build request: %v", err))
		return
	}
	req.Header.Set("Authorization", "Bearer "+apiKey)

	httpClient := &http.Client{Timeout: 15 * time.Second}
	resp, err := httpClient.Do(req)
	if err != nil {
		log.Printf("[api] HandleModelList 请求失败 url=%s err=%v", modelsURL, err)
		writeError(c, consts.StatusBadGateway, fmt.Sprintf("failed to fetch models: %v", err))
		return
	}
	defer resp.Body.Close()

	bodyBytes, err := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if err != nil {
		log.Printf("[api] HandleModelList 读取响应失败 url=%s err=%v", modelsURL, err)
		writeError(c, consts.StatusInternalServerError, fmt.Sprintf("failed to read response: %v", err))
		return
	}

	if resp.StatusCode != http.StatusOK {
		log.Printf("[api] HandleModelList 上游返回非200 url=%s status=%d body=%s", modelsURL, resp.StatusCode, string(bodyBytes))
		writeError(c, consts.StatusBadGateway, fmt.Sprintf("provider returned HTTP %d: %s", resp.StatusCode, truncate(string(bodyBytes), 200)))
		return
	}

	var models modelListResponse
	if err := json.Unmarshal(bodyBytes, &models); err != nil {
		log.Printf("[api] HandleModelList 解析失败 url=%s body=%s err=%v", modelsURL, truncate(string(bodyBytes), 200), err)
		writeError(c, consts.StatusBadGateway, fmt.Sprintf("invalid response from provider: %v", err))
		return
	}

	log.Printf("[api] HandleModelList 成功 url=%s count=%d", modelsURL, len(models.Data))
	writeJSON(c, consts.StatusOK, map[string]interface{}{
		"models": models.Data,
	})
}

func truncate(s string, max int) string {
	if len(s) <= max {
		return s
	}
	return s[:max] + "..."
}
