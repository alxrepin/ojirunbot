package user

import "ojirun/internal/context/domain"

type user struct {
	ID             string
	TelegramUserID int64
	Username       string
	DisplayName    string
}

func (u user) toDomain() domain.User {
	return domain.User{
		ID:             u.ID,
		TelegramUserID: u.TelegramUserID,
		Username:       u.Username,
		DisplayName:    u.DisplayName,
	}
}
