package core_test

import (
	"testing"

	"github.com/mattbriancon/pinchflat/internal/core"
	"github.com/mattbriancon/pinchflat/internal/core/coretest"
)

func TestSources_Schema(t *testing.T) {
	ta := coretest.NewApp(t)
	defer ta.App.DB.Close()

	t.Run("source_metadata is deleted when the source is deleted", func(t *testing.T) {
	})

	t.Run("can be JSON encoded without error", func(t *testing.T) {
	})
}

func TestSources_OutputPathTemplate(t *testing.T) {
	ta := coretest.NewApp(t)
	defer ta.App.DB.Close()

	t.Run("returns the source's override if present", func(t *testing.T) {
	})

	t.Run("returns the media profile's template if no override is present", func(t *testing.T) {
	})

	t.Run("Treats empty strings as being blank", func(t *testing.T) {
	})
}

func TestSources_UseCookies(t *testing.T) {
	t.Run("returns true if the source has been set to use cookies", func(t *testing.T) {
		source := &core.Source{CookieBehaviour: core.SourceCookieBehaviourAllOperations}
		if !core.SourcesUseCookies(source, "downloading") {
			t.Error("expected true")
		}
	})

	t.Run("returns false if the source has not been set to use cookies", func(t *testing.T) {
		source := &core.Source{CookieBehaviour: core.SourceCookieBehaviourDisabled}
		if core.SourcesUseCookies(source, "downloading") {
			t.Error("expected false")
		}
	})

	t.Run("returns true if the action is indexing and the source is set to :when_needed", func(t *testing.T) {
		source := &core.Source{CookieBehaviour: core.SourceCookieBehaviourWhenNeeded}
		if !core.SourcesUseCookies(source, "indexing") {
			t.Error("expected true")
		}
	})

	t.Run("returns false if the action is downloading and the source is set to :when_needed", func(t *testing.T) {
		source := &core.Source{CookieBehaviour: core.SourceCookieBehaviourWhenNeeded}
		if core.SourcesUseCookies(source, "downloading") {
			t.Error("expected false")
		}
	})

	t.Run("returns true if the action is error_recovery and the source is set to :when_needed", func(t *testing.T) {
		source := &core.Source{CookieBehaviour: core.SourceCookieBehaviourWhenNeeded}
		if !core.SourcesUseCookies(source, "error_recovery") {
			t.Error("expected true")
		}
	})
}

func TestSources_ListSources(t *testing.T) {
	ta := coretest.NewApp(t)
	defer ta.App.DB.Close()

	t.Run("it returns all sources", func(t *testing.T) {
	})
}

func TestSources_ListSourcesFor(t *testing.T) {
	ta := coretest.NewApp(t)
	defer ta.App.DB.Close()

	t.Run("returns all sources for a given media profile", func(t *testing.T) {
	})
}

func TestSources_GetSource(t *testing.T) {
	ta := coretest.NewApp(t)
	defer ta.App.DB.Close()

	t.Run("it returns the source with given id", func(t *testing.T) {
	})
}

func TestSources_CreateSource(t *testing.T) {
	ta := coretest.NewApp(t)
	defer ta.App.DB.Close()

	t.Run("automatically sets the UUID", func(t *testing.T) {
	})

	t.Run("UUID is not writable by the user", func(t *testing.T) {
	})

	t.Run("creates a source and adds name + ID from runner response for channels", func(t *testing.T) {
	})

	t.Run("creates a source and adds name + ID for playlists", func(t *testing.T) {
	})

	t.Run("adds an error if the runner fails", func(t *testing.T) {
	})

	t.Run("adds an error if the runner succeeds but the result was invalid JSON", func(t *testing.T) {
	})

	t.Run("you can specify a custom custom_name", func(t *testing.T) {
	})

	t.Run("friendly name is pulled from collection_name if not specified", func(t *testing.T) {
	})

	t.Run("creation enforces uniqueness of collection_id scoped to the media_profile and title regex", func(t *testing.T) {
	})

	t.Run("creation lets you duplicate collection_ids and profiles as long as the regex is different", func(t *testing.T) {
	})

	t.Run("creation lets you duplicate collection_ids as long as the media profile is different", func(t *testing.T) {
	})

	t.Run("collection_type is inferred from source details", func(t *testing.T) {
	})

	t.Run("creation with invalid data returns error changeset", func(t *testing.T) {
		attrs := core.Attrs{"name": nil, "collection_id": nil}
		_, err := ta.App.SourcesCreateSource(ta.Ctx, attrs, core.KW{})
		if err == nil {
			t.Error("expected error")
		}
		cs, ok := core.AsChangesetError(err)
		if !ok {
			t.Fatalf("expected changeset error, got %T", err)
		}
		if cs.Valid() {
			t.Error("expected changeset to be invalid, but it was valid")
		}
	})

	t.Run("creation with invalid data fails fast and does not call the runner", func(t *testing.T) {
	})

	t.Run("creation will schedule the indexing task", func(t *testing.T) {
	})

	t.Run("creation will schedule a fast indexing job if the fast_index option is set", func(t *testing.T) {
	})

	t.Run("creation will not schedule a fast indexing job if the fast_index option is not set", func(t *testing.T) {
	})

	t.Run("creation schedules an index test even if the index frequency is 0", func(t *testing.T) {
	})

	t.Run("fast_index forces the index frequency to be a default value", func(t *testing.T) {
	})

	t.Run("disabling fast index will not change the index frequency", func(t *testing.T) {
	})

	t.Run("creating will kickoff a metadata storage worker", func(t *testing.T) {
	})
}

