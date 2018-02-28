package steps

import (
	"fmt"
	"io"
	"net/url"
	"time"

	"code.cloudfoundry.org/bbs/models"
	"code.cloudfoundry.org/bytefmt"
	"code.cloudfoundry.org/cacheddownloader"
	"code.cloudfoundry.org/executor/depot/log_streamer"
	"code.cloudfoundry.org/executor/depot/vci/vstore"
	"code.cloudfoundry.org/executor/model"
	"code.cloudfoundry.org/garden"
	"code.cloudfoundry.org/lager"
	"github.com/Azure/go-autorest/autorest/azure"
	"github.com/virtualcloudfoundry/goaci"
	"github.com/virtualcloudfoundry/goaci/aci"
)

type downloadStep struct {
	container        garden.Container
	model            models.DownloadAction
	cachedDownloader cacheddownloader.CachedDownloader
	streamer         log_streamer.LogStreamer
	rateLimiter      chan struct{}

	logger lager.Logger

	*canceller
}

func NewDownload(
	container garden.Container,
	model models.DownloadAction,
	cachedDownloader cacheddownloader.CachedDownloader,
	rateLimiter chan struct{},
	streamer log_streamer.LogStreamer,
	logger lager.Logger,
) *downloadStep {
	logger = logger.Session("download-step", lager.Data{
		"to":       model.To,
		"cacheKey": model.CacheKey,
		"user":     model.User,
	})

	return &downloadStep{
		container:        container,
		model:            model,
		cachedDownloader: cachedDownloader,
		streamer:         streamer,
		rateLimiter:      rateLimiter,
		logger:           logger,

		canceller: newCanceller(),
	}
}

func (step *downloadStep) Perform() error {
	step.logger.Info("acquiring-limiter")
	select {
	case step.rateLimiter <- struct{}{}:
	case <-step.Cancelled():
		return ErrCancelled
	}
	defer func() {
		<-step.rateLimiter
	}()
	step.logger.Info("acquired-limiter")

	err := step.perform()
	if err != nil {
		select {
		case <-step.Cancelled():
			return ErrCancelled
		default:
			return err
		}
	}

	return nil
}

func (step *downloadStep) perform() error {
	step.emit("Downloading %s...\n", step.model.Artifact)

	downloadedFile, downloadedSize, err := step.fetch()
	if err != nil {
		var errString string
		if step.model.Artifact != "" {
			errString = fmt.Sprintf("Downloading %s failed", step.model.Artifact)
		} else {
			errString = "Downloading failed"
		}

		step.emitError(fmt.Sprintf("%s\n", errString))
		return NewEmittableError(err, errString)
	}

	err = step.vStreamIn(step.model.To, downloadedFile)

	err = step.streamIn(step.model.To, downloadedFile)
	if err != nil {
		var errString string
		if step.model.Artifact != "" {
			errString = fmt.Sprintf("Copying %s into the container failed: %v", step.model.Artifact, err)
		} else {
			errString = fmt.Sprintf("Copying into the container failed: %v", err)
		}
		step.emitError(fmt.Sprintf("%s\n", errString))
		return NewEmittableError(err, errString)
	}

	if downloadedSize != 0 {
		step.emit("Downloaded %s (%s)\n", step.model.Artifact, bytefmt.ByteSize(uint64(downloadedSize)))
	} else {
		step.emit("Downloaded %s\n", step.model.Artifact)
	}

	return nil
}

func (step *downloadStep) fetch() (io.ReadCloser, int64, error) {
	step.logger.Info("fetch-starting")
	url, err := url.ParseRequestURI(step.model.From)
	if err != nil {
		step.logger.Error("parse-request-uri-error", err)
		return nil, 0, err
	}

	tarStream, downloadedSize, err := step.cachedDownloader.Fetch(
		step.logger.Session("downloader"),
		url,
		step.model.CacheKey,
		cacheddownloader.ChecksumInfoType{
			Algorithm: step.model.GetChecksumAlgorithm(),
			Value:     step.model.GetChecksumValue(),
		},
		step.Cancelled(),
	)
	if err != nil {
		step.logger.Error("fetch-failed", err)
		return nil, 0, err
	}
	step.logger.Info("########(andliu) fetch result.", lager.Data{"model": step.model})
	step.logger.Info("fetch-complete", lager.Data{"size": downloadedSize})
	return tarStream, downloadedSize, nil
}

