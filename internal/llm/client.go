package llm

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/shopspring/decimal"

	"github.com/Saveliy12/expenses/internal/model"
)

const (
	defaultTimeout     = 20 * time.Second
	maxTokens          = 600
	maxResponseBodyLen = 1 << 20
	maxAttempts        = 3
	maxRetryDelay      = 4 * time.Second
)

type Config struct {
	BaseURL string
	APIKey  string
	Model   string
	Timeout time.Duration
}

type Client struct {
	baseURL    string
	apiKey     string
	model      string
	timeout    time.Duration
	httpClient *http.Client
	logger     *slog.Logger
}

func NewClient(cfg Config, logger *slog.Logger) *Client {
	if logger == nil {
		logger = slog.Default()
	}
	timeout := cfg.Timeout
	if timeout <= 0 {
		timeout = defaultTimeout
	}

	return &Client{
		baseURL: strings.TrimRight(cfg.BaseURL, "/"),
		apiKey:  cfg.APIKey,
		model:   cfg.Model,
		timeout: timeout,
		httpClient: &http.Client{
			Timeout: timeout,
		},
		logger: logger,
	}
}

func (c *Client) Parse(ctx context.Context, req Request) (Intent, error) {
	text := strings.TrimSpace(req.Text)
	if text == "" {
		return Intent{}, fmt.Errorf("%w: empty user text", ErrInvalidResponse)
	}

	ctx, cancel := context.WithTimeout(ctx, c.timeout)
	defer cancel()

	payload := chatCompletionRequest{
		Model: c.model,
		Messages: []chatMessage{
			{Role: "system", Content: systemPrompt(req.Today)},
			{Role: "user", Content: text},
		},
		Temperature:     0,
		MaxTokens:       maxTokens,
		ReasoningEffort: reasoningEffort(c.model),
		ResponseFormat: responseFormat{
			Type: "json_schema",
			JSONSchema: jsonSchema{
				Name:   "expense_intent",
				Strict: true,
				Schema: intentSchema(),
			},
		},
	}

	body, err := json.Marshal(payload)
	if err != nil {
		return Intent{}, fmt.Errorf("encode llm request: %w", err)
	}

	var (
		response     *http.Response
		responseBody []byte
	)
	for attempt := 1; attempt <= maxAttempts; attempt++ {
		request, err := http.NewRequestWithContext(
			ctx,
			http.MethodPost,
			c.baseURL+"/chat/completions",
			bytes.NewReader(body),
		)
		if err != nil {
			return Intent{}, fmt.Errorf("create llm request: %w", err)
		}
		request.Header.Set("Authorization", "Bearer "+c.apiKey)
		request.Header.Set("Content-Type", "application/json")
		request.Header.Set("Accept", "application/json")
		request.Header.Set("User-Agent", "FamilyExpensesBot/1.0")

		startedAt := time.Now()
		response, err = c.httpClient.Do(request)
		if err != nil {
			if attempt < maxAttempts && sleepContext(ctx, retryBackoff(attempt)) {
				c.logger.Warn("llm request failed, retrying",
					"attempt", attempt,
					"error", err,
				)
				continue
			}
			return Intent{}, fmt.Errorf("%w: %v", ErrUnavailable, err)
		}

		responseBody, err = io.ReadAll(io.LimitReader(response.Body, maxResponseBodyLen))
		_ = response.Body.Close()
		if err != nil {
			return Intent{}, fmt.Errorf("%w: read response: %v", ErrUnavailable, err)
		}

		c.logger.Debug("llm request",
			"model", c.model,
			"status", response.StatusCode,
			"duration", time.Since(startedAt),
		)

		if response.StatusCode != http.StatusTooManyRequests &&
			response.StatusCode < 500 &&
			!isRetryableGenerationError(response.StatusCode, responseBody) {
			break
		}
		if attempt == maxAttempts {
			break
		}

		delay := retryDelay(response, attempt)
		if delay > maxRetryDelay {
			if response.StatusCode == http.StatusTooManyRequests {
				break
			}
			delay = maxRetryDelay
		}

		c.logger.Warn("llm request retrying",
			"status", response.StatusCode,
			"attempt", attempt,
			"delay", delay,
		)
		if !sleepContext(ctx, delay) {
			return Intent{}, fmt.Errorf("%w: %v", ErrUnavailable, ctx.Err())
		}
	}

	requestID := response.Header.Get("X-Request-ID")

	switch {
	case response.StatusCode == http.StatusTooManyRequests:
		return Intent{}, fmt.Errorf(
			"%w: status %d, request id %q",
			ErrRateLimited,
			response.StatusCode,
			requestID,
		)
	case response.StatusCode < 200 || response.StatusCode >= 300:
		return Intent{}, fmt.Errorf(
			"%w: status %d, request id %q",
			ErrUnavailable,
			response.StatusCode,
			requestID,
		)
	}

	var completion chatCompletionResponse
	if err := json.Unmarshal(responseBody, &completion); err != nil {
		return Intent{}, fmt.Errorf("%w: decode completion: %v", ErrInvalidResponse, err)
	}
	if len(completion.Choices) == 0 {
		return Intent{}, fmt.Errorf("%w: empty choices", ErrInvalidResponse)
	}

	content := strings.TrimSpace(completion.Choices[0].Message.Content)
	if content == "" {
		return Intent{}, fmt.Errorf("%w: empty content", ErrInvalidResponse)
	}

	var raw rawIntent
	if err := json.Unmarshal([]byte(content), &raw); err != nil {
		return Intent{}, fmt.Errorf("%w: decode intent: %v", ErrInvalidResponse, err)
	}

	return convert(raw)
}