func TestSources_CreateSourceWhenTestingYtDlpOptions(t *testing.T) {
	ta := coretest.NewApp(t)
	defer ta.App.DB.Close()

	t.Run("sets use_cookies to true if the source has been set to use cookies", func(t *testing.T) {
	})

	t.Run("does not set use_cookies if the source uses cookies when needed", func(t *testing.T) {
	})

	t.Run("does not set use_cookies if the source has not been set to use cookies", func(t *testing.T) {
	})

	t.Run("skips sleep interval", func(t *testing.T) {
	})
}

func TestSources_CreateSourceWhenTestingOptions(t *testing.T) {
	ta := coretest.NewApp(t)
	defer ta.App.DB.Close()

	t.Run("run_post_commit_tasks: false won't enqueue post-commit tasks", func(t *testing.T) {
	})
}

func TestSources_UpdateSource(t *testing.T) {
	ta := coretest.NewApp(t)
	defer ta.App.DB.Close()

	t.Run("updates with valid data updates the source", func(t *testing.T) {
	})

	t.Run("updates with invalid data fails fast and does not call the runner", func(t *testing.T) {
	})

	t.Run("updating the original_url will re-fetch the source details for channels", func(t *testing.T) {
	})

	t.Run("updating the original_url will re-fetch the source details for playlists", func(t *testing.T) {
	})

	t.Run("not updating the original_url will not re-fetch the source details", func(t *testing.T) {
	})

	t.Run("updates with invalid data returns error changeset", func(t *testing.T) {
	})

	t.Run("updating will kickoff a metadata storage worker if the original_url changes", func(t *testing.T) {
	})

	t.Run("updating will not kickoff a metadata storage worker other attrs change", func(t *testing.T) {
	})
}

func TestSources_UpdateSourceWhenTestingMediaDownloadTasks(t *testing.T) {
	ta := coretest.NewApp(t)
	defer ta.App.DB.Close()

	t.Run("enabling the download_media attribute will schedule a download task", func(t *testing.T) {
	})

	t.Run("disabling the download_media attribute will cancel the download task", func(t *testing.T) {
	})

	t.Run("enabling download_media will not schedule a task if the source is disabled", func(t *testing.T) {
	})

	t.Run("disabling a source will cancel any pending download tasks", func(t *testing.T) {
	})

	t.Run("enabling a source will schedule a download task if download_media is true", func(t *testing.T) {
	})

	t.Run("enabling a source will not schedule a download task if download_media is false", func(t *testing.T) {
	})
}

func TestSources_UpdateSourceWhenTestingSlowIndexing(t *testing.T) {
	ta := coretest.NewApp(t)
	defer ta.App.DB.Close()

	t.Run("updating the index frequency to >0 will re-schedule the indexing task", func(t *testing.T) {
	})

	t.Run("updating the index frequency to 0 will not re-schedule the indexing task", func(t *testing.T) {
	})

	t.Run("updating the index frequency to 0 will delete any pending tasks", func(t *testing.T) {
	})

	t.Run("not updating the index frequency will not re-schedule the indexing task or delete tasks", func(t *testing.T) {
	})

	t.Run("disabling a source will delete any pending tasks", func(t *testing.T) {
	})

	t.Run("updating the index frequency will not create a task if the source is disabled", func(t *testing.T) {
	})

	t.Run("enabling a source will create a task if the index frequency is >0", func(t *testing.T) {
	})

	t.Run("enabling a source will not create a task if the index frequency is 0", func(t *testing.T) {
	})
}

