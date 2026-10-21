package bot

import (
	"context"

	"github.com/Saveliy12/expenses/internal/model"
)

type TextProcessor interface {
	Process(ctx context.Context, actor model.Actor, text string) (string, error)
}
