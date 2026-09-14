package telegram

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestEphemeralSendPayload(t *testing.T) {
	var body map[string]any
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewDecoder(r.Body).Decode(&body)
		_, _ = io.WriteString(w, `{"ok":true,"result":{"message_id":0,"ephemeral_message_id":9,"chat":{"id":-100,"type":"supergroup"}}}`)
	}))
	defer server.Close()

	client, err := NewClient(server.URL, "TOKEN", "")
	if err != nil {
		t.Fatal(err)
	}
	sent, err := client.SendMessage(context.Background(), int64(-100), "hi", &SendOptions{
		Ephemeral:       &EphemeralMessageParameters{ReceiverUserID: 42},
		ReplyParameters: &ReplyParameters{MessageID: 7},
	})
	if err != nil {
		t.Fatal(err)
	}

	params, ok := body["ephemeral_message_parameters"].(map[string]any)
	if !ok || params["receiver_user_id"] != float64(42) {
		t.Fatalf("ephemeral parameters not sent: %v", body)
	}
	reply := body["reply_parameters"].(map[string]any)
	if _, has := reply["ephemeral_message_id"]; has || reply["message_id"] != float64(7) {
		t.Fatalf("reply parameters = %v, want only message_id 7", reply)
	}
	if sent.EphemeralMessageID != 9 {
		t.Fatalf("ephemeral message id = %d, want 9", sent.EphemeralMessageID)
	}
}
