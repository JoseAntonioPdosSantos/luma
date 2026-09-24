package flashcard

import (
	"strings"
	"testing"
	"time"
)

var fixedNow = time.Date(2026, 1, 1, 12, 0, 0, 0, time.UTC)

func TestValidateQuestion(t *testing.T) {
	got, err := ValidateQuestion("  How do you say 'rescue'?  ")
	if err != nil {
		t.Fatalf("ValidateQuestion() returned unexpected error: %v", err)
	}
	if got != "How do you say 'rescue'?" {
		t.Errorf("ValidateQuestion() = %q", got)
	}
}

func TestValidateQuestion_Empty(t *testing.T) {
	if _, err := ValidateQuestion("   "); err == nil {
		t.Error("ValidateQuestion(\"   \") should return an error")
	}
}

func TestValidateQuestion_TooLong(t *testing.T) {
	if _, err := ValidateQuestion(strings.Repeat("a", MaxQuestionLength+1)); err == nil {
		t.Error("ValidateQuestion() with an over-long question should return an error")
	}
}

func TestValidateAnswer_Empty(t *testing.T) {
	if _, err := ValidateAnswer(""); err == nil {
		t.Error("ValidateAnswer(\"\") should return an error")
	}
}

func TestValidateAnswer_TooLong(t *testing.T) {
	if _, err := ValidateAnswer(strings.Repeat("a", MaxAnswerLength+1)); err == nil {
		t.Error("ValidateAnswer() with an over-long answer should return an error")
	}
}

func TestValidateExtendedExampleText_TooLong(t *testing.T) {
	if _, err := ValidateExtendedExampleText(strings.Repeat("a", MaxExampleLength+1)); err == nil {
		t.Error("ValidateExtendedExampleText() with over-long text should return an error")
	}
}

func TestValidateExtendedExampleText_EmptyIsAllowed(t *testing.T) {
	if _, err := ValidateExtendedExampleText(""); err != nil {
		t.Errorf("ValidateExtendedExampleText(\"\") returned unexpected error: %v", err)
	}
}

func TestValidateAudioContentType_Allowed(t *testing.T) {
	for _, ct := range []string{"audio/mpeg", "audio/wav", "audio/ogg"} {
		if err := ValidateAudioContentType(ct); err != nil {
			t.Errorf("ValidateAudioContentType(%q) returned unexpected error: %v", ct, err)
		}
	}
}

func TestValidateAudioContentType_Rejected(t *testing.T) {
	for _, ct := range []string{"video/mp4", "text/plain", "", "audio/mp3"} {
		if err := ValidateAudioContentType(ct); err == nil {
			t.Errorf("ValidateAudioContentType(%q) should return an error", ct)
		}
	}
}

func TestNew_StartsInNewState(t *testing.T) {
	card := New("user-1", "deck-1", "Question?", "Answer", nil, fixedNow)

	if card.Scheduling.State != "new" {
		t.Errorf("Scheduling.State = %q, want %q", card.Scheduling.State, "new")
	}
	if card.IsArchived() {
		t.Error("a freshly created flashcard should not be archived")
	}
}

func TestValidateHint(t *testing.T) {
	got, err := ValidateHint("  Starts with R  ")
	if err != nil || got != "Starts with R" {
		t.Errorf("ValidateHint() = %q, %v; want the trimmed hint", got, err)
	}
	if got, err := ValidateHint(""); err != nil || got != "" {
		t.Errorf("ValidateHint(\"\") = %q, %v; an empty hint is allowed", got, err)
	}
	if _, err := ValidateHint(strings.Repeat("a", MaxHintLength+1)); err == nil {
		t.Error("ValidateHint() with over-long text should return an error")
	}
}
