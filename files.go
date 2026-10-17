package notitiago

import (
	"context"
	"io"
	"mime"
	"mime/multipart"
	"net/http"
	"net/textproto"
	"path/filepath"
	"strings"
)

var quoteEscaper = strings.NewReplacer(`\`, `\\`, `"`, `\"`)

// UploadFile stores an image, video, audio file or PDF up to 20 MB and returns
// its public URL, which you can use as a notification Image or Icon. The type
// is detected from the content, and uploading the same file twice returns the
// same URL. The upload is bounded by ctx only.
func (c *Client) UploadFile(ctx context.Context, fileName string, r io.Reader) (string, error) {
	name := filepath.Base(strings.TrimSpace(fileName))
	if name == "" || name == "." || name == ".." || name == string(filepath.Separator) {
		return "", invalidArg("file name is required")
	}
	if r == nil {
		return "", invalidArg("file content is required")
	}

	contentType := mime.TypeByExtension(filepath.Ext(name))
	if contentType == "" {
		contentType = "application/octet-stream"
	}

	pr, pw := io.Pipe()
	defer pr.Close()
	mw := multipart.NewWriter(pw)

	go func() {
		header := textproto.MIMEHeader{}
		header.Set("Content-Disposition", `form-data; name="file"; filename="`+quoteEscaper.Replace(name)+`"`)
		header.Set("Content-Type", contentType)

		part, err := mw.CreatePart(header)
		if err == nil {
			_, err = io.Copy(part, r)
		}
		if err == nil {
			err = mw.Close()
		}
		pw.CloseWithError(err)
	}()

	var url string
	err := c.do(ctx, request{
		method:      http.MethodPost,
		path:        c.api("/upload-file"),
		stream:      pr,
		contentType: mw.FormDataContentType(),
	}, &url)
	if err != nil {
		return "", err
	}
	return url, nil
}
