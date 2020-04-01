package cmd

import (
	"context"
	"fmt"
	"path"
	"path/filepath"

	"github.com/cortexlabs/cortex/cli/local"
	"github.com/cortexlabs/cortex/pkg/lib/debug"
	"github.com/cortexlabs/cortex/pkg/lib/errors"
	"github.com/cortexlabs/cortex/pkg/lib/exit"
	"github.com/cortexlabs/cortex/pkg/lib/files"
	"github.com/cortexlabs/cortex/pkg/lib/hash"
	"github.com/cortexlabs/cortex/pkg/lib/prompt"
	s "github.com/cortexlabs/cortex/pkg/lib/strings"
	"github.com/cortexlabs/cortex/pkg/lib/table"
	"github.com/cortexlabs/cortex/pkg/lib/zip"
	"github.com/cortexlabs/cortex/pkg/operator/schema"
	"github.com/cortexlabs/cortex/pkg/types/spec"
	"github.com/docker/docker/api/types"
	dockertypes "github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/filters"
	"github.com/spf13/cobra"
)

func init() {
	localCmd.PersistentFlags()
	addEnvFlag(localCmd)
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
		configPath := getConfigPath(args)
		deploymentMap := deploymentBytes(configPath, false)
		projectFileMap, err := zip.UnzipMemToMem(deploymentMap["project.zip"])
		if err != nil {
			exit.Error(err)
		}

		apiConfigs, err := spec.ExtractAPIConfigs(deploymentMap["config"], projectFileMap, configPath)
		if err != nil {
			exit.Error(err)
		}

		err = local.ValidateLocalAPIs(apiConfigs, projectFileMap)
		if err != nil {
			exit.Error(err)
		}

		projectID := hash.Bytes(deploymentMap["project.zip"])

		results := make([]schema.DeployResult, len(apiConfigs))
		for i, apiConfig := range apiConfigs {
			api, msg, err := local.UpdateAPI(&apiConfig, projectID)
			results[i].Message = msg
			if err != nil {
				results[i].Error = errors.Message(err)
			} else {
				results[i].API = *api
			}
		}
		debug.Pp(results)
	},
}

var localGet = &cobra.Command{
	Use:   "local-get",
	Short: "local an application",
	Long:  "local an application.",
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		containers := GetContainerByAPI(args[0])
		debug.Pp(containers)
		rows := [][]interface{}{}

		for _, container := range containers {
			rows = append(rows, []interface{}{
				container.Labels["apiName"], container.State,
			})
		}

		t := table.Table{
			Headers: []table.Header{
				{
					Title: "api name",
				}, {
					Title: "status",
				},
			},
			Rows: rows,
		}
		fmt.Println(t.MustFormat())
	},
}

func GetContainerByAPI(apiName string) []dockertypes.Container {
	docker, err := getDockerClient()
	if err != nil {
		panic(err)
	}

	dargs := filters.NewArgs()
	dargs.Add("label", "cortex=true")
	dargs.Add("label", "apiName="+apiName)

	containers, err := docker.ContainerList(context.Background(), types.ContainerListOptions{
		All:     true,
		Filters: dargs,
	})
	if err != nil {
		exit.Error(err)
	}

	return containers
}

var localDelete = &cobra.Command{
	Use:   "local-delete",
	Short: "local an application",
	Long:  "local an application.",
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		apiName := args[0]
		err := local.DeleteContainers(apiName)
		if err != nil {
			exit.Error(err)
		}
	},
}

var localLogs = &cobra.Command{
	Use:   "local-logs",
	Short: "local an application",
	Long:  "local an application.",
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		path, err := files.GetAbsPath(args[0])
		if err != nil {
			panic(err)
		}
		fmt.Println(path)
		// containers := GetContainerByAPI(args[0])
		// containerIDs := []string{}
		// for _, container := range containers {
		// 	containerIDs = append(containerIDs, container.ID)
		// }

		// streamDockerLogs(containerIDs[0], containerIDs[1:]...)
	},
}
