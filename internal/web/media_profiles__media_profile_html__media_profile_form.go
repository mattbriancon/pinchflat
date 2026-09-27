package web

// Plain-Go half of media_profile_form.html.heex (attribute helpers that
// templ's DSL can't build inline).

import "github.com/a-h/templ"

// mediaProfilesFormAttrs are the form tag's rest attributes: action, method
// (always "post" on the wire; method="patch" adds a hidden _method field,
// see mediaProfilesMethodOverride) and the Alpine advancedMode toggle.
func mediaProfilesFormAttrs(action string) templ.Attributes {
	return templ.Attributes{
		"action": action,
		"method": "post",
		"x-data": "{ advancedMode: !!JSON.parse(localStorage.getItem('advancedMode')) }",
		"x-init": "$watch('advancedMode', value => localStorage.setItem('advancedMode', JSON.stringify(value)))",
	}
}

// mediaProfilesField is f[:name]: Form.Field returns a value, but
// CoreInputProps wants a pointer.
func mediaProfilesField(f *Form, name string) *FormField {
	ff := f.Field(name)
	return &ff
}
