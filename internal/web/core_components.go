package web

// Port of lib/pinchflat_web/components/core_components.ex. The renderable
// pieces live in core_components.templ (templ requires its own file type);
// this file holds the plain-Go parts: prop structs and the `input/1`
// multi-clause dispatch, which templ's DSL can't express directly.

import (
	"strings"

	"github.com/a-h/templ"
)

// CoreSelectOption is Phoenix.HTML.Form.options_for_select/2's {label, value}.
type CoreSelectOption struct {
	Label string
	Value string
}

// CoreListItem is the :item slot on list/1.
type CoreListItem struct {
	Title   string
	Content templ.Component
}

// CoreInputProps is attr :id/:name/:label/... on core_components.ex's input/1.
// Pass Field (a Form.Field) the way Elixir pattern-matches on
// %Phoenix.HTML.FormField{}; anything set explicitly (ID, Name, Value,
// Errors) takes priority, matching Elixir's assign_new/3.
type CoreInputProps struct {
	ID          string
	Name        string
	Label       string
	LabelSuffix string
	Value       any
	Help        string
	HTMLHelp    bool
	Type        string // default "text"
	Field       *FormField
	Errors      []string
	Checked     *bool // nil => derived from Value, like Phoenix's normalize_value
	Prompt      string
	Options     []CoreSelectOption
	Multiple    bool
	InputClass  string
	Rest        templ.Attributes
	InputAppend templ.Component
}

// resolveCoreInputProps is input(%{field: %Phoenix.HTML.FormField{}} = assigns).
func resolveCoreInputProps(p CoreInputProps) CoreInputProps {
	if p.Field != nil {
		f := p.Field
		if p.ID == "" {
			p.ID = f.ID
		}
		if p.Errors == nil {
			p.Errors = f.Errors
		}
		if p.Name == "" {
			if p.Multiple {
				p.Name = f.Name + "[]"
			} else {
				p.Name = f.Name
			}
		}
		if p.Value == nil {
			p.Value = f.Value
		}
	}
	if p.Type == "" {
		p.Type = "text"
	}
	if p.Checked == nil {
		c := Checked(p.Value)
		p.Checked = &c
	}
	if p.Errors == nil {
		p.Errors = []string{}
	}
	return p
}

// CoreInput is core_components.ex's input/1: it dispatches on @type the way
// the Elixir multi-clause function does.
func CoreInput(p CoreInputProps) templ.Component {
	p = resolveCoreInputProps(p)
	switch p.Type {
	case "checkbox":
		return coreInputCheckbox(p)
	case "checkbox_group":
		return coreInputCheckboxGroup(p)
	case "toggle":
		return coreInputToggle(p)
	case "select":
		return coreInputSelect(p)
	case "textarea":
		return coreInputTextarea(p)
	default:
		return coreInputDefault(p)
	}
}

// coreOptionSelected is Phoenix.HTML.Form.options_for_select/2's selected
// comparison (string equality against the normalized current value).
func coreOptionSelected(value any, option string) bool {
	return InputValue(value) == option
}

// coreOptionsInList is `option_value in @value` for checkbox_group.
func coreOptionsInList(value any, option string) bool {
	list, _ := value.([]string)
	for _, v := range list {
		if v == option {
			return true
		}
	}
	return false
}

// TranslateError is core_components.ex's translate_error/1. The Go port has
// no Gettext backend, so it returns the message unchanged (a documented
// Phase 1 deviation; see the report).
func TranslateError(msg string) string { return msg }

// TranslateErrors is translate_errors/2.
func TranslateErrors(errors []string) []string { return errors }

func coreJoinNonEmpty(parts ...string) string {
	out := make([]string, 0, len(parts))
	for _, p := range parts {
		if p != "" {
			out = append(out, p)
		}
	}
	return strings.Join(out, " ")
}
