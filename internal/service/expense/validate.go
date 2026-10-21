package expense

import "github.com/Saveliy12/expenses/internal/model"

func validateActor(actor model.Actor) error {
	if actor.TelegramID <= 0 {
		return model.NewValidationError("telegram_id", "must be positive")
	}
	return nil
}

func isEmptyPatch(patch model.UpdateExpenseInput) bool {
	return patch.Amount == nil && patch.Category == nil && patch.SpentAt == nil && patch.Comment == nil
}
