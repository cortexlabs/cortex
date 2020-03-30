package cmd

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"os"
	"os/signal"
	"path"
	"path/filepath"
	"syscall"
	"time"

	"github.com/cortexlabs/cortex/pkg/lib/errors"
	"github.com/cortexlabs/cortex/pkg/lib/exit"
	"github.com/cortexlabs/cortex/pkg/lib/files"
	"github.com/cortexlabs/cortex/pkg/lib/prompt"
	s "github.com/cortexlabs/cortex/pkg/lib/strings"
	"github.com/cortexlabs/cortex/pkg/lib/zip"
	dockertypes "github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/mount"
	"github.com/docker/go-connections/nat"
	"github.com/spf13/cobra"
)

var _localWorkSpace string

func init() {
	localCmd.PersistentFlags()
	addEnvFlag(localCmd)
	_localWorkSpace = filepath.Join(_localDir, "local_workspace")

}

func deploymentBytes(configPath string, force bool) map[string][]byte {
	configBytes, err := files.ReadFileBytes(configPath)
	if err != nil {
		exit.Error(err)
	}

	uploadBytes := map[string][]byte{
		"config": configBytes,
	}

	projectRoot := filepath.Dir(files.UserRelToAbsPath(configPath))

	ignoreFns := []files.IgnoreFn{
		files.IgnoreSpecificFiles(files.UserRelToAbsPath(configPath)),
		files.IgnoreCortexDebug,
		files.IgnoreHiddenFiles,
		files.IgnoreHiddenFolders,
		files.IgnorePythonGeneratedFiles,
	}

	cortexIgnorePath := path.Join(projectRoot, ".cortexignore")
	if files.IsFile(cortexIgnorePath) {
		cortexIgnore, err := files.GitIgnoreFn(cortexIgnorePath)
		if err != nil {
			exit.Error(err)
		}
		ignoreFns = append(ignoreFns, cortexIgnore)
	}

	if !_flagDeployYes {
		ignoreFns = append(ignoreFns, files.PromptForFilesAboveSize(_warningFileBytes, "do you want to upload %s (%s)?"))
	}

	projectPaths, err := files.ListDirRecursive(projectRoot, false, ignoreFns...)
	if err != nil {
		exit.Error(err)
	}

	canSkipPromptMsg := "you can skip this prompt next time with `cortex deploy --yes`\n"
	rootDirMsg := "this directory"
	if s.EnsureSuffix(projectRoot, "/") != _cwd {
		rootDirMsg = fmt.Sprintf("./%s", files.DirPathRelativeToCWD(projectRoot))
	}

	didPromptFileCount := false
	if !_flagDeployYes && len(projectPaths) >= _warningFileCount {
		msg := fmt.Sprintf("cortex will zip %d files in %s and upload them to the cluster; we recommend that you upload large files/directories (e.g. models) to s3 and download them in your api's __init__ function, and avoid sending unnecessary files by removing them from this directory or referencing them in a .cortexignore file. Would you like to continue?", len(projectPaths), rootDirMsg)
		prompt.YesOrExit(msg, canSkipPromptMsg, "")
		didPromptFileCount = true
	}

	projectZipBytes, err := zip.ToMem(&zip.Input{
		FileLists: []zip.FileListInput{
			{
				Sources:      projectPaths,
				RemovePrefix: projectRoot,
			},
		},
	})
	if err != nil {
		exit.Error(errors.Wrap(err, "failed to zip project folder"))
	}

	if !_flagDeployYes && !didPromptFileCount && len(projectZipBytes) >= _warningProjectBytes {
		msg := fmt.Sprintf("cortex will zip %d files in %s (%s) and upload them to the cluster, though we recommend you upload large files (e.g. models) to s3 and download them in your api's __init__ function. Would you like to continue?", len(projectPaths), rootDirMsg, s.IntToBase2Byte(len(projectZipBytes)))
		prompt.YesOrExit(msg, canSkipPromptMsg, "")
	}

	uploadBytes["project.zip"] = projectZipBytes
	return uploadBytes
}