func TestSources_UpdateSourceWhenTestingFastIndexing(t *testing.T) {
	ta := coretest.NewApp(t)
	defer ta.App.DB.Close()

	t.Run("enabling fast_index will schedule a fast indexing task", func(t *testing.T) {
	})

	t.Run("disabling fast_index will cancel the fast indexing task", func(t *testing.T) {
	})

	t.Run("fast_index forces the index frequency to be a default value", func(t *testing.T) {
	})

	t.Run("disabling fast index will not change the index frequency", func(t *testing.T) {
	})

	t.Run("disabling a source will delete any pending tasks", func(t *testing.T) {
	})

	t.Run("updating fast indexing will not create a task if the source is disabled", func(t *testing.T) {
	})

	t.Run("enabling a source will create a task if fast_index is true", func(t *testing.T) {
	})

	t.Run("enabling a source will not create a task if fast_index is false", func(t *testing.T) {
	})
}

func TestSources_UpdateSourceWhenTestingOptions(t *testing.T) {
	ta := coretest.NewApp(t)
	defer ta.App.DB.Close()

	t.Run("run_post_commit_tasks: false won't enqueue post-commit tasks", func(t *testing.T) {
	})
}

func TestSources_DeleteSource(t *testing.T) {
	ta := coretest.NewApp(t)
	defer ta.App.DB.Close()

	t.Run("it deletes the source", func(t *testing.T) {
	})

	t.Run("it returns a source changeset", func(t *testing.T) {
	})

	t.Run("deletion also deletes all associated tasks", func(t *testing.T) {
	})

	t.Run("deletion also deletes all associated media items", func(t *testing.T) {
	})

	t.Run("deletion does not delete media files by default", func(t *testing.T) {
	})

	t.Run("deletes the source's metadata files", func(t *testing.T) {
	})

	t.Run("does not delete the source's non-metadata files", func(t *testing.T) {
	})
}

func TestSources_DeleteSourceWhenDeletingFiles(t *testing.T) {
	ta := coretest.NewApp(t)
	defer ta.App.DB.Close()

	t.Run("deletes source and media_items", func(t *testing.T) {
	})

	t.Run("also deletes media files", func(t *testing.T) {
	})

	t.Run("deletes the source's non-metadata files", func(t *testing.T) {
	})
}

func TestSources_ChangeSource(t *testing.T) {
	ta := coretest.NewApp(t)
	defer ta.App.DB.Close()

	t.Run("it returns a changeset", func(t *testing.T) {
	})
}

func TestSources_ChangeSourceWhenTestingRegexValidation(t *testing.T) {
	ta := coretest.NewApp(t)
	defer ta.App.DB.Close()

	t.Run("succeeds when a valid regex is provided", func(t *testing.T) {
	})

	t.Run("succeeds when a regex is set back to nil", func(t *testing.T) {
	})

	t.Run("fails when an invalid regex is provided", func(t *testing.T) {
	})
}

func TestSources_ChangeSourceWhenTestingMinMaxDurationValidations(t *testing.T) {
	ta := coretest.NewApp(t)
	defer ta.App.DB.Close()

	t.Run("succeeds if min and max are nil", func(t *testing.T) {
	})

	t.Run("succeeds if either min or max is nil", func(t *testing.T) {
	})

	t.Run("succeeds if min is less than max", func(t *testing.T) {
	})

	t.Run("fails if min is greater than or equal to max", func(t *testing.T) {
	})
}

func TestSources_ChangeSourceWhenTestingOriginalURLValidation(t *testing.T) {
	ta := coretest.NewApp(t)
	defer ta.App.DB.Close()

	t.Run("succeeds when an original URL is valid", func(t *testing.T) {
	})

	t.Run("fails when an original URL points to a video", func(t *testing.T) {
	})

	t.Run("passes when a non-youtube link is provided", func(t *testing.T) {
	})
}
