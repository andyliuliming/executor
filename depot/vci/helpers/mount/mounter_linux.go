package mount // import "code.cloudfoundry.org/executor/depot/vci/helpers/mount"
import (
	"fmt"
	"os/exec"
	"strings"
)

type MounterLinux struct {
	mounterPath string
}

func (mounter *MounterLinux) Mount(source string, target string, fstype string, options []string) error {
	fsTypesNeedMounter := sets.NewString("nfs", "glusterfs", "ceph", "cifs")
	if fsTypesNeedMounter.Has(fstype) {
		mounterPath = mounter.mounterPath
	}

	return mounter.doMount(mounterPath, defaultMountCommand, source, target, fstype, options)
}

func (mounter *MounterLinux) doMount(mounterPath string, mountCmd string, source string, target string, fstype string, options []string) error {
	mountArgs := makeMountArgs(source, target, fstype, options)
	if len(mounterPath) > 0 {
		mountArgs = append([]string{mountCmd}, mountArgs...)
		mountCmd = mounterPath
	}
	command := exec.Command(mountCmd, mountArgs...)
	output, err := command.CombinedOutput()
	if err != nil {
		args := strings.Join(mountArgs, " ")
		glog.Errorf("Mount failed: %v\nMounting command: %s\nMounting arguments: %s\nOutput: %s\n", err, mountCmd, args, string(output))
		return fmt.Errorf("mount failed: %v\nMounting command: %s\nMounting arguments: %s\nOutput: %s\n",
			err, mountCmd, args, string(output))
	}
	return err
}

func (mounter *MounterLinux) Unmount(target string) error {
	return nil
}

// makeMountArgs makes the arguments to the mount(8) command.
func makeMountArgs(source, target, fstype string, options []string) []string {
	// Build mount command as follows:
	//   mount [-t $fstype] [-o $options] [$source] $target
	mountArgs := []string{}
	if len(fstype) > 0 {
		mountArgs = append(mountArgs, "-t", fstype)
	}
	if len(options) > 0 {
		mountArgs = append(mountArgs, "-o", strings.Join(options, ","))
	}
	if len(source) > 0 {
		mountArgs = append(mountArgs, source)
	}
	mountArgs = append(mountArgs, target)

	return mountArgs
}

func NewMounterLinux(mounterPath string) Interface {
	return &MounterLinux{
		mounterPath: mounterPath,
	}
}
