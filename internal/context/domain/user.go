package domain

import "strings"

type User struct {
	ID             string
	TelegramUserID int64
	Username       string
	DisplayName    string
}

func DisplayNameFor(firstName, lastName, username string) string {
	if name := strings.TrimSpace(firstName + " " + lastName); name != "" {
		return name
	}
	if username != "" {
		return username
	}
	return "Пользователь"
}
