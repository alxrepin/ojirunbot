package telegram

type Update struct {
	UpdateID      int64          `json:"update_id"`
	Message       *Message       `json:"message,omitempty"`
	CallbackQuery *CallbackQuery `json:"callback_query,omitempty"`
}

type Message struct {
	MessageID          int64       `json:"message_id"`
	From               *User       `json:"from,omitempty"`
	Chat               Chat        `json:"chat"`
	Date               int64       `json:"date"`
	Text               string      `json:"text,omitempty"`
	Caption            string      `json:"caption,omitempty"`
	Photo              []PhotoSize `json:"photo,omitempty"`
	ReplyToMessage     *Message    `json:"reply_to_message,omitempty"`
	EphemeralMessageID int64       `json:"ephemeral_message_id,omitempty"`
	ReceiverUser       *User       `json:"receiver_user,omitempty"`
}

type User struct {
	ID        int64  `json:"id"`
	IsBot     bool   `json:"is_bot"`
	FirstName string `json:"first_name"`
	LastName  string `json:"last_name,omitempty"`
	Username  string `json:"username,omitempty"`
}

type Chat struct {
	ID       int64  `json:"id"`
	Type     string `json:"type"`
	Title    string `json:"title,omitempty"`
	Username string `json:"username,omitempty"`
}

type PhotoSize struct {
	FileID       string `json:"file_id"`
	FileUniqueID string `json:"file_unique_id"`
	Width        int    `json:"width"`
	Height       int    `json:"height"`
	FileSize     int64  `json:"file_size,omitempty"`
}

type CallbackQuery struct {
	ID      string   `json:"id"`
	From    User     `json:"from"`
	Message *Message `json:"message,omitempty"`
	Data    string   `json:"data,omitempty"`
}

type File struct {
	FileID       string `json:"file_id"`
	FileUniqueID string `json:"file_unique_id"`
	FileSize     int64  `json:"file_size,omitempty"`
	FilePath     string `json:"file_path,omitempty"`
}

type ChatMember struct {
	Status   string `json:"status"`
	User     User   `json:"user"`
	IsMember bool   `json:"is_member,omitempty"`
}

func (m Message) CommandText() string {
	if m.Text != "" {
		return m.Text
	}
	return m.Caption
}

const maxPhotoSide = 1280

func (m Message) BestPhoto() (PhotoSize, bool) {
	if len(m.Photo) == 0 {
		return PhotoSize{}, false
	}
	best := m.Photo[0]
	for _, photo := range m.Photo[1:] {
		if photoPreferred(photo, best) {
			best = photo
		}
	}
	return best, true
}

func photoPreferred(candidate, best PhotoSize) bool {
	candidateSide := max(candidate.Width, candidate.Height)
	bestSide := max(best.Width, best.Height)
	if (candidateSide <= maxPhotoSide) != (bestSide <= maxPhotoSide) {
		return candidateSide <= maxPhotoSide
	}
	if candidateSide <= maxPhotoSide {
		return candidateSide > bestSide
	}
	return candidateSide < bestSide
}
