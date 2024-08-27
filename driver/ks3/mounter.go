package ks3

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"net"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/container-storage-interface/spec/lib/go/csi"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"k8s.io/klog/v2"
	"k8s.io/mount-utils"
	"os/exec"
)

// Mounter is responsible for formatting and mounting volumes
type Mounter interface {
	mount.Interface

	// ForceUnmount the given target
	ForceUnmount(target string) error
}

var DefaultMounter = NewMounter()

type mounter struct {
	mount.Interface
}

// NewMounter returns a new mounter instance
func NewMounter() Mounter {
	return &mounter{Interface: mount.New("")}
}

func (m *mounter) ForceUnmount(target string) error {
	umountCmd := "umount"
	if target == "" {
		return errors.New("target is not specified for unmounting the volume")
	}

	umountArgs := []string{"-f", target}

	klog.Infof("ForceUnmount %s, the command is %s %v", target, umountCmd, umountArgs)

	out, err := exec.Command(umountCmd, umountArgs...).CombinedOutput()
	if err != nil {
		return fmt.Errorf("unmounting failed: %v cmd: '%s -f %s' output: %q",
			err, umountCmd, target, string(out))
	}

	return nil
}

func CheckDeviceAvailable(devicePath string) error {
	if devicePath == "" {
		return status.Error(codes.Internal, "devicePath is empty, cannot used for Volume")
	}

	if _, err := os.Stat(devicePath); os.IsNotExist(err) {
		return err
	}

	// check the device is used for system
	if devicePath == "/dev/vda" || devicePath == "/dev/vda1" {
		return fmt.Errorf("devicePath(%s) is system device, cannot used for Volume", devicePath)
	}

	checkCmd := fmt.Sprintf("mount | grep \"%s on /var/lib/kubelet type\" | wc -l", devicePath)
	if out, err := run(checkCmd); err != nil {
		return fmt.Errorf("devicePath(%s) is used to kubelet", devicePath)
	} else if strings.TrimSpace(out) != "0" {
		return fmt.Errorf("devicePath(%s) is used as DataDisk for kubelet, cannot used fo Volume", devicePath)
	}
	return nil
}

func validateNodePublishVolumeRequest(req *csi.NodePublishVolumeRequest) error {
	if req.GetVolumeId() == "" {
		return errors.New("volume ID missing in request")
	}
	if req.GetTargetPath() == "" {
		return errors.New("target path missing in request")
	}
	if req.GetVolumeCapability() == nil {
		return errors.New("volume capability missing in request")
	}
	return nil
}

func validateNodeUnpublishVolumeRequest(req *csi.NodeUnpublishVolumeRequest) error {
	if req.GetVolumeId() == "" {
		return errors.New("volume ID missing in request")
	}
	if req.GetTargetPath() == "" {
		return errors.New("target path missing in request")
	}
	return nil
}

func parseKS3fsOptions(attributes map[string]string) (*ks3fsOptions, error) {
	options := &ks3fsOptions{}
	for k, v := range attributes {
		switch strings.ToLower(k) {
		case paramURL:
			options.URL = v
		case paramBucket:
			options.Bucket = v
		case paramPath:
			options.Path = v
		case paramAdditionalArgs:
			options.AdditionalArgs = v
		case paramDbgLevel:
			options.DbgLevel = v
		}
	}

	if options.DbgLevel == "" {
		options.DbgLevel = defaultDBGLevel
	}

	return options, validateKS3fsOptions(options)
}

func validateKS3fsOptions(options *ks3fsOptions) error {
	if options.URL == "" {
		return errors.New("KS3 service URL can't be empty")
	}
	if options.Bucket == "" {
		return errors.New("KS3 bucket can't be empty")
	}
	return nil
}