type chatCompletionRequest struct {
	Model           string         `json:"model"`
	Messages        []chatMessage  `json:"messages"`
	Temperature     float64        `json:"temperature"`
	MaxTokens       int            `json:"max_tokens"`
	ReasoningEffort string         `json:"reasoning_effort,omitempty"`
	ResponseFormat  responseFormat `json:"response_format"`
}

func reasoningEffort(model string) string {
	if strings.HasPrefix(model, "openai/gpt-oss") {
		return "low"
	}

	return ""
}

type chatMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type responseFormat struct {
	Type       string     `json:"type"`
	JSONSchema jsonSchema `json:"json_schema"`
}

type jsonSchema struct {
	Name   string         `json:"name"`
	Strict bool           `json:"strict"`
	Schema map[string]any `json:"schema"`
}

type chatCompletionResponse struct {
	Choices []struct {
		Message struct {
			Content string `json:"content"`
		} `json:"message"`
	} `json:"choices"`
}

type rawIntent struct {
	Action         string  `json:"action"`
	Amount         *string `json:"amount"`
	OldAmount      *string `json:"old_amount"`
	Category       *string `json:"category"`
	DateHint       *string `json:"date_hint"`
	Comment        *string `json:"comment"`
	ClearComment   bool    `json:"clear_comment"`
	TargetMode     *string `json:"target_mode"`
	TargetID       *int64  `json:"target_id"`
	TargetQuery    *string `json:"target_query"`
	TargetDateHint *string `json:"target_date_hint"`
	TargetCategory *string `json:"target_category"`
	PeriodPreset   *string `json:"period_preset"`
	PeriodFrom     *string `json:"period_from"`
	PeriodTo       *string `json:"period_to"`
	Clarification  *string `json:"clarification"`
}

func convert(raw rawIntent) (Intent, error) {
	action := Action(strings.ToLower(strings.TrimSpace(raw.Action)))
	switch action {
	case ActionCreate, ActionList, ActionUpdate, ActionDelete, ActionClarify:
	default:
		return Intent{}, fmt.Errorf("%w: unsupported action %q", ErrInvalidResponse, raw.Action)
	}

	intent := Intent{Action: action}

	amount, err := parseAmountPointer(raw.Amount)
	if err != nil {
		return Intent{}, fmt.Errorf("%w: amount: %v", ErrInvalidResponse, err)
	}
	intent.Amount = amount

	oldAmount, err := parseAmountPointer(raw.OldAmount)
	if err != nil {
		return Intent{}, fmt.Errorf("%w: old_amount: %v", ErrInvalidResponse, err)
	}
	intent.OldAmount = oldAmount

	if categoryValue := cleanStringPointer(raw.Category); categoryValue != nil {
		category, err := model.ParseCategory(*categoryValue)
		if err != nil {
			return Intent{}, fmt.Errorf("%w: category: %v", ErrInvalidResponse, err)
		}
		intent.Category = &category
	}

	intent.DateHint, err = parseDateHintPointer(raw.DateHint, "date_hint")
	if err != nil {
		return Intent{}, err
	}

	intent.Comment = cleanStringPointer(raw.Comment)
	intent.ClearComment = raw.ClearComment

	if raw.TargetID != nil {
		if *raw.TargetID <= 0 {
			return Intent{}, fmt.Errorf("%w: target_id must be positive", ErrInvalidResponse)
		}
		targetID := *raw.TargetID
		intent.TargetID = &targetID
	}
	intent.TargetQuery = cleanStringPointer(raw.TargetQuery)

	intent.TargetDateHint, err = parseDateHintPointer(raw.TargetDateHint, "target_date_hint")
	if err != nil {
		return Intent{}, err
	}

	if categoryValue := cleanStringPointer(raw.TargetCategory); categoryValue != nil {
		category, err := model.ParseCategory(*categoryValue)
		if err != nil {
			return Intent{}, fmt.Errorf("%w: target_category: %v", ErrInvalidResponse, err)
		}
		intent.TargetCategory = &category
	}

	targetMode := TargetMode(strings.ToLower(strings.TrimSpace(deref(raw.TargetMode))))
	switch targetMode {
	case "", TargetLatest, TargetID, TargetQuery:
	default:
		return Intent{}, fmt.Errorf("%w: unsupported target_mode %q", ErrInvalidResponse, deref(raw.TargetMode))
	}
	if targetMode == "" {
		switch {
		case intent.TargetID != nil:
			targetMode = TargetID
		case intent.TargetQuery != nil || intent.TargetCategory != nil || intent.TargetDateHint != nil:
			targetMode = TargetQuery
		}
	}
	if targetMode == TargetID && intent.TargetID == nil {
		return Intent{}, fmt.Errorf("%w: target_id is required for target_mode=id", ErrInvalidResponse)
	}
	if targetMode == TargetQuery &&
		intent.TargetQuery == nil &&
		intent.TargetCategory == nil &&
		intent.TargetDateHint == nil {
		return Intent{}, fmt.Errorf(
			"%w: target_query, target_category or target_date_hint is required for target_mode=query",
			ErrInvalidResponse,
		)
	}
	intent.TargetMode = targetMode

	periodPreset := PeriodPreset(strings.ToLower(strings.TrimSpace(deref(raw.PeriodPreset))))
	switch periodPreset {
	case "", PeriodToday, PeriodYesterday, PeriodWeek, PeriodMonth, PeriodYear, PeriodAll:
	default:
		return Intent{}, fmt.Errorf(
			"%w: unsupported period_preset %q",
			ErrInvalidResponse,
			deref(raw.PeriodPreset),
		)
	}
	intent.PeriodPreset = periodPreset

	intent.PeriodFrom, err = parseDatePointer(raw.PeriodFrom, "period_from")
	if err != nil {
		return Intent{}, err
	}
	intent.PeriodTo, err = parseDatePointer(raw.PeriodTo, "period_to")
	if err != nil {
		return Intent{}, err
	}
	if intent.PeriodFrom != nil && intent.PeriodTo != nil && intent.PeriodFrom.After(*intent.PeriodTo) {
		return Intent{}, fmt.Errorf("%w: period_from must not be after period_to", ErrInvalidResponse)
	}

	if clarification := cleanStringPointer(raw.Clarification); clarification != nil {
		intent.Clarification = *clarification
	}

	return intent, nil
}

