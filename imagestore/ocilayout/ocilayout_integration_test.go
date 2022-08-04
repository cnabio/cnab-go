package ocilayout

import (
	"fmt"
	"net/http"
	"os/exec"
	"testing"

	"github.com/pivotal/image-relocation/pkg/image"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/cnabio/cnab-go/imagestore"
	"github.com/cnabio/cnab-go/imagestore/tests"
)

func TestOCILayout_PushToInsecureRegistry(t *testing.T) {
	// Start an insecure registry and get the port that it's running on
	regPort := tests.StartTestRegistry(t)
	registry := fmt.Sprintf("localhost:%s", regPort)

	// Using hello-world
	srcDigest, err := image.NewDigest("sha256:f54a58bc1aac5ea1a25d796ae155dc228b3f0e11d046ae276b39c4bf2f13d8c4")
	require.NoError(t, err, "image.NewDigest failed")
	sourceImg, err := image.NewName(fmt.Sprintf("docker.io/library/hello-world@%s", srcDigest.String()))
	require.NoError(t, err, "image.NewName failed")
	destImg, err := image.NewName(fmt.Sprintf("%s/hello-world:latest", registry))
	require.NoError(t, err, "image.NewName failed")

	t.Run("failed push", func(t *testing.T) {
		// Try to push without passing in a http client with skipTLS set to true
		store, err := Create(imagestore.WithArchiveDir(t.TempDir()))
		require.NoError(t, err, "ocilayout.Create failed")

		// Add the source image to our registry
		dig, err := store.Add(sourceImg.String())
		require.NoError(t, err, "Add failed")
		assert.Equal(t, srcDigest.String(), dig, "incorrect content digest returned by Add")

		err = store.Push(srcDigest, sourceImg, destImg)
		require.Errorf(t, err, "expected push to fail because skipTLS was not specified")
		assert.Contains(t, err.Error(), "Client sent an HTTP request to an HTTPS server", "expected push to fail because TLS wasn't configured properly")
	})

	t.Run("successful push", func(t *testing.T) {
		// Create a http transport that allows connecting to our insecure registry
		skipTLS := http.DefaultTransport.(*http.Transport).Clone()
		skipTLS.TLSClientConfig.InsecureSkipVerify = true

		store, err := Create(imagestore.WithTransport(skipTLS), imagestore.WithArchiveDir(t.TempDir()))
		require.NoError(t, err, "ocilayout.Create failed")

		// Add the source image to our registry
		dig, err := store.Add(sourceImg.String())
		require.NoError(t, err, "Add failed")
		assert.Equal(t, srcDigest.String(), dig, "incorrect content digest returned by Add")

		// Push a test container (hello-world) into our registry
		err = store.Push(srcDigest, sourceImg, destImg)
		require.NoError(t, err, "Push failed")

		// Validate the image was copied to the new location
		err = exec.Command("docker", "pull", destImg.String()).Run()
		require.NoError(t, err, "the image was not present in the destination registry")
	})

	t.Run("push from archive", func(t *testing.T) {
		// Create a http transport that allows connecting to our insecure registry
		skipTLS := http.DefaultTransport.(*http.Transport).Clone()
		skipTLS.TLSClientConfig.InsecureSkipVerify = true

		parms := imagestore.Create(
			imagestore.WithArchiveDir("testdata"),
			imagestore.WithTransport(skipTLS))
		layout, err := LocateOciLayout(parms)
		require.NoError(t, err, "LocateOciLayout failed")

		//
		// Add the image from the archive into the index
		//
		img, err := image.NewName("docker.io/library/hello-world@sha256:f54a58bc1aac5ea1a25d796ae155dc228b3f0e11d046ae276b39c4bf2f13d8c4")
		require.NoError(t, err, "NewName failed")

		dig, err := layout.Add(img.String())
		require.NoError(t, err, "Add failed")
		assert.Equal(t, "sha256:f54a58bc1aac5ea1a25d796ae155dc228b3f0e11d046ae276b39c4bf2f13d8c4", dig, "incorrect content digest returned by Add")

		imgDig, err := image.NewDigest(dig)
		require.NoError(t, err, "NewDigest failed")

		// Push the image
		err = layout.Push(imgDig, img, destImg)
		require.NoError(t, err, "Push failed")

		// Validate the image was copied to the new location
		err = exec.Command("docker", "pull", destImg.String()).Run()
		require.NoError(t, err, "the image was not present in the destination registry")
	})
}
