package upload_test

import (
	"bytes"
	"context"
	"errors"
	"io"
	"strings"
	"testing"

	appshared "github.com/zatrano/gocore/internal/application/shared"
	appupload "github.com/zatrano/gocore/internal/application/upload"
)

type stubStorage struct {
	err error
}

func (s stubStorage) Put(_ context.Context, key string, _ io.Reader, contentType string, size int64) (appshared.FileObject, error) {
	if s.err != nil {
		return appshared.FileObject{}, s.err
	}
	return appshared.FileObject{Key: key, ContentType: contentType, Size: size}, nil
}

func (s stubStorage) Get(context.Context, string) (io.ReadCloser, appshared.FileObject, error) {
	return nil, appshared.FileObject{}, errors.New("not implemented")
}

func (s stubStorage) Delete(context.Context, string) error { return nil }

func pngFile(name string) appupload.IncomingFile {
	data := []byte{0x89, 0x50, 0x4E, 0x47, 0x0D, 0x0A, 0x1A, 0x0A}
	return appupload.IncomingFile{
		Filename: name,
		Size:     int64(len(data)),
		Open: func() (io.ReadCloser, error) {
			return io.NopCloser(bytes.NewReader(data)), nil
		},
	}
}

func TestService_UploadBatch_success(t *testing.T) {
	svc := appupload.NewService(stubStorage{}, 1024, []string{"image/png"}, nil)
	res := svc.UploadBatch(context.Background(), []appupload.IncomingFile{
		pngFile("a.png"),
		pngFile("b.png"),
	})
	if res.Total != 2 || res.Accepted != 2 || len(res.Invalid) != 0 {
		t.Fatalf("unexpected result: %+v", res)
	}
}

func TestService_UploadBatch_partial(t *testing.T) {
	svc := appupload.NewService(stubStorage{}, 8, []string{"image/png"}, nil)
	res := svc.UploadBatch(context.Background(), []appupload.IncomingFile{
		pngFile("ok.png"),
		{Filename: "big.png", Size: 20, Open: pngFile("big.png").Open},
	})
	if res.Accepted != 1 || len(res.Invalid) != 1 || res.Invalid[0].Reason != "upload.too_large" {
		t.Fatalf("unexpected result: %+v", res)
	}
}

func TestService_UploadBatch_unsupportedType(t *testing.T) {
	svc := appupload.NewService(stubStorage{}, 1024, []string{"image/png"}, nil)
	bad := appupload.IncomingFile{
		Filename: "x.txt",
		Size:     3,
		Open: func() (io.ReadCloser, error) {
			return io.NopCloser(strings.NewReader("abc")), nil
		},
	}
	res := svc.UploadBatch(context.Background(), []appupload.IncomingFile{bad})
	if res.Accepted != 0 || len(res.Invalid) != 1 || res.Invalid[0].Reason != "upload.unsupported_type_detail" {
		t.Fatalf("unexpected result: %+v", res)
	}
}