func (step *downloadStep) vStreamIn(destination string, reader io.ReadCloser) error {
	// extract the tar to the target model.To
	// TODO create one share folder for /tmp
	// 1. get the container configs.
	var finaldestination string
	if destination == "." {
		// workaround, we guess . is the /home/vcap.
		// will extract the droplet file to this folder.
		finaldestination = "/home/vcap"
	} else {
		finaldestination = destination
	}
	handle := step.container.Handle()
	step.logger.Info("##########(andliu) perform vStreamIn step.", lager.Data{
		"handle":      handle,
		"destination": finaldestination})
	var azAuth *goaci.Authentication

	executorEnv := model.GetExecutorEnvInstance()
	config := executorEnv.Config.ContainerProviderConfig
	azAuth = goaci.NewAuthentication(azure.PublicCloud.Name, config.ContainerId, config.ContainerSecret, config.SubscriptionId, config.OptionalParam1)

	aciClient, err := aci.NewClient(azAuth)
	if err == nil {
		containerGroupGot, err, code := aciClient.GetContainerGroup(executorEnv.ResourceGroup, handle)
		if err == nil {
			step.logger.Info("##########(andliu) download step in get container group.",
				lager.Data{"code": code, "containerGroupGot": *containerGroupGot})
			// create a folder
			vstore := vstore.NewVStore()
			// handle = "downloadstep" // TODO remove this, hard code for consistent folder.
			shareName, err := vstore.CreateFolder("downloadstep", finaldestination)

			executorEnv := model.GetExecutorEnvInstance()
			if err == nil {
				step.logger.Info("#########(andliu) shareName.", lager.Data{"shareName": shareName})
				azureFile := &aci.AzureFileVolume{
					ReadOnly:           false,
					ShareName:          shareName,
					StorageAccountName: executorEnv.Config.ContainerProviderConfig.StorageId,
					StorageAccountKey:  executorEnv.Config.ContainerProviderConfig.StorageSecret,
				}
				newVolume := aci.Volume{Name: shareName, AzureFile: azureFile}
				containerGroupGot.ContainerGroupProperties.Volumes = append(
					containerGroupGot.ContainerGroupProperties.Volumes, newVolume)
				volumeMount := aci.VolumeMount{
					Name:      shareName,
					MountPath: finaldestination,
					ReadOnly:  false,
				}

				// save back the storage account key
				for idx, _ := range containerGroupGot.ContainerGroupProperties.Volumes {
					containerGroupGot.ContainerGroupProperties.Volumes[idx].AzureFile.StorageAccountKey =
						executorEnv.Config.ContainerProviderConfig.StorageSecret
				}
				for idx, _ := range containerGroupGot.ContainerGroupProperties.Containers {
					containerGroupGot.ContainerGroupProperties.Containers[idx].VolumeMounts = append(
						containerGroupGot.ContainerGroupProperties.Containers[idx].VolumeMounts, volumeMount)
				}
				step.logger.Info("#########(andliu) update container group:", lager.Data{"containerGroupGot": *containerGroupGot})
				containerGroupUpdated, err := aciClient.UpdateContainerGroup(executorEnv.ResourceGroup, handle, *containerGroupGot)
				retry := 0
				for err != nil && retry < 10 {
					step.logger.Info("#########(andliu) update container group failed.", lager.Data{"err": err.Error()})
					time.Sleep(time.Second * 20)
					containerGroupUpdated, err = aciClient.UpdateContainerGroup(executorEnv.ResourceGroup, handle, *containerGroupGot)
					retry++
				}
				if err == nil {
					step.logger.Info("##########(andliu) update container group succeeded.", lager.Data{"containerGroupUpdated": containerGroupUpdated})
				} else {
					step.logger.Info("#########(andliu) update container group failed.", lager.Data{"err": err.Error()})
				}
			} else {
				step.logger.Info("#########(andliu) shareName failed.", lager.Data{"err": err.Error()})
			}
		} else {
			step.logger.Info("##########(andliu) GetContainerGroup.", lager.Data{"err": err.Error()})
		}
	} else {
		step.logger.Info("##########(andliu) new client.", lager.Data{"err": err.Error()})
	}
	return nil
}

func (step *downloadStep) streamIn(destination string, reader io.ReadCloser) error {
	step.logger.Info("stream-in-starting")
	step.logger.Info("##########(andliu) streamIn:", lager.Data{"destination": destination})
	// StreamIn will close the reader
	err := step.container.StreamIn(garden.StreamInSpec{Path: destination, TarStream: reader, User: step.model.User})
	if err != nil {
		step.logger.Error("stream-in-failed", err, lager.Data{
			"destination": destination,
		})
		return err
	}

	step.logger.Info("stream-in-complete")
	return nil
}

func (step *downloadStep) emit(format string, a ...interface{}) {
	if step.model.Artifact != "" {
		fmt.Fprintf(step.streamer.Stdout(), format, a...)
	}
}

func (step *downloadStep) emitError(format string, a ...interface{}) {
	err_bytes := []byte(fmt.Sprintf(format, a...))
	if len(err_bytes) > 1024 {
		truncation_length := 1024 - len([]byte(" (error truncated)"))
		err_bytes = append(err_bytes[:truncation_length], []byte(" (error truncated)")...)
	}

	fmt.Fprintf(step.streamer.Stderr(), string(err_bytes))
}
