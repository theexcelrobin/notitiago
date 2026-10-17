package notitiago

import (
	"bytes"
	"context"
	"errors"
	"io"
	"net/http"
	"strings"
	"testing"
	"testing/iotest"
)

func TestUploadFile(t *testing.T) {
	content := bytes.Repeat([]byte{0x89, 'P', 'N', 'G'}, 1024)
	c := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		expect(t, r, http.MethodPost, "/developers/v1/upload-file")
		if !strings.HasPrefix(r.Header.Get("Content-Type"), "multipart/form-data; boundary=") {
			t.Errorf("content type = %q", r.Header.Get("Content-Type"))
		}
		file, header, err := r.FormFile("file")
		if err != nil {
			t.Fatalf("form file: %v", err)
		}
		defer file.Close()
		got, _ := io.ReadAll(file)
		if header.Filename != "cover.png" || !bytes.Equal(got, content) || header.Header.Get("Content-Type") != "image/png" {
			t.Errorf("filename=%q len=%d type=%q", header.Filename, len(got), header.Header.Get("Content-Type"))
		}
		ok(w, `"https://cdn.lwnotifications.com/projects/p1/upload/9f2c41e0.png"`)
	})
	url, err := c.UploadFile(context.Background(), "cover.png", bytes.NewReader(content))
	if err != nil || url != "https://cdn.lwnotifications.com/projects/p1/upload/9f2c41e0.png" {
		t.Fatalf("url=%q err=%v", url, err)
	}
}

func TestUploadFileStripsDirectoriesAndEscapesName(t *testing.T) {
	c := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		_, header, err := r.FormFile("file")
		if err != nil {
			t.Fatalf("form file: %v", err)
		}
		if header.Filename != `we"ird.pdf` {
			t.Errorf("filename = %q", header.Filename)
		}
		if header.Header.Get("Content-Type") != "application/pdf" {
			t.Errorf("type = %q", header.Header.Get("Content-Type"))
		}
		ok(w, `"https://cdn/x.pdf"`)
	})
	if _, err := c.UploadFile(context.Background(), `../secret/we"ird.pdf`, strings.NewReader("%PDF")); err != nil {
		t.Fatal(err)
	}
}

func TestUploadFileUnknownExtension(t *testing.T) {
	c := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		_, header, _ := r.FormFile("file")
		if header.Header.Get("Content-Type") != "application/octet-stream" {
			t.Errorf("type = %q", header.Header.Get("Content-Type"))
		}
		ok(w, `"u"`)
	})
	if _, err := c.UploadFile(context.Background(), "blob", strings.NewReader("x")); err != nil {
		t.Fatal(err)
	}
}

func TestUploadFileServerRejection(t *testing.T) {
	c := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		_, _ = io.Copy(io.Discard, r.Body)
		fail(w, 400, "Files must be 20 MB or smaller")
	})
	_, err := c.UploadFile(context.Background(), "big.png", strings.NewReader("x"))
	if !errors.Is(err, ErrBadRequest) || !strings.Contains(err.Error(), "20 MB") {
		t.Fatalf("got %v", err)
	}
}

func TestUploadFileReaderError(t *testing.T) {
	c := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		_, _ = io.Copy(io.Discard, r.Body)
		ok(w, `"u"`)
	})
	_, err := c.UploadFile(context.Background(), "a.png", iotest.ErrReader(errors.New("disk gone")))
	if err == nil {
		t.Fatal("a failing reader must fail the upload")
	}
}

func TestUploadFileValidation(t *testing.T) {
	c := newOfflineClient(t)
	for name, fn := range map[string]func() error{
		"no name":   func() error { _, err := c.UploadFile(context.Background(), "", strings.NewReader("x")); return err },
		"dots":      func() error { _, err := c.UploadFile(context.Background(), "..", strings.NewReader("x")); return err },
		"nil input": func() error { _, err := c.UploadFile(context.Background(), "a.png", nil); return err },
	} {
		if err := fn(); !errors.Is(err, ErrInvalidArgument) {
			t.Errorf("%s: got %v", name, err)
		}
	}
}

func TestUploadsAreNotRetried(t *testing.T) {
	calls := 0
	c := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		calls++
		_, _ = io.Copy(io.Discard, r.Body)
		w.Header().Set("Retry-After", "1")
		writeJSON(w, 429, `{"error":"Too Many Requests"}`)
	}, WithMaxRetries(3))
	if _, err := c.UploadFile(context.Background(), "a.png", strings.NewReader("x")); !errors.Is(err, ErrRateLimited) || calls != 1 {
		t.Fatalf("err=%v calls=%d", err, calls)
	}
}
