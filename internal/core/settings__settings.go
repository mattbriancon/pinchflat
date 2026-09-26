package core

import (
	"context"
)

// SettingsRecord/0
func (a *App) SettingsRecord(ctx context.Context) (*Setting, error) {
	panic("unported: Pinchflat.Settings.record/0")
}

// SettingsUpdateSetting/2
func (a *App) SettingsUpdateSetting(ctx context.Context, setting *Setting, attrs Attrs) (*Setting, error) {
	panic("unported: Pinchflat.Settings.update_setting/2")
}

// SettingsSet/1
func (a *App) SettingsSet(ctx context.Context, kw KW) (any, error) {
	panic("unported: Pinchflat.Settings.set/1")
}

// SettingsGet/1
func (a *App) SettingsGet(ctx context.Context, name string) (any, error) {
	panic("unported: Pinchflat.Settings.get/1")
}

// SettingsGetBang/1
func (a *App) SettingsGetBang(ctx context.Context, name string) (any, error) {
	panic("unported: Pinchflat.Settings.get!/1")
}

// SettingsChangeSetting/2
func (a *App) SettingsChangeSetting(ctx context.Context, setting *Setting, attrs Attrs) *Changeset {
	panic("unported: Pinchflat.Settings.change_setting/2")
}
