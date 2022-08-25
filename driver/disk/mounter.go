package driver

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"strings"

	"github.com/golang/glog"
	mountutils "k8s.io/mount-utils"
)

type findmntResponse struct {
	FileSystems []fileSystem `json:"filesystems"`
}

type fileSystem struct {
	Target      string `json:"target"`
	Propagation string `json:"propagation"`
	FsType      string `json:"fstype"`
	Options     string `json:"options"`
}

// Mounter is responsible for formatting and mounting volumes
type Mounter interface {
	// Format formats the source with the given filesystem type
	Format(source, fsType string) error

	// Mount mounts source to target with the given fstype and options.
	Mount(source, target, fsType string, options ...string) error

	// Unmount unmounts the given target
	Unmount(target string) error

	// IsFormatted checks whether the source device is formatted or not. It
	// returns true if the source device is already formatted.
	IsFormatted(source string) (bool, error)

	// IsMounted checks whether the target path is a correct mount (i.e:
	// propagated). It returns true if it's mounted. An error is returned in
	// case of system errors or if it's mounted incorrectly.
	IsMounted(target string) (bool, error)
	//Expand FileSystem only xfs and ext*(2,3,4) support expand
	Expand(fsType, source string) (bool, error)

	PathExists(path string) (bool, error)
}

// TODO(arslan): this is Linux only for now. Refactor this into a package with
// architecture specific code in the future, such as mounter_darwin.go,
// mounter_linux.go, etc..
type mounter struct {
}

// newMounter returns a new mounter instance
func newMounter() *mounter {
	return &mounter{}
}

// This function is mirrored in ./sanity_test.go to make sure sanity test covered this block of code
// Please mirror the change to func MakeFile in ./sanity_test.go
func (m *mounter) PathExists(path string) (bool, error) {
	return mountutils.PathExists(path)
}

func (m *mounter) Format(source, fsType string) error {
	mkfsCmd := fmt.Sprintf("mkfs.%s", fsType)

	_, err := exec.LookPath(mkfsCmd)
	if err != nil {
		if err == exec.ErrNotFound {
			return fmt.Errorf("%q executable not found in $PATH", mkfsCmd)
		}
		return err
	}

	mkfsArgs := []string{}

	if fsType == "" {
		return errors.New("fs type is not specified for formatting the volume")
	}

	if source == "" {
		return errors.New("source is not specified for formatting the volume")
	}

	mkfsArgs = append(mkfsArgs, source)
	if fsType == "ext4" || fsType == "ext3" {
		mkfsArgs = []string{"-F", source}
	}

	glog.Infof("executing format command, cmd: %v, args: %v", mkfsCmd, mkfsArgs)
	out, err := exec.Command(mkfsCmd, mkfsArgs...).CombinedOutput()
	if err != nil {
		//  TODO error:
		/**
		formatting disk failed:
		exit status 1
		cmd: 'mkfs.ext4 -F /dev/disk/by-id/virtio-04eb4eb8-9894-417f-8'
		output: "mke2fs 1.45.2 (27-May-2019)\nThe file /dev/disk/by-id/virtio-04eb4eb8-9894-417f-8 does not exist and no size was specified.
		*/
		// test exec partprobe
		return fmt.Errorf("formatting disk failed: %v cmd: '%s %s' output: %q",
			err, mkfsCmd, strings.Join(mkfsArgs, " "), string(out))
	}

	return nil
}

func (m *mounter) Mount(source, target, fsType string, opts ...string) error {
	mountCmd := "mount"
	mountArgs := []string{}

	if fsType == "" {
		return errors.New("fs type is not specified for mounting the volume")
	}

	if source == "" {
		return errors.New("source is not specified for mounting the volume")
	}

	if target == "" {
		return errors.New("target is not specified for mounting the volume")
	}

	mountArgs = append(mountArgs, "-t", fsType)

	if len(opts) > 0 {
		mountArgs = append(mountArgs, "-o", strings.Join(opts, ","))
	}

	mountArgs = append(mountArgs, source)
	mountArgs = append(mountArgs, target)

	// create target, os.Mkdirall is noop if it exists
	// 0755 保持与kublet 创建目录文件权限一致
	// 0777 暂时兼容非root权限使用csi的fsgroup不生效,导致无法读写的问题
	err := os.MkdirAll(target, os.FileMode(0755))
	if err != nil {
		return err
	}

	fileinfo, err := os.Stat(source)
	if err != nil {
		return err
	}
	mode10 := uint32(fileinfo.Mode().Perm())
	// if mode10 == 493 || mode10 == 488 {
	// 	out, err := exec.Command("chmod", []string{"777", target}...).CombinedOutput()
	// 	if err != nil {
	// 		return fmt.Errorf("chmod failed: %v cmd: '%s %s' output: %q",
	// 			err, mountCmd, strings.Join(mountArgs, " "), string(out))
	// 	}

	// 	out, err = exec.Command("chmod", []string{"g+s", target}...).CombinedOutput()
	// 	if err != nil {
	// 		return fmt.Errorf("chmod failed: %v cmd: '%s %s' output: %q",
	// 			err, mountCmd, strings.Join(mountArgs, " "), string(out))
	// 	}

	// }
	// fileinfo, _ = os.Stat(target)
	glog.Infof("source mode: %d", uint32(fileinfo.Mode().Perm()))
	glog.Infof("executing mount command, cmd: %v, args: %v", mountCmd, mountArgs)
	//err = syscall.Mount(source, target, fsType, 0, "")
	out, err := exec.Command(mountCmd, mountArgs...).CombinedOutput()
	if err != nil {
		return fmt.Errorf("mounting failed: %v cmd: '%s %s' output: %q",
			err, mountCmd, strings.Join(mountArgs, " "), string(out))
	}

	if mode10 < 511 {
		out, err := exec.Command("chmod", []string{"777", target}...).CombinedOutput()
		if err != nil {
			return fmt.Errorf("chmod failed: %v cmd: '%s %s' output: %q",
				err, mountCmd, strings.Join(mountArgs, " "), string(out))
		}

		out, err = exec.Command("chmod", []string{"g+s", target}...).CombinedOutput()
		if err != nil {
			return fmt.Errorf("chmod failed: %v cmd: '%s %s' output: %q",
				err, mountCmd, strings.Join(mountArgs, " "), string(out))
		}

	}

	return nil
}

