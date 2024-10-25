package docker

import (
	"archive/tar"
	"context"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	unix_path "path"
	"strconv"
	"strings"

	"github.com/docker/cli/cli/command"
	"github.com/docker/distribution/reference"
	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/container"
	imagetypes "github.com/docker/docker/api/types/image"
	registrytypes "github.com/docker/docker/api/types/registry"
	"github.com/docker/docker/api/types/strslice"
	"github.com/docker/docker/client"
	"github.com/docker/docker/pkg/jsonmessage"
	"github.com/docker/docker/pkg/stdcopy"
	"github.com/docker/docker/registry"
	"github.com/mitchellh/copystructure"
	"github.com/pkg/errors"

	"github.com/cnabio/cnab-go/bundle"
	"github.com/cnabio/cnab-go/driver"
)

const (
	// SettingNetwork is the environment variable for the driver that specifies
	// the docker network to which the invocation image should be attached.
	SettingNetwork = "DOCKER_NETWORK"
)

// Driver is capable of running Docker invocation images using Docker itself.
type Driver struct {
	config map[string]string
	// If true, this will not actually run Docker
	Simulate                   bool
	dockerCli                  command.Cli
	dockerConfigurationOptions []ConfigurationOption
	containerOut               io.Writer
	containerErr               io.Writer
	containerHostCfg           container.HostConfig
	containerCfg               container.Config
}

// Run executes the Docker driver
func (d *Driver) Run(op *driver.Operation) (driver.OperationResult, error) {
	return d.exec(op)
}

// Handles indicates that the Docker driver supports "docker" and "oci"
func (d *Driver) Handles(dt string) bool {
	return dt == driver.ImageTypeDocker || dt == driver.ImageTypeOCI
}

// AddConfigurationOptions adds configuration callbacks to the driver
func (d *Driver) AddConfigurationOptions(opts ...ConfigurationOption) {
	d.dockerConfigurationOptions = append(d.dockerConfigurationOptions, opts...)
}

// GetContainerConfig returns a copy of the container configuration
// used by the driver during container exec
func (d *Driver) GetContainerConfig() (container.Config, error) {
	cpy, err := copystructure.Copy(d.containerCfg)
	if err != nil {
		return container.Config{}, err
	}

	cfg, ok := cpy.(container.Config)
	if !ok {
		return container.Config{}, errors.New("unable to process container config")
	}

	return cfg, nil
}

// GetContainerHostConfig returns a copy of the container host configuration
// used by the driver during container exec
func (d *Driver) GetContainerHostConfig() (container.HostConfig, error) {
	cpy, err := copystructure.Copy(d.containerHostCfg)
	if err != nil {
		return container.HostConfig{}, err
	}

	cfg, ok := cpy.(container.HostConfig)
	if !ok {
		return container.HostConfig{}, errors.New("unable to process container host config")
	}

	return cfg, nil
}

// Config returns the Docker driver configuration options
func (d *Driver) Config() map[string]string {
	return map[string]string{
		"PULL_ALWAYS":         "Always pull image, even if locally available (0|1)",
		"DOCKER_DRIVER_QUIET": "Make the Docker driver quiet (only print container stdout/stderr)",
		"CLEANUP_CONTAINERS":  "If true, the docker container will be destroyed when it finishes running. If false, it will not be destroyed. The supported values are true and false. Defaults to true.",
		SettingNetwork:        "Attach the invocation image to the specified docker network",
	}
}

// SetConfig sets Docker driver configuration
func (d *Driver) SetConfig(settings map[string]string) error {
	// Set default and provide feedback on acceptable input values.
	value, ok := settings["CLEANUP_CONTAINERS"]
	if !ok {
		settings["CLEANUP_CONTAINERS"] = "true"
	} else if value != "true" && value != "false" {
		return fmt.Errorf("environment variable CLEANUP_CONTAINERS has unexpected value %q. Supported values are 'true', 'false', or unset", value)
	}

	d.config = settings
	return nil
}

// SetDockerCli makes the driver use an already initialized cli
func (d *Driver) SetDockerCli(dockerCli command.Cli) {
	d.dockerCli = dockerCli
}

// SetContainerOut sets the container output stream
func (d *Driver) SetContainerOut(w io.Writer) {
	d.containerOut = w
}

