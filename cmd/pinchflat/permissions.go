package main

// Port of Pinchflat.Release.check_file_permissions/0 (rel/overlays/bin/check_file_permissions).

import (
	"errors"
	"fmt"
	"io/fs"
	"log/slog"
	"os"
	"path/filepath"
)

func checkFilePermissions(dirs []string) error {
	for _, dir := range dirs {
		slog.Info("Checking permissions for " + dir)
		p := filepath.Join(dir, ".keep")
		err := os.MkdirAll(dir, 0o755)
		if err == nil {
			err = os.WriteFile(p, nil, 0o644)
		}
		switch {
		case err == nil:
			slog.Info("Permissions OK")
		case errors.Is(err, fs.ErrPermission):
			slog.Error(permissionDeniedScreed(dir))
			return fmt.Errorf("Permission denied")
		default:
			slog.Error(fmt.Sprintf("Permissions check failed: %v", err))
			return fmt.Errorf("Unknown error")
		}
	}
	return nil
}

func permissionDeniedScreed(dir string) string {
	return fmt.Sprintf(`The directory "%[1]s" is not writeable by the Docker container.

Please ensure that the directory exists and is writeable by the Docker
container. All setups are different, but you may be able to run something
like this on the *host*:

  chown nobody -R <host path that maps to %[1]s>
  chmod 755 -R <host path that maps to %[1]s>

Swapping in your real host path. Then, you should set the user running
this container by editing your `+"`docker run`"+` command like so:

    docker run --user 99:100 <rest of the command>

Or adding `+"`user: '99:100'`"+` to the Pinchflat service of your Docker Compose
file. Again, there are many ways to do this depending on your setup and
this is just one example. See issue #106 in the Pinchflat Github for more.

No matter the case, this _is_ a permissions error and allowing the container
to write to the directory is the only way to fix it. It is not recommended
to run the container as `+"`root`"+` because files created by Pinchflat may not
be accessible to other apps that want to modify them.
`, dir)
}
