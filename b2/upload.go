package b2

import (
	"fmt"
	"io"
	"net/http"
	"net/url"
	"time"
)

// Upload B2 upload information
type Upload struct {
	BucketID  string `json:"bucketId"`
	UploadURL string `json:"uploadUrl"`
	AuthToken string `json:"authorizationToken"`
}

// UploadFile uploads one file to B2
func (u *Upload) UploadFile(data io.Reader, fileName string, fileSize int64, contentType string, sha1 string, mtime *time.Time, info map[string]string) (*FileInfo, error) {
	req, err := http.NewRequest("POST", u.UploadURL, data)
	if err != nil {
		return nil, err
	}

	// content length is necessary for buffers like os.File
	req.ContentLength = fileSize

	// use B2's autodetect content type if one is not passed
	if contentType == "" {
		contentType = "b2/x-auto"
	}

	// encode fileName via URL encoding per B2's documentation
	fileEncoded, err := url.Parse(fileName)
	if err != nil {
		return nil, err
	}

	fileName = fileEncoded.String()

	req.Header.Add("Authorization", u.AuthToken)
	req.Header.Add("X-Bz-File-Name", fileName)
	req.Header.Add("Content-Type", contentType)
	req.Header.Add("X-Bz-Content-Sha1", sha1)

	// B2 requires time to be in UNIX milliseconds
	if mtime != nil {
		req.Header.Add("X-Bz-Info-src_last_modified_millis", fmt.Sprint(mtime.UnixNano()/1000000))
	}

	if info != nil {
		for name, value := range info {
			req.Header.Add("X-Bz-Info-"+name, value)
		}
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}

	fileInfo := &FileInfo{}
	err = readResp(resp, fileInfo)
	if err != nil {
		return nil, err
	}

	return fileInfo, nil
}