// SetContainerErr sets the container error stream
func (d *Driver) SetContainerErr(w io.Writer) {
	d.containerErr = w
}

func pullImage(ctx context.Context, cli command.Cli, image string) error {
	ref, err := reference.ParseNormalizedNamed(image)
	if err != nil {
		return err
	}

	// Resolve the Repository name from fqn to RepositoryInfo
	repoInfo, err := registry.ParseRepositoryInfo(ref)
	if err != nil {
		return err
	}
	authConfig := command.ResolveAuthConfig(cli.ConfigFile(), repoInfo.Index)
	encodedAuth, err := registrytypes.EncodeAuthConfig(authConfig)
	if err != nil {
		return err
	}
	options := imagetypes.PullOptions{
		RegistryAuth: encodedAuth,
	}
	responseBody, err := cli.Client().ImagePull(ctx, image, options)
	if err != nil {
		return err
	}
	defer responseBody.Close()

	// passing isTerm = false here because of https://github.com/Nvveen/Gotty/pull/1
	return jsonmessage.DisplayJSONMessagesStream(responseBody, cli.Out(), cli.Out().FD(), false, nil)
}

func (d *Driver) initializeDockerCli() (command.Cli, error) {
	if d.dockerCli != nil {
		return d.dockerCli, nil
	}

	cli, err := GetDockerClient()
	if err != nil {
		return nil, err
	}

	if d.config["DOCKER_DRIVER_QUIET"] == "1" {
		cli.Apply(command.WithCombinedStreams(ioutil.Discard))
	}

	d.dockerCli = cli
	return cli, nil
}

func (d *Driver) exec(op *driver.Operation) (driver.OperationResult, error) {
	ctx := context.Background()

	cli, err := d.initializeDockerCli()
	if err != nil {
		return driver.OperationResult{}, err
	}

	if d.Simulate {
		return driver.OperationResult{}, nil
	}
	if d.config["PULL_ALWAYS"] == "1" {
		if err := pullImage(ctx, cli, op.Image.Image); err != nil {
			return driver.OperationResult{}, err
		}
	}

	ii, err := d.inspectImage(ctx, op.Image)
	if err != nil {
		return driver.OperationResult{}, err
	}

	err = d.validateImageDigest(op.Image, ii.RepoDigests)
	if err != nil {
		return driver.OperationResult{}, errors.Wrap(err, "image digest validation failed")
	}

	if err := d.setConfigurationOptions(op); err != nil {
		return driver.OperationResult{}, err
	}

	resp, err := cli.Client().ContainerCreate(ctx, &d.containerCfg, &d.containerHostCfg, nil, nil, "")
	if err != nil {
		return driver.OperationResult{}, fmt.Errorf("cannot create container: %v", err)
	}

	if d.config["CLEANUP_CONTAINERS"] == "true" {
		defer cli.Client().ContainerRemove(ctx, resp.ID, container.RemoveOptions{})
	}

	containerUID := getContainerUserID(ii.Config.User)
	tarContent, err := generateTar(op.Files, containerUID)
	if err != nil {
		return driver.OperationResult{}, fmt.Errorf("error staging files: %s", err)
	}
	options := types.CopyToContainerOptions{
		AllowOverwriteDirWithFile: false,
	}
	// This copies the tar to the root of the container. The tar has been assembled using the
	// path from the given file, starting at the /.
	err = cli.Client().CopyToContainer(ctx, resp.ID, "/", tarContent, options)
	if err != nil {
		return driver.OperationResult{}, fmt.Errorf("error copying to / in container: %s", err)
	}

	attach, err := cli.Client().ContainerAttach(ctx, resp.ID, container.AttachOptions{
		Stream: true,
		Stdout: true,
		Stderr: true,
		Logs:   true,
	})
	if err != nil {
		return driver.OperationResult{}, fmt.Errorf("unable to retrieve logs: %v", err)
	}
	var (
		stdout io.Writer = os.Stdout
		stderr io.Writer = os.Stderr
	)
	if d.containerOut != nil {
		stdout = d.containerOut
	} else if op.Out != nil {
		stdout = op.Out
	}
	if d.containerErr != nil {
		stderr = d.containerErr
	} else if op.Err != nil {
		stderr = op.Err
	}
	go func() {
		defer attach.Close()
		for {
			_, err = stdcopy.StdCopy(stdout, stderr, attach.Reader)
			if err != nil {
				break
			}
		}
	}()

	if err = cli.Client().ContainerStart(ctx, resp.ID, container.StartOptions{}); err != nil {
		return driver.OperationResult{}, fmt.Errorf("cannot start container: %v", err)
	}
	statusc, errc := cli.Client().ContainerWait(ctx, resp.ID, container.WaitConditionNotRunning)
	select {
	case err := <-errc:
		if err != nil {
			opResult, fetchErr := d.fetchOutputs(ctx, resp.ID, op)
			return opResult, containerError("error in container", err, fetchErr)
		}
	case s := <-statusc:
		if s.StatusCode == 0 {
			return d.fetchOutputs(ctx, resp.ID, op)
		}
		if s.Error != nil {
			opResult, fetchErr := d.fetchOutputs(ctx, resp.ID, op)
			return opResult, containerError(fmt.Sprintf("container exit code: %d, message", s.StatusCode), err, fetchErr)
		}
		opResult, fetchErr := d.fetchOutputs(ctx, resp.ID, op)
		return opResult, containerError(fmt.Sprintf("container exit code: %d, message", s.StatusCode), err, fetchErr)
	}
	opResult, fetchErr := d.fetchOutputs(ctx, resp.ID, op)
	if fetchErr != nil {
		return opResult, fmt.Errorf("fetching outputs failed: %s", fetchErr)
	}
	return opResult, err
}

