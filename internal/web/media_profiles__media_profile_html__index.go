package web

// Plain-Go half of media_profile_html/index.html.heex: the <.table> props.

import "github.com/a-h/templ"

// mediaProfilesIndexTableProps is table/1's assigns for the media profiles
// index page.
func mediaProfilesIndexTableProps(profiles []MediaProfileWithCount) TableProps {
	rows := make([]any, len(profiles))
	for i, p := range profiles {
		rows[i] = p
	}
	return TableProps{
		Rows:       rows,
		TableClass: "text-black dark:text-white",
		Columns: []TableColumn{
			{
				Label: "Name",
				Class: "cell-wrap",
				Render: func(row any) templ.Component {
					return mediaProfilesIndexNameCell(row.(MediaProfileWithCount))
				},
			},
			{
				Label: "Preferred Resolution",
				Render: func(row any) templ.Component {
					return mediaProfilesIndexResolutionCell(row.(MediaProfileWithCount))
				},
			},
			{
				Label: "Sources",
				Render: func(row any) templ.Component {
					return mediaProfilesIndexSourcesCell(row.(MediaProfileWithCount))
				},
			},
			{
				Label: "",
				Class: "flex justify-end",
				Render: func(row any) templ.Component {
					return mediaProfilesIndexEditCell(row.(MediaProfileWithCount))
				},
			},
		},
	}
}
