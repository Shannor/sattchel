package tui

import (
	"errors"
	"testing"

	tea "charm.land/bubbletea/v2"
)

func TestListSelectModelResult(t *testing.T) {
	option := ListOption{TitleStr: "Option", ValueStr: "value"}

	t.Run("returns selected option on enter", func(t *testing.T) {
		updated, _ := NewListSelect("title", []ListOption{option}).Update(tea.KeyPressMsg(tea.Key{Code: tea.KeyEnter}))
		model := updated.(ListSelectModel)

		selected, err := model.Result()
		if err != nil {
			t.Fatalf("expected no error, got %v", err)
		}
		if selected == nil || selected.ValueStr != option.ValueStr {
			t.Fatalf("expected selected option %q, got %+v", option.ValueStr, selected)
		}
	})

	t.Run("returns ErrUserAborted on escape", func(t *testing.T) {
		updated, _ := NewListSelect("title", []ListOption{option}).Update(tea.KeyPressMsg(tea.Key{Code: tea.KeyEsc}))
		model := updated.(ListSelectModel)

		selected, err := model.Result()
		if selected != nil {
			t.Fatalf("expected no selection, got %+v", selected)
		}
		if !errors.Is(err, ErrUserAborted) {
			t.Fatalf("expected ErrUserAborted, got %v", err)
		}
	})
}