// getContainerUserID determines the user id that the container will execute as
// based on the image's configured user. Defaults to 0 (root) if a user id is not set.
func getContainerUserID(user string) int {
	if user != "" {
		// Only look at the user, strip off a group if one was specified with USER uid:gid
		if uid, err := strconv.Atoi(strings.Split(user, ":")[0]); err == nil {
			return uid
		}
	}
	return 0
}

// ApplyConfigurationOptions applies the configuration options set on the driver by the user.
func (d *Driver) ApplyConfigurationOptions() error {
	for _, opt := range d.dockerConfigurationOptions {
		if err := opt(&d.containerCfg, &d.containerHostCfg); err != nil {
			return err
		}
	}
	return nil
}

// setConfigurationOptions initializes the container and host configuration options on the driver,
// combining the default configuration with any overrides set by the user.
func (d *Driver) setConfigurationOptions(op *driver.Operation) error {
	var env []string
	for k, v := range op.Environment {
		env = append(env, fmt.Sprintf("%s=%v", k, v))
	}

	d.containerCfg = container.Config{
		Image:        op.Image.Image,
		Env:          env,
		Entrypoint:   strslice.StrSlice{"/cnab/app/run"},
		AttachStderr: true,
		AttachStdout: true,
	}

	d.containerHostCfg = container.HostConfig{}

	if network, ok := d.config[SettingNetwork]; ok {
		d.containerHostCfg.NetworkMode = container.NetworkMode(network)
	}

	if err := d.ApplyConfigurationOptions(); err != nil {
		return err
	}

	return nil
}

func containerError(containerMessage string, containerErr, fetchErr error) error {
	if fetchErr != nil {
		return fmt.Errorf("%s: %v. fetching outputs failed: %s", containerMessage, containerErr, fetchErr)
	}
	return fmt.Errorf("%s: %v", containerMessage, containerErr)
}

// fetchOutputs takes a context and a container ID; it copies the /cnab/app/outputs directory from that container.
// The goal is to collect all the files in the directory (recursively) and put them in a flat map of path to contents.
// This map will be inside the OperationResult. When fetchOutputs returns an error, it may also return partial results.
func (d *Driver) fetchOutputs(ctx context.Context, container string, op *driver.Operation) (driver.OperationResult, error) {
	opResult := driver.OperationResult{
		Outputs: map[string]string{},
	}
	// The /cnab/app/outputs directory probably only exists if outputs are created. In the
	// case there are no outputs defined on the operation, there probably are none to copy
	// and we should return early.
	if len(op.Outputs) == 0 {
		return opResult, nil
	}
	ioReader, _, err := d.dockerCli.Client().CopyFromContainer(ctx, container, "/cnab/app/outputs")
	if err != nil {
		return opResult, fmt.Errorf("error copying outputs from container: %s", err)
	}
	tarReader := tar.NewReader(ioReader)
	header, err := tarReader.Next()
	// io.EOF pops us out of loop on successful run.
	for err == nil {
		// skip directories because we're gathering file contents
		if header.FileInfo().IsDir() {
			header, err = tarReader.Next()
			continue
		}

		var contents []byte
		// CopyFromContainer strips prefix above outputs directory.
		pathInContainer := unix_path.Join("/cnab", "app", header.Name)
		outputName, shouldCapture := op.Outputs[pathInContainer]
		if shouldCapture {
			contents, err = ioutil.ReadAll(tarReader)
			if err != nil {
				return opResult, fmt.Errorf("error while reading %q from outputs tar: %s", pathInContainer, err)
			}
			opResult.Outputs[outputName] = string(contents)
		}

		header, err = tarReader.Next()
	}

	if err != io.EOF {
		return opResult, err
	}

	return opResult, nil
}

