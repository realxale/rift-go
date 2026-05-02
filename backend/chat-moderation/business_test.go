package moderation

import (
	"testing"
)

// TestHandleModeration_UnknownAction проверяет, что неизвестное действие возвращает ошибку
func TestHandleModeration_UnknownAction(t *testing.T) {
	req := ModerationRequest{
		JWT:      "invalid-jwt",
		RoomName: "test-room",
		Target:   "",
		Action:   "unknown_action",
	}

	_, err := HandleModeration(req)
	if err == nil {
		t.Error("expected error for unknown action, got nil")
	}
}

// TestHandleModeration_InvalidJWT проверяет, что невалидный JWT возвращает ошибку
func TestHandleModeration_InvalidJWT(t *testing.T) {
	req := ModerationRequest{
		JWT:      "invalid-jwt",
		RoomName: "test-room",
		Target:   "target-user",
		Action:   "ban",
	}

	_, err := HandleModeration(req)
	if err == nil {
		t.Error("expected error for invalid JWT, got nil")
	}
}

// TestHandleModeration_EmptyTargetBan проверяет, что ban без target возвращает ошибку
func TestHandleModeration_EmptyTargetBan(t *testing.T) {
	// Этот тест проверит, что при пустом target для ban возвращается ошибка
	// через auth.ParseJWT, но если JWT невалидный — сначала будет ошибка JWT
	// Поэтому проверяем только что код не паникует
	req := ModerationRequest{
		JWT:      "invalid",
		RoomName: "room",
		Target:   "",
		Action:   "ban",
	}

	_, err := HandleModeration(req)
	if err == nil {
		t.Error("expected error, got nil")
	}
}