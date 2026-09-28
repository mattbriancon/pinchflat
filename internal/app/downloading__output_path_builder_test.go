package app_test

import (
	"testing"

	"github.com/mattbriancon/pinchflat/internal/app"
)

func TestOutputPathBuilder_Build(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		input   string
		options map[string]string
		want    string
	}{
		{
			name:    "expands 'standard' curly brace variables in the template",
			input:   "/videos/{{ title }}.{{ ext }}",
			options: map[string]string{},
			want:    "/videos/%(title)S.%(ext)S",
		},
		{
			name:    "expands 'custom' curly brace variables in the template",
			input:   "/videos/{{ upload_year }}.{{ ext }}",
			options: map[string]string{},
			want:    "/videos/%(upload_date>%Y)S.%(ext)S",
		},
		{
			name:    "respects additional options",
			input:   "/videos/{{ custom }}.{{ ext }}",
			options: map[string]string{"custom": "test"},
			want:    "/videos/test.%(ext)S",
		},
		{
			name:    "leaves yt-dlp variables alone",
			input:   "/videos/%(title)s.%(ext)s",
			options: map[string]string{},
			want:    "/videos/%(title)s.%(ext)s",
		},
		{
			name:  "recursively expands variables",
			input: "{{ season_episode_index_from_date }}.{{ ext }}",
			options: map[string]string{
				"media_upload_date_index": "99",
			},
			want: "s%(upload_date>%Y)Se%(upload_date>%m%d)S99.%(ext)S",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := app.OutputPathBuilderBuild(tt.input, tt.options)
			if err != nil {
				t.Errorf("got error: %v", err)
			}
			if result != tt.want {
				t.Errorf("expected '%s', got '%s'", tt.want, result)
			}
		})
	}
}
