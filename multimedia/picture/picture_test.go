// SPDX-License-Identifier: ice License 1.0

package picture

import (
	"context"
	"embed"
	"fmt"
	"mime/multipart"
	"os"
	"sync"
	"testing"
	stdlibtime "time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/ice-blockchain/wintr/multimedia/picture/fixture"
)

const (
	testReadApplicationYAMLKey  = "read"
	testWriteApplicationYAMLKey = "write"
	ignoreFilenameRegex         = "^[0-9a-fA-F]{8}\\b-[0-9a-fA-F]{4}\\b-[0-9a-fA-F]{4}\\b-[0-9a-fA-F]{4}\\b-[0-9a-fA-F]{12}[.](jpg|png)$"
)

// .
var (
	//nolint:gochecknoglobals // It's a stateless singleton for tests.
	client Client
	//nolint:gochecknoglobals // It's a stateless singleton for tests.
	noDeleteClient Client
	//go:embed .testdata/*.jpg .testdata/*.png
	pictures embed.FS
)

func TestMain(m *testing.M) {
	client = New(testWriteApplicationYAMLKey)
	noDeleteClient = New(testWriteApplicationYAMLKey, ignoreFilenameRegex)
	os.Exit(m.Run())
}

func TestClientUpload(t *testing.T) {
	t.Parallel()
	ctx, cancel := context.WithTimeout(context.Background(), 30*stdlibtime.Second)
	defer cancel()

	pic1 := getPic1(t)
	require.NoError(t, client.UploadPicture(ctx, pic1, ""))
	fixture.AssertPictureUploaded(ctx, t, testWriteApplicationYAMLKey, pic1.Filename)

	pic2 := getPic2(t)
	require.NoError(t, client.UploadPicture(ctx, pic2, client.DownloadURL(pic1.Filename)))
	fixture.AssertPictureUploaded(ctx, t, testWriteApplicationYAMLKey, pic2.Filename)
	fixture.AssertPictureDeleted(ctx, t, testWriteApplicationYAMLKey, pic1.Filename)

	pic1 = getPic1(t)
	require.NoError(t, client.UploadPicture(ctx, pic1, pic2.Filename))
	fixture.AssertPictureUploaded(ctx, t, testWriteApplicationYAMLKey, pic1.Filename)
	fixture.AssertPictureDeleted(ctx, t, testWriteApplicationYAMLKey, pic2.Filename)

	require.NoError(t, client.UploadPicture(ctx, nil, pic1.Filename))
	fixture.AssertPictureDeleted(ctx, t, testWriteApplicationYAMLKey, pic1.Filename)
}

func TestClientUploadNoDelete(t *testing.T) { //nolint:funlen // .
	t.Parallel()
	ctx, cancel := context.WithTimeout(context.Background(), 30*stdlibtime.Second)
	defer cancel()

	var cleanupFilenames [3]string
	pic1 := getPic1(t)
	cleanupFilenames[0] = pic1.Filename
	require.NoError(t, noDeleteClient.UploadPicture(ctx, pic1, ""))
	fixture.AssertPictureUploaded(ctx, t, testWriteApplicationYAMLKey, pic1.Filename)

	pic2 := getPic2(t)
	cleanupFilenames[1] = pic2.Filename
	require.NoError(t, noDeleteClient.UploadPicture(ctx, pic2, noDeleteClient.DownloadURL(pic1.Filename)))
	fixture.AssertPictureUploaded(ctx, t, testWriteApplicationYAMLKey, pic2.Filename)
	fixture.AssertPictureUploaded(ctx, t, testWriteApplicationYAMLKey, pic1.Filename) // Not deleted.

	pic1 = getPic1(t)
	cleanupFilenames[2] = pic1.Filename
	require.NoError(t, noDeleteClient.UploadPicture(ctx, pic1, pic2.Filename))
	fixture.AssertPictureUploaded(ctx, t, testWriteApplicationYAMLKey, pic1.Filename)
	fixture.AssertPictureUploaded(ctx, t, testWriteApplicationYAMLKey, pic2.Filename) // Not deleted.

	wg := new(sync.WaitGroup)
	wg.Add(len(cleanupFilenames))
	for _, picName := range &cleanupFilenames {
		go func(filename string) {
			defer wg.Done()
			require.NoError(t, client.UploadPicture(ctx, nil, filename))
			fixture.AssertPictureDeleted(ctx, t, testWriteApplicationYAMLKey, filename)
		}(picName)
	}
	wg.Wait()
}

func TestClientDownloadURL(t *testing.T) {
	t.Parallel()

	assert.Equal(t, "https://ice-staging.b-cdn.net/profile/a.jpg", client.DownloadURL("a.jpg"))
	assert.Equal(t, "a.jpg", client.StripDownloadURL("https://ice-staging.b-cdn.net/profile/a.jpg"))
	assert.Equal(t, "'https://ice-staging.b-cdn.net/profile/' || pic", client.SQLAliasDownloadURL("pic"))
	assert.Equal(t, "a.jpg", New(testReadApplicationYAMLKey).(*picture).uploadURL("a.jpg"))                         //nolint:forcetypeassert // 100% sure.
	assert.Equal(t, "https://storage.bunnycdn.com/ice-staging/profile/a.jpg", client.(*picture).uploadURL("a.jpg")) //nolint:forcetypeassert // 100% sure.
}

func getPic1(tb testing.TB) *multipart.FileHeader {
	tb.Helper()

	return fixture.MultipartFileHeader(tb, pictures, "testing_pic1.jpg", "picture", fmt.Sprintf("%v.jpg", uuid.NewString()))
}

func getPic2(tb testing.TB) *multipart.FileHeader {
	tb.Helper()

	return fixture.MultipartFileHeader(tb, pictures, "testing_pic2.png", "picture", fmt.Sprintf("%v.png", uuid.NewString()))
}
