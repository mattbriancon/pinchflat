package core_test

import (
	"testing"

	"github.com/mattbriancon/pinchflat/internal/core"
)

func TestOutputPathBuilder_Build(t *testing.T) {
	t.Run("it expands 'standard' curly brace variables in the template", func(t *testing.T) {
		result, err := core.OutputPathBuilderBuild("/videos/{{ title }}.{{ ext }}", map[string]string{})
		if err != nil {
			t.Errorf("got error: %v", err)
		}
		expected := "/videos/%(title)S.%(ext)S"
		if result != expected {
			t.Errorf("expected '%s', got '%s'", expected, result)
		}
	})

	t.Run("it expands 'custom' curly brace variables in the template", func(t *testing.T) {
		result, err := core.OutputPathBuilderBuild("/videos/{{ upload_year }}.{{ ext }}", map[string]string{})
		if err != nil {
			t.Errorf("got error: %v", err)
		}
		expected := "/videos/%(upload_date>%Y)S.%(ext)S"
		if result != expected {
			t.Errorf("expected '%s', got '%s'", expected, result)
		}
	})

	t.Run("it respects additional options", func(t *testing.T) {
		result, err := core.OutputPathBuilderBuild("/videos/{{ custom }}.{{ ext }}", map[string]string{"custom": "test"})
		if err != nil {
			t.Errorf("got error: %v", err)
		}
		expected := "/videos/test.%(ext)S"
		if result != expected {
			t.Errorf("expected '%s', got '%s'", expected, result)
		}
	})

	t.Run("it leaves yt-dlp variables alone", func(t *testing.T) {
		result, err := core.OutputPathBuilderBuild("/videos/%(title)s.%(ext)s", map[string]string{})
		if err != nil {
			t.Errorf("got error: %v", err)
		}
		expected := "/videos/%(title)s.%(ext)s"
		if result != expected {
			t.Errorf("expected '%s', got '%s'", expected, result)
		}
	})

	t.Run("recursively expands variables", func(t *testing.T) {
		additionalOptions := map[string]string{
			"media_upload_date_index": "99",
		}

		result, err := core.OutputPathBuilderBuild("{{ season_episode_index_from_date }}.{{ ext }}", additionalOptions)
		if err != nil {
			t.Errorf("got error: %v", err)
		}
		expected := "s%(upload_date>%Y)Se%(upload_date>%m%d)S99.%(ext)S"
		if result != expected {
			t.Errorf("expected '%s', got '%s'", expected, result)
		}
	})
}