func createCredentialFile(volID, bucket string, secrets map[string]string) (string, error) {
	credential, err := getSecretCredential(secrets)
	if err != nil {
		klog.Errorf("getSecretCredential info from NodeStageSecrets failed: %v", err)
		return "", status.Errorf(codes.InvalidArgument, "get credential failed: %v", err)
	}

	// compute sha256 and add on password file name
	credSHA := sha256.New()
	credSHA.Write([]byte(credential))
	shaString := hex.EncodeToString(credSHA.Sum(nil))
	passwdFilename := fmt.Sprintf("%s%s_%s", ks3PasswordFileDirectory, bucket, shaString)

	klog.Infof("ks3fs password file name is %s", passwdFilename)

	if _, err := os.Stat(passwdFilename); err != nil {
		if os.IsNotExist(err) {
			if err := ioutil.WriteFile(passwdFilename, []byte(credential), 0600); err != nil {
				klog.Errorf("create password file for volume %s failed: %v", volID, err)
				return "", status.Errorf(codes.Internal, "create tmp password file failed: %v", err)
			}
		} else {
			klog.Errorf("stat password file  %s failed: %v", passwdFilename, err)
			return "", status.Errorf(codes.Internal, "stat password file failed: %v", err)
		}
	} else {
		klog.Infof("password file %s is exist, and sha256 is same", passwdFilename)
	}

	return passwdFilename, nil
}

func getSecretCredential(secrets map[string]string) (string, error) {
	sid := strings.TrimSpace(secrets[credentialID])
	skey := strings.TrimSpace(secrets[credentialKey])
	if sid == "" || skey == "" {
		return "", fmt.Errorf("secret must contains %v and %v", credentialID, credentialKey)
	}
	return strings.Join([]string{sid, skey}, ":"), nil
}

func ks3mount(options *ks3fsOptions, mountPoint string, credentialFilePath string) error {
	klog.V(2).Infof("KS3 mount socket")
	klog.V(2).Infof("KS3 mount options: %+v", options)
	httpClient := http.Client{
		Transport: &http.Transport{
			DialContext: func(_ context.Context, _, _ string) (net.Conn, error) {
				return net.Dial("unix", socketPath)
			},
		},
	}

	bucketOrWithSubDir := options.Bucket
	if options.Path != "" {
		bucketOrWithSubDir = fmt.Sprintf("%s:%s", options.Bucket, options.Path)
	}
	args := []string{
		bucketOrWithSubDir,
		mountPoint,
		"-ourl=" + options.URL,
		"-odbglevel=" + options.DbgLevel,
		"-opasswd_file=" + credentialFilePath,
	}
	if options.AdditionalArgs != "" {
		args = append(args, options.AdditionalArgs)
	}
	if options.NotsupCompatDir {
		args = append(args, "-onotsup_compat_dir")
	}

	body := make(map[string]string)
	body["command"] = fmt.Sprintf("s3fs %s", strings.Join(args, " "))
	bodyJson, _ := json.Marshal(body)
	response, err := httpClient.Post("http://unix/launcher", "application/json", strings.NewReader(string(bodyJson)))
	if err != nil {
		return err
	}

	defer response.Body.Close()
	respBody, err := ioutil.ReadAll(response.Body)
	if err != nil {
		return err
	}

	if response.StatusCode != http.StatusOK {
		return fmt.Errorf("the response of launcher(action: s3fs) is: %v", string(respBody))
	}

	klog.Info("send s3fs command to launcher successfully")

	return nil
}

func checkKS3Mounted(mountPoint string) error {
	// Wait until KS3 is successfully mounted.
	// Totally 4 seconds
	retryTimes := 20
	interval := time.Millisecond * 200
	notMnt := true
	var err error
	for i := 0; i < retryTimes; i++ {
		if notMnt, err = DefaultMounter.IsLikelyNotMountPoint(mountPoint); err == nil {
			if !notMnt {
				break
			} else {
				time.Sleep(interval)
			}
		} else {
			return err
		}
	}
	if notMnt {
		return errors.New("check ks3 mounted timeout")
	}
	return nil
}
