package store_test

import (
	"net/url"
	"reflect"
	"testing"

	"github.com/mattbriancon/pinchflat/internal/store"
)

// mediaProfileErrorCases lists submitted forms (field -> value) and the exact
// errors expected when creating (existing nil) or updating a profile.
var mediaProfileErrorCases = []struct {
	name  string
	form  map[string]string
	multi map[string][]string
	want  map[string][]string
}{
	{"valid", map[string]string{"name": "a", "output_path_template": "x.{{ ext }}"}, nil, map[string][]string{}},
	{"blank name and template", map[string]string{"name": "", "output_path_template": ""}, nil,
		map[string][]string{"name": {"can't be blank"}, "output_path_template": {"can't be blank"}}},
	{"whitespace name", map[string]string{"name": "   ", "output_path_template": "x.{{ ext }}"}, nil,
		map[string][]string{"name": {"can't be blank"}}},
	{"template without ext", map[string]string{"name": "a", "output_path_template": "x.txt"}, nil,
		map[string][]string{"output_path_template": {"must end with .{{ ext }}"}}},
	{"template with old-style ext", map[string]string{"name": "a", "output_path_template": "x.%(ext)S"}, nil, map[string][]string{}},
	{"negative redownload", map[string]string{"name": "a", "output_path_template": "x.{{ ext }}", "redownload_delay_days": "-1"}, nil,
		map[string][]string{"redownload_delay_days": {"must be greater than or equal to 0"}}},
	{"non-numeric redownload", map[string]string{"name": "a", "output_path_template": "x.{{ ext }}", "redownload_delay_days": "abc"}, nil,
		map[string][]string{"redownload_delay_days": {"is invalid"}}},
	{"blank redownload", map[string]string{"name": "a", "output_path_template": "x.{{ ext }}", "redownload_delay_days": ""}, nil, map[string][]string{}},
	{"bad enums", map[string]string{"name": "a", "output_path_template": "x.{{ ext }}", "shorts_behaviour": "nope",
		"livestream_behaviour": "nope", "preferred_resolution": "1p", "sponsorblock_behaviour": "nope"}, nil,
		map[string][]string{
			"shorts_behaviour": {"is invalid"}, "livestream_behaviour": {"is invalid"},
			"preferred_resolution": {"is invalid"}, "sponsorblock_behaviour": {"is invalid"},
		}},
	{"bad bool", map[string]string{"name": "a", "output_path_template": "x.{{ ext }}", "download_subs": "maybe"}, nil,
		map[string][]string{"download_subs": {"is invalid"}}},
	{"scalar categories", map[string]string{"name": "a", "output_path_template": "x.{{ ext }}", "sponsorblock_categories": "sponsor"}, nil,
		map[string][]string{"sponsorblock_categories": {"is invalid"}}},
	{"parse error plus blank name", map[string]string{"name": "", "output_path_template": "x.{{ ext }}", "embed_subs": "zzz"}, nil,
		map[string][]string{"name": {"can't be blank"}, "embed_subs": {"is invalid"}}},
}

func formValues(form map[string]string, multi map[string][]string) url.Values {
	v := url.Values{}
	for k, s := range form {
		v.Set("media_profile["+k+"]", s)
	}
	for k, s := range multi {
		v["media_profile["+k+"][]"] = s
	}
	return v
}

// validateForm runs the parse and validation steps the controller does.
func validateForm(form url.Values, existing *store.MediaProfile) map[string][]string {
	p, errs := store.ParseMediaProfileParams(form)
	for f, msgs := range p.Validate(existing) {
		errs[f] = append(errs[f], msgs...)
	}
	return errs
}

func TestMediaProfileParams_Errors(t *testing.T) {
	t.Parallel()
	for _, tt := range mediaProfileErrorCases {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got := validateForm(formValues(tt.form, tt.multi), store.NewMediaProfile())
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("errors = %v, want %v", got, tt.want)
			}
		})
	}
}