func (m *mounter) Unmount(target string) error {
	umountCmd := "umount"
	if target == "" {
		return errors.New("target is not specified for unmounting the volume")
	}

	umountArgs := []string{target}

	glog.Infof("executing umount command, cmd: %v, args: %v", umountCmd, umountArgs)
	out, err := exec.Command(umountCmd, umountArgs...).CombinedOutput()
	if err != nil {
		return fmt.Errorf("unmounting failed: %v cmd: '%s %s' output: %q",
			err, umountCmd, target, string(out))
	}
	glog.Infof("executing umount command result: %s", string(out))
	return nil
}

func (m *mounter) IsFormatted(source string) (bool, error) {
	if source == "" {
		return false, errors.New("source is not specified")
	}

	blkidCmd := "blkid"
	_, err := exec.LookPath(blkidCmd)
	if err != nil {
		if err == exec.ErrNotFound {
			return false, fmt.Errorf("%q executable not found in $PATH", blkidCmd)
		}
		return false, err
	}

	blkidArgs := []string{source}

	glog.Infof("checking if source is formatted, cmd: %v, args: %v", blkidCmd, blkidArgs)
	out, _ := exec.Command(blkidCmd, blkidArgs...).CombinedOutput()
	glog.Infof("exec blkid cmd, return %s.", string(out))
	if strings.TrimSpace(string(out)) == "" {
		return false, nil
	}

	return true, nil
}

func (m *mounter) IsMounted(target string) (bool, error) {
	if target == "" {
		return false, errors.New("target is not specified for checking the mount")
	}

	findmntCmd := "findmnt"
	_, err := exec.LookPath(findmntCmd)
	if err != nil {
		if err == exec.ErrNotFound {
			return false, fmt.Errorf("%q executable not found in $PATH", findmntCmd)
		}
		return false, err
	}

	findmntArgs := []string{"-o", "TARGET,PROPAGATION,FSTYPE,OPTIONS", "-M", target, "-J"}

	glog.Infof("checking if target is mounted, cmd: %v, args: %v", findmntCmd, findmntArgs)
	out, err := exec.Command(findmntCmd, findmntArgs...).CombinedOutput()
	if err != nil {
		// findmnt exits with non zero exit status if it couldn't find anything
		if strings.TrimSpace(string(out)) == "" {
			return false, nil
		}

		return false, fmt.Errorf("checking mounted failed: %v cmd: %q output: %q",
			err, findmntCmd, string(out))
	}

	// no response means there is no mount
	if string(out) == "" {
		return false, nil
	}

	var resp *findmntResponse
	err = json.Unmarshal(out, &resp)
	if err != nil {
		return false, fmt.Errorf("couldn't unmarshal data: %q: %s", string(out), err)
	}

	targetFound := false
	for _, fs := range resp.FileSystems {
		// check if the mount is propagated correctly. It should be set to shared.
		if fs.Propagation != "shared" {
			return true, fmt.Errorf("mount propagation for target %q is not enabled", target)
		}

		// the mountpoint should match as well
		if fs.Target == target {
			targetFound = true
		}
	}

	return targetFound, nil
}

// Expand 当前扩容文件系统方式是： 扩展裸盘文件系统
func (m *mounter) Expand(fsType, source string) (bool, error) {
	expandCmdForEXT := "resize2fs"
	expandCmdForXFS := "xfs_growfs"
	if fsType == "xfs" {
		out, err := exec.Command(expandCmdForXFS, []string{source}...).CombinedOutput()
		if err != nil {
			if strings.TrimSpace(string(out)) == "" {
				return false, nil
			}
			return false, fmt.Errorf("xfs filesystem expand failed: %v cmd: %q output: %q",
				err, expandCmdForXFS, string(out))
		} else {
			return true, nil
		}
	}
	if fsType == "ext4" || fsType == "ext3" || fsType == "ext2" {
		out, err := exec.Command(expandCmdForEXT, []string{source}...).CombinedOutput()
		if err != nil {
			if strings.TrimSpace(string(out)) == "" {
				return false, nil
			}
			return false, fmt.Errorf("ext filesystem expand failed: %v cmd: %q output: %q",
				err, expandCmdForEXT, string(out))
		} else {
			return true, nil
		}
	}
	return false, errors.New("not supported fs type")
}
