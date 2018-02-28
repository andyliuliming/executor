package helpers

import (
	"fmt"
	"io/ioutil"

	"code.cloudfoundry.org/executor/depot/vci/helpers/fsync"
	"code.cloudfoundry.org/executor/depot/vci/helpers/mount"
	"code.cloudfoundry.org/lager"
)

// copy one folder to one azure share.

type VSync struct {
	logger lager.Logger
}

func NewVSync(logger lager.Logger) *VSync {
	return &VSync{
		logger: logger,
	}
}

func (v *VSync) CopyFolderToAzureShare(src, storageID, storageSecret, shareName string) error {
	// vers=<smb-version>,username=<storage-account-name>,password=<storage-account-key>,dir_mode=0777,file_mode=0777
	//,serverino
	options := []string{
		"vers=3.0",
		fmt.Sprintf("username=%s", storageID),
		fmt.Sprintf("password=%s", storageSecret),
		"dir_mode=0777,file_mode=0777,serverino",
	}

	mounter := mount.NewMounter()
	// io.Temp

	tempFolder, err := ioutil.TempDir("/tmp", "folder_to_azure")
	if err == nil {
		// tmpl, err := template.New("").Parse("//{{.storageid}}.file.core.windows.net/{{.sharename}}")
		// if err == nil {
		// scriptBuf := &bytes.Buffer{}
		// tmpl.Execute(scriptBuf, struct {
		// 	storageid string
		// 	sharename string
		// }{
		// 	storageID,
		// 	storageSecret,
		// })
		azureFilePath := fmt.Sprintf("//%s.file.core.windows.net/%s", storageID, storageSecret)
		mounter.Mount(azureFilePath, tempFolder, "cifs", options)

		fsync := fsync.NewFSync()
		err = fsync.CopyFolder(src, tempFolder)

		mounter.Unmount(tempFolder)
		// } else {

		// }

	} else {
		v.logger.Info("########(andliu) create temp folder failed.", lager.Data{"err": err.Error()})
	}
	// tmpl := template.Must(template.New("").Parse(stagerScript))
	// if err := tmpl.Execute(scriptBuf, struct {
	// 	RSync         bool
	// 	BuildpackMD5s []string
	// }{
	// mounter.Mount()

	return nil
}
