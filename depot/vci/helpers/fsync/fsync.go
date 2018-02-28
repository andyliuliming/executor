package fsync // import "code.cloudfoundry.org/executor/depot/vci/helpers/fsync"
import (
	"fmt"
	"os/exec"
	"strings"
)

type FSync interface {
	CopyFolder(src, dest string) error
}

type fSync struct {
}

const (
	// Default mount command if mounter path is not specified
	defaultRsyncCommand string = "rsync"
)

func NewFSync() FSync {
	return &fSync{}
}

func (f *fSync) CopyFolder(src, dest string) error {
	// 	const stagerScript = `
	// 	set -e
	// 	{{- range .BuildpackMD5s}}
	// 	su vcap -c "unzip -qq /tmp/{{.}}.zip -d /tmp/buildpacks/{{.}}" && rm /tmp/{{.}}.zip
	// 	{{- end}}

	// 	chown -R vcap:vcap /tmp/app /tmp/cache
	// 	{{if not .RSync}}exec {{end}}su vcap -p -c "PATH=$PATH exec /tmp/lifecycle/builder -buildpackOrder '$0' -skipDetect=$1"
	// 	{{- if .RSync}}
	// 	rsync -a /tmp/app/ /tmp/local/
	// 	{{- end}}
	// `
	// mountArgs := makeMountArgs(source, target, fstype, options)
	// if len(mounterPath) > 0 {
	// 	mountArgs = append([]string{mountCmd}, mountArgs...)
	// 	mountCmd = mounterPath
	// }
	return f.doRsync(defaultRsyncCommand, src, dest)
}

func (f *fSync) doRsync(rsyncCommand, src, dest string) error {
	rsyncArgs := makeRsyncArgs(src, dest)
	command := exec.Command("rsync", rsyncArgs...)
	output, err := command.CombinedOutput()
	if err != nil {
		args := strings.Join(rsyncArgs, " ")
		// glog.Errorf("Mount failed: %v\nMounting command: %s\nMounting arguments: %s\nOutput: %s\n", err, mountCmd, args, string(output))
		return fmt.Errorf("rsync failed: %v\nRsync command: %s\nRsync arguments: %s\nOutput: %s\n",
			err, rsyncCommand, args, string(output))
	}
	return err
}

// makeMountArgs makes the arguments to the mount(8) command.
func makeRsyncArgs(source, target string) []string {
	mountArgs := []string{}
	mountArgs = append(mountArgs, "-a", source, target)
	return mountArgs
}
