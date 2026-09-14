package usecase

import (
	"context"

	"ojirun/internal/context/domain"
)

type GetProfile struct {
	users    userReader
	profiles profileReader
}

func NewGetProfile(users userReader, profiles profileReader) *GetProfile {
	return &GetProfile{users: users, profiles: profiles}
}

func (uc *GetProfile) Execute(ctx context.Context, telegramUserID int64) (domain.Profile, error) {
	user, err := uc.users.GetByTelegramID(ctx, telegramUserID)
	if err != nil {
		return domain.Profile{}, err
	}
	return uc.profiles.GetByUserID(ctx, user.ID)
}