var localCmd = &cobra.Command{
	Use:   "local",
	Short: "local an application",
	Long:  "local an application.",
	Args:  cobra.ExactArgs(0),
	Run: func(cmd *cobra.Command, args []string) {
		docker, err := getDockerClient()
		if err != nil {
			panic(err)
		}

		// _localWorkSpace = filepath.Join(_localDir, "local_workspace")

		// configPath := getConfigPath(args)
		// deploymentMap := deploymentBytes(configPath, false)
		// projectFileMap, err := zip.UnzipMemToMem(deploymentMap["project.zip"])
		// if err != nil {
		// 	exit.Error(err)
		// }

		// apiConfigs, err := spec.ExtractAPIConfigs(deploymentMap["config"], projectFileMap, configPath)
		// if err != nil {
		// 	exit.Error(err)
		// }

		// err = spec.ValidateLocalAPIs(apiConfigs, projectFileMap)
		// if err != nil {
		// 	exit.Error(err)
		// }

		// projectID := hash.Bytes(deploymentMap["project.zip"])
		// deploymentID := k8s.RandomName()
		// api := spec.GetAPISpec(&apiConfigs[0], projectID, deploymentID)

		hostConfig := &container.HostConfig{
			PortBindings: nat.PortMap{
				"8888/tcp": []nat.PortBinding{
					{
						HostPort: "8888",
					},
				},
			},
			Mounts: []mount.Mount{
				{
					Type:   mount.TypeBind,
					Source: "/home/ubuntu/src/github.com/cortexlabs/cortex/examples/pytorch/iris-classifier",
					Target: "/mnt/project",
				},
			},
		}

		containerConfig := &container.Config{
			Image:        "cortexlabs/python-serve:latest",
			Tty:          true,
			AttachStdout: true,
			AttachStderr: true,
			Env: []string{
				"CORTEX_VERSION=master",
				"CORTEX_SERVING_PORT=8888",
				"CORTEX_CACHE_DIR=" + "/mnt/cache",
				"CORTEX_API_SPEC=" + "app.yaml",
				"CORTEX_PROJECT_DIR=" + "/mnt/project",
				"CORTEX_WORKERS_PER_REPLICA=1",
				"CORTEX_MAX_WORKER_CONCURRENCY=10",
				"CORTEX_SO_MAX_CONN=10",
				"CORTEX_THREADS_PER_WORKER=1",
				"AWS_ACCESS_KEY_ID=" + os.Getenv("AWS_ACCESS_KEY_ID"),
				"AWS_SECRET_ACCESS_KEY=" + os.Getenv("AWS_SECRET_ACCESS_KEY"),
			},
			ExposedPorts: nat.PortSet{
				"8888/tcp": struct{}{},
			},
		}

		containerInfo, err := docker.ContainerCreate(context.Background(), containerConfig, hostConfig, nil, "")
		if err != nil {
			panic(err)
		}

		err = docker.ContainerStart(context.Background(), containerInfo.ID, dockertypes.ContainerStartOptions{})
		if err != nil {
			panic("ContainerStart")
		}

		removeContainer := func() {
			docker.ContainerRemove(context.Background(), containerInfo.ID, dockertypes.ContainerRemoveOptions{
				RemoveVolumes: true,
				Force:         true,
			})
		}

		defer removeContainer()

		// Make sure to remove container immediately on ctrl+c
		c := make(chan os.Signal, 1)
		signal.Notify(c, os.Interrupt, syscall.SIGTERM)
		caughtCtrlC := false
		go func() {
			<-c
			caughtCtrlC = true
			removeContainer()
			exit.Error(ErrorDockerCtrlC())
		}()

		err = docker.ContainerStart(context.Background(), containerInfo.ID, dockertypes.ContainerStartOptions{})
		if err != nil {
			panic("ContainerInspect")
		}

		logsOutput, err := docker.ContainerAttach(context.Background(), containerInfo.ID, dockertypes.ContainerAttachOptions{
			Stream: true,
			Stdout: true,
			Stderr: true,
		})
		if err != nil {
			panic("ContainerAttach")
		}
		defer logsOutput.Close()

		var outputBuffer bytes.Buffer
		tee := io.TeeReader(logsOutput.Reader, &outputBuffer)

		_, err = io.Copy(os.Stdout, tee)
		if err != nil {
			panic("Copy")
		}

		// Let the ctrl+c handler run its course
		if caughtCtrlC {
			time.Sleep(5 * time.Second)
		}

		info, err := docker.ContainerInspect(context.Background(), containerInfo.ID)
		if err != nil {
			panic("ContainerInspect")
		}

		if info.State.Running {
			panic("info.State.Running")
		}

	},
}