func parseAmountPointer(raw *string) (*decimal.Decimal, error) {
	if raw == nil {
		return nil, nil
	}
	value := strings.TrimSpace(*raw)
	if value == "" {
		return nil, nil
	}

	value = strings.ReplaceAll(value, " ", "")
	value = strings.ReplaceAll(value, "\u00a0", "")
	value = strings.TrimSuffix(value, "₽")
	if strings.Count(value, ",") == 1 && !strings.Contains(value, ".") {
		value = strings.Replace(value, ",", ".", 1)
	}

	amount, err := decimal.NewFromString(value)
	if err != nil {
		return nil, err
	}

	return &amount, nil
}

func parseDateHintPointer(raw *string, field string) (*string, error) {
	value := cleanStringPointer(raw)
	if value == nil {
		return nil, nil
	}

	normalized := strings.ToLower(*value)
	switch normalized {
	case "today", "yesterday":
		return &normalized, nil
	default:
		if _, err := time.Parse(dateLayout, *value); err != nil {
			return nil, fmt.Errorf("%w: invalid %s %q", ErrInvalidResponse, field, *value)
		}
		return value, nil
	}
}

func parseDatePointer(raw *string, field string) (*time.Time, error) {
	value := cleanStringPointer(raw)
	if value == nil {
		return nil, nil
	}

	date, err := time.Parse(dateLayout, *value)
	if err != nil {
		return nil, fmt.Errorf("%w: invalid %s %q", ErrInvalidResponse, field, *value)
	}

	return &date, nil
}

func cleanStringPointer(raw *string) *string {
	if raw == nil {
		return nil
	}
	value := strings.TrimSpace(*raw)
	if value == "" {
		return nil
	}

	return &value
}

func deref(value *string) string {
	if value == nil {
		return ""
	}

	return *value
}

func isRetryableGenerationError(status int, body []byte) bool {
	if status != http.StatusBadRequest {
		return false
	}

	return bytes.Contains(body, []byte("json_validate_failed")) ||
		bytes.Contains(body, []byte("failed_generation"))
}

func retryDelay(response *http.Response, attempt int) time.Duration {
	if raw := strings.TrimSpace(response.Header.Get("Retry-After")); raw != "" {
		seconds, err := strconv.Atoi(raw)
		if err == nil && seconds >= 0 {
			return time.Duration(seconds) * time.Second
		}
	}

	return retryBackoff(attempt)
}

func retryBackoff(attempt int) time.Duration {
	return time.Duration(attempt) * time.Second
}

func sleepContext(ctx context.Context, delay time.Duration) bool {
	timer := time.NewTimer(delay)
	defer timer.Stop()

	select {
	case <-ctx.Done():
		return false
	case <-timer.C:
		return true
	}
}
