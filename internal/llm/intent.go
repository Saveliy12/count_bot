package llm

import (
	"context"
	"errors"
	"time"

	"github.com/shopspring/decimal"

	"github.com/Saveliy12/expenses/internal/model"
)

var (
	ErrInvalidResponse = errors.New("invalid llm response")
	ErrRateLimited     = errors.New("llm rate limited")
	ErrUnavailable     = errors.New("llm unavailable")
)

type Action string

const (
	ActionCreate  Action = "create"
	ActionList    Action = "list"
	ActionUpdate  Action = "update"
	ActionDelete  Action = "delete"
	ActionClarify Action = "clarify"
)

type TargetMode string

const (
	TargetLatest TargetMode = "latest"
	TargetID     TargetMode = "id"
	TargetQuery  TargetMode = "query"
)

type PeriodPreset string

const (
	PeriodToday     PeriodPreset = "today"
	PeriodYesterday PeriodPreset = "yesterday"
	PeriodWeek      PeriodPreset = "week"
	PeriodMonth     PeriodPreset = "month"
	PeriodYear      PeriodPreset = "year"
	PeriodAll       PeriodPreset = "all"
)

type Request struct {
	Text  string
	Today time.Time
}

type Intent struct {
	Action         Action
	Amount         *decimal.Decimal
	OldAmount      *decimal.Decimal
	Category       *model.Category
	DateHint       *string
	Comment        *string
	ClearComment   bool
	TargetMode     TargetMode
	TargetID       *int64
	TargetQuery    *string
	TargetDateHint *string
	TargetCategory *model.Category
	PeriodPreset   PeriodPreset
	PeriodFrom     *time.Time
	PeriodTo       *time.Time
	Clarification  string
}

type Provider interface {
	Parse(ctx context.Context, req Request) (Intent, error)
}
