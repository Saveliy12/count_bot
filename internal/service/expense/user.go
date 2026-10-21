package expense

import (
	"context"
	"strings"

	"github.com/Saveliy12/expenses/internal/model"
)

func (s *Service) EnsureUser(ctx context.Context, profile model.UserProfile) (model.User, error) {
	if profile.TelegramID <= 0 {
		return model.User{}, model.NewValidationError("telegram_id", "must be positive")
	}

	displayName := strings.TrimSpace(profile.DisplayName)
	if displayName == "" {
		return model.User{}, model.NewValidationError("display_name", "must not be empty")
	}
	username := strings.TrimPrefix(strings.TrimSpace(profile.Username), "@")

	family, err := s.families.GetDefault(ctx)
	if err != nil {
		return model.User{}, err
	}

	now := s.now().UTC()
	user := model.User{
		TelegramID:  profile.TelegramID,
		FamilyID:    family.ID,
		DisplayName: displayName,
		Username:    username,
		CreatedAt:   now,
		UpdatedAt:   now,
	}

	if err := model.ValidateUser(user); err != nil {
		return model.User{}, err
	}

	return s.users.Upsert(ctx, user)
}

func (s *Service) ListFamilyUsers(ctx context.Context) ([]model.User, error) {
	family, err := s.families.GetDefault(ctx)
	if err != nil {
		return nil, err
	}

	return s.users.ListFamilyMembers(ctx, family.ID)
}

func (s *Service) familyUser(ctx context.Context, telegramID int64) (model.User, error) {
	family, err := s.families.GetDefault(ctx)
	if err != nil {
		return model.User{}, err
	}

	user, err := s.users.GetByTelegramID(ctx, telegramID)
	if err != nil {
		return model.User{}, err
	}
	if user.FamilyID != family.ID {
		return model.User{}, model.ErrNotFound
	}

	return user, nil
}