// generateTar creates a tarfile containing the specified files, with the owner
// set to the uid that the container runs as so that it is guaranteed to have
// read access to the files we copy into the container.
func generateTar(files map[string]string, uid int) (io.Reader, error) {
	r, w := io.Pipe()
	tw := tar.NewWriter(w)
	for path := range files {
		if !unix_path.IsAbs(path) {
			return nil, fmt.Errorf("destination path %s should be an absolute unix path", path)
		}
	}
	go func() {
		for path, content := range files {
			// Write a header for the parent directories so that newly created intermediate directories are accessible by the user
			dir := path
			for dir != "/" {
				dir = unix_path.Dir(dir)
				dirHdr := &tar.Header{
					Typeflag: tar.TypeDir,
					Name:     dir,
					Mode:     0700,
					Uid:      uid,
					Size:     0,
				}
				tw.WriteHeader(dirHdr)
			}

			// Grant access to just the owner (container user), so that files can be read by the container
			fildHdr := &tar.Header{
				Typeflag: tar.TypeReg,
				Name:     path,
				Mode:     0600,
				Size:     int64(len(content)),
				Uid:      uid,
			}
			tw.WriteHeader(fildHdr)
			tw.Write([]byte(content))
		}
		w.Close()
	}()
	return r, nil
}

// ConfigurationOption is an option used to customize docker driver container and host config
type ConfigurationOption func(*container.Config, *container.HostConfig) error

// inspectImage inspects the operation image and returns an object of types.ImageInspect,
// pulling the image if not found locally
func (d *Driver) inspectImage(ctx context.Context, image bundle.InvocationImage) (types.ImageInspect, error) {
	ii, _, err := d.dockerCli.Client().ImageInspectWithRaw(ctx, image.Image)
	switch {
	case client.IsErrNotFound(err):
		fmt.Fprintf(d.dockerCli.Err(), "Unable to find image '%s' locally\n", image.Image)
		if err := pullImage(ctx, d.dockerCli, image.Image); err != nil {
			return ii, err
		}
		if ii, _, err = d.dockerCli.Client().ImageInspectWithRaw(ctx, image.Image); err != nil {
			return ii, errors.Wrapf(err, "cannot inspect image %s", image.Image)
		}
	case err != nil:
		return ii, errors.Wrapf(err, "cannot inspect image %s", image.Image)
	}

	return ii, nil
}

// validateImageDigest validates the operation image digest, if exists, against
// the supplied repoDigests
func (d *Driver) validateImageDigest(image bundle.InvocationImage, repoDigests []string) error {
	if image.Digest == "" {
		return nil
	}

	if len(repoDigests) == 0 {
		return fmt.Errorf("image %s has no repo digests", image.Image)
	}

	for _, repoDigest := range repoDigests {
		// RepoDigests are of the form 'imageName@sha256:<sha256>' or imageName:<tag>
		// We only care about the ones in digest form
		ref, err := reference.ParseNormalizedNamed(repoDigest)
		if err != nil {
			return fmt.Errorf("unable to parse repo digest %s", repoDigest)
		}

		digestRef, ok := ref.(reference.Digested)
		if !ok {
			continue
		}

		digest := digestRef.Digest().String()

		// image.Digest is the digest of the original invocation image defined in the bundle.
		// It persists even when the bundle's invocation image has been relocated.
		if digest == image.Digest {
			return nil
		}
	}

	return fmt.Errorf("content digest mismatch: invocation image %s was defined in the bundle with the digest %s but no matching repoDigest was found upon inspecting the image", image.Image, image.Digest)
}
