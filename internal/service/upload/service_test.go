package upload

import (
	"bytes"
	"mime/multipart"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestSaveFileSuccess(t *testing.T) {
	dir := t.TempDir()
	svc := NewService(dir)

	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)
	part, err := writer.CreateFormFile("file", "test.pdf")
	require.NoError(t, err)
	_, err = part.Write([]byte("fake-pdf-content"))
	require.NoError(t, err)
	require.NoError(t, writer.Close())

	_, header, err := readerFromForm(body, writer)
	require.NoError(t, err)

	result, err := svc.SaveFile(header)
	require.NoError(t, err)
	require.Equal(t, "test.pdf", result.OriginalName)
	require.Equal(t, "application/pdf", result.ContentType)
	require.Equal(t, int64(16), result.FileSize)
	require.Contains(t, result.FilePath, "documents/")
	require.Contains(t, result.FilePath, "test.pdf")

	_, err = os.Stat(filepath.Join(dir, result.FilePath))
	require.NoError(t, err)
}

func TestSaveFileWithSceneAvatar(t *testing.T) {
	dir := t.TempDir()
	svc := NewService(dir)

	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)
	part, err := writer.CreateFormFile("file", "avatar.png")
	require.NoError(t, err)
	_, err = part.Write([]byte("fake-png-content"))
	require.NoError(t, err)
	require.NoError(t, writer.Close())

	_, header, err := readerFromForm(body, writer)
	require.NoError(t, err)

	result, err := svc.SaveFileWithScene(header, "avatar")
	require.NoError(t, err)
	require.Contains(t, result.FilePath, "avatars/")
	require.Contains(t, result.FilePath, "avatar.png")
}

func TestSaveFileAutoRoutesImagesToImagesCategory(t *testing.T) {
	dir := t.TempDir()
	svc := NewService(dir)

	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)
	part, err := writer.CreateFormFile("file", "poster.jpg")
	require.NoError(t, err)
	_, err = part.Write([]byte("fake-jpg-content"))
	require.NoError(t, err)
	require.NoError(t, writer.Close())

	_, header, err := readerFromForm(body, writer)
	require.NoError(t, err)

	result, err := svc.SaveFile(header)
	require.NoError(t, err)
	require.Contains(t, result.FilePath, "images/")
	require.Contains(t, result.FilePath, "poster.jpg")
}

func TestSaveFileRejectsLargeFile(t *testing.T) {
	dir := t.TempDir()
	svc := NewService(dir)

	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)
	part, err := writer.CreateFormFile("file", "big.pdf")
	require.NoError(t, err)
	large := make([]byte, 1024*1024)
	for i := 0; i < 31; i++ {
		_, err = part.Write(large)
		require.NoError(t, err)
	}
	require.NoError(t, writer.Close())

	_, header, err := readerFromForm(body, writer)
	require.NoError(t, err)

	_, err = svc.SaveFile(header)
	require.Error(t, err)
	require.Contains(t, err.Error(), "too large")
}

func TestSaveFileRejectsUnsupportedType(t *testing.T) {
	dir := t.TempDir()
	svc := NewService(dir)

	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)
	part, err := writer.CreateFormFile("file", "malware.exe")
	require.NoError(t, err)
	_, err = part.Write([]byte("fake-exe"))
	require.NoError(t, err)
	require.NoError(t, writer.Close())

	_, header, err := readerFromForm(body, writer)
	require.NoError(t, err)

	_, err = svc.SaveFile(header)
	require.Error(t, err)
	require.Contains(t, err.Error(), "unsupported")
}

func readerFromForm(body *bytes.Buffer, writer *multipart.Writer) (*multipart.Reader, *multipart.FileHeader, error) {
	reader := multipart.NewReader(body, writer.Boundary())
	form, err := reader.ReadForm(10 * 1024 * 1024)
	if err != nil {
		return nil, nil, err
	}
	files := form.File["file"]
	if len(files) == 0 {
		return nil, nil, nil
	}
	return reader, files[0], nil
}
