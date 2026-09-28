package web

// Plain-Go half of the "Sources" tab table on media_profile_html/show.html.heex.

import (
	"github.com/a-h/templ"
	"github.com/mattbriancon/pinchflat/internal/store"
)

func mediaProfilesShowSourcesTableProps(sources []*store.Source) TableProps {
	rows := make([]any, len(sources))
	for i, s := range sources {
		rows[i] = s
	}
	return TableProps{
		Rows:       rows,
		TableClass: "text-black dark:text-white",
		Columns: []TableColumn{
			{
				Label: "Name",
				Render: func(row any) templ.Component {
					return mediaProfilesShowSourceNameCell(row.(*store.Source))
				},
			},
			{
				Label: "Type",
				Render: func(row any) templ.Component {
					return mediaProfilesShowSourceTypeCell(row.(*store.Source))
				},
			},
			{
				Label: "Should Download?",
				Render: func(row any) templ.Component {
					return mediaProfilesShowSourceDownloadCell(row.(*store.Source))
				},
			},
			{
				Label: "",
				Class: "flex justify-end",
				Render: func(row any) templ.Component {
					return mediaProfilesShowSourceEditCell(row.(*store.Source))
				},
			},
		},
	}
}
