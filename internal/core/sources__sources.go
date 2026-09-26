package core

import (
	"context"
)

// output_path_template/1
func (a *App) SourcesOutputPathTemplate(ctx context.Context, source *Source) string {
	panic("unported: Pinchflat.Sources.output_path_template/1")
}

// use_cookies?/2
func SourcesUseCookies(source *Source, operation string) bool {
	panic("unported: Pinchflat.Sources.use_cookies?/2")
}

// ListSources/0
func (a *App) SourcesListSources(ctx context.Context) ([]*Source, error) {
	panic("unported: Pinchflat.Sources.list_sources/0")
}

// ListSourcesFor/1
func (a *App) SourcesListSourcesFor(ctx context.Context, profile *MediaProfile) ([]*Source, error) {
	panic("unported: Pinchflat.Sources.list_sources_for/1")
}

// GetSourceBang/1
func (a *App) SourcesGetSource(ctx context.Context, id int64) (*Source, error) {
	panic("unported: Pinchflat.Sources.get_source!/1")
}

// CreateSource/1 and CreateSource/2
func (a *App) SourcesCreateSource(ctx context.Context, attrs Attrs, opts KW) (*Source, error) {
	panic("unported: Pinchflat.Sources.create_source/2")
}

// UpdateSource/2 and UpdateSource/3
func (a *App) SourcesUpdateSource(ctx context.Context, source *Source, attrs Attrs, opts KW) (*Source, error) {
	panic("unported: Pinchflat.Sources.update_source/3")
}

// DeleteSource/1 and DeleteSource/2
func (a *App) SourcesDeleteSource(ctx context.Context, source *Source, opts KW) (*Source, error) {
	panic("unported: Pinchflat.Sources.delete_source/2")
}

// ChangeSource/2 and ChangeSource/3
func (a *App) SourcesChangeSource(ctx context.Context, source *Source, attrs Attrs, validationStage string) *Changeset {
	panic("unported: Pinchflat.Sources.change_source/3")
}
