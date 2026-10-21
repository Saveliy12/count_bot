package bot

import (
	"context"

	"github.com/Saveliy12/expenses/internal/model"
)

type StubProcessor struct{}

func (StubProcessor) Process(context.Context, model.Actor, string) (string, error) {
	return LLMNotReady, nil
}
