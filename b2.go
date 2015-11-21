package b2

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"time"
)

// APIurl base address for the B2 API
const APIurl = "https://api.backblaze.com"

// APIsuffix the version of the API
const APIsuffix = "/b2api/v1"

// GoodStatus status code for a successful API call
const GoodStatus = 200

// ErrGeneric generic error from API
var ErrGeneric = errors.New("Received invalid response from B2 API")

// B2 communicates to B2 API and holds information for the connection
type B2 struct {
	AccountID   string  `json:"accountId"`
	APIUrl      string  `json:"apiUrl"`
	AuthToken   string  `json:"authorizationToken"`
	DownloadURL string  `json:"downloadUrl"`
	AppKey      string  `json:"-"`
	Upload      *Upload `json:"-"`
}

// Upload B2 upload information
type Upload struct {
	BucketID  string `json:"bucketId"`
	UploadURL string `json:"uploadUrl"`
	AuthToken string `json:"authorizationToken"`
}

// Bucket B2 bucket type
type Bucket struct {
	AccountID string `json:"accountId"`
	ID        string `json:"bucketId"`
	Name      string `json:"bucketName"`
	Type      string `json:"bucketType"`
}

// FileInfo B2 file information
type FileInfo struct {
	AccountID string            `json:"accountId"`
	ID        string            `json:"fileId"`
	Name      string            `json:"fileName"`
	BucketID  string            `json:"bucketId"`
	Length    int64             `json:"contentLength"`
	Sha1      string            `json:"contentSha1"`
	Type      string            `json:"contentType"`
	Info      map[string]string `json:"fileInfo"`
}

type b2Err struct {
	Code    string `json:"code"`
	Message string `json:"message"`
	Status  int    `json:"status"`
}

func b2ErrToErr(err *b2Err) error {
	return fmt.Errorf("code: '%s' status: '%d' message: '%s'", err.Code, err.Status, err.Message)
}

func readResp(decoder *json.Decoder, output interface{}) error {
	err := decoder.Decode(output)
	if err != nil && err != io.EOF {
		return err
	}

	return nil
}

// NewB2 create a new B2 API handler
func NewB2(accountID string, applicationKey string) (*B2, error) {
	req, err := http.NewRequest("GET", APIurl+APIsuffix+"/b2_authorize_account", nil)
	if err != nil {
		return nil, err
	}

	req.SetBasicAuth(accountID, applicationKey)
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}

	decoder := json.NewDecoder(resp.Body)

	if resp.StatusCode == GoodStatus {
		b2 := &B2{}

		err := readResp(decoder, b2)
		if err != nil {
			return nil, err
		}

		return b2, nil
	}

	errb2 := &b2Err{}
	err = readResp(decoder, errb2)
	if err != nil {
		return nil, err
	}

	return nil, b2ErrToErr(errb2)
}

// CreateBucket creates a new bucket
func (b *B2) CreateBucket(bucketName string, bucketType string) (*Bucket, error) {
	req, err := http.NewRequest("GET", b.APIUrl+APIsuffix+"/b2_create_bucket", nil)
	if err != nil {
		return nil, err
	}

	req.Header.Add("Authorization", b.AuthToken)
	q := req.URL.Query()
	q.Add("accountId", b.AccountID)
	q.Add("bucketName", bucketName)
	q.Add("bucketType", bucketType)
	req.URL.RawQuery = q.Encode()

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}

	decoder := json.NewDecoder(resp.Body)

	if resp.StatusCode == GoodStatus {
		bucket := &Bucket{}
		err := readResp(decoder, bucket)
		if err != nil {
			return nil, err
		}

		return bucket, nil
	}

	errb2 := &b2Err{}
	err = readResp(decoder, errb2)
	if err != nil {
		return nil, err
	}

	return nil, b2ErrToErr(errb2)
}

// DeleteBucket deletes the bucket specified
func (b *B2) DeleteBucket(bucketID string) (*Bucket, error) {
	data, err := json.Marshal(map[string]string{
		"accountId": b.AccountID,
		"bucketId":  bucketID,
	})
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequest("POST", b.APIUrl+APIsuffix+"/b2_delete_bucket", bytes.NewReader(data))
	if err != nil {
		return nil, err
	}

	req.Header.Add("Authorization", b.AuthToken)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}

	decoder := json.NewDecoder(resp.Body)

	if resp.StatusCode == GoodStatus {
		bucket := &Bucket{}
		err := readResp(decoder, bucket)
		if err != nil {
			return nil, err
		}

		return bucket, nil
	}

	errb2 := &b2Err{}
	err = readResp(decoder, errb2)
	if err != nil {
		return nil, err
	}

	return nil, b2ErrToErr(errb2)
}

// GetUploadURL gets an URL to use for uploading files
func (b *B2) GetUploadURL(bucketID string) (*Upload, error) {
	data, err := json.Marshal(map[string]string{
		"bucketId": bucketID,
	})
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequest("POST", b.APIUrl+APIsuffix+"/b2_get_upload_url", bytes.NewReader(data))
	if err != nil {
		return nil, err
	}

	req.Header.Add("Authorization", b.AuthToken)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}

	decoder := json.NewDecoder(resp.Body)

	if resp.StatusCode == GoodStatus {
		upload := &Upload{}

		err := readResp(decoder, upload)
		if err != nil {
			return nil, err
		}

		return upload, nil
	}

	errb2 := &b2Err{}
	err = readResp(decoder, errb2)
	if err != nil {
		return nil, err
	}

	return nil, b2ErrToErr(errb2)
}

// UploadFile uploads one file to B2
func (b *B2) UploadFile(data io.Reader, fileName string, contentType string, sha1 string, mtime *time.Time, info map[string]string, bucketID string) (*FileInfo, error) {
	if b.Upload == nil {
		if bucketID == "" {
			return nil, errors.New("Must run GetUploadURL and set B2.Upload, or provide bucket id to upload")
		}

		var err error
		b.Upload, err = b.GetUploadURL(bucketID)
		if err != nil {
			return nil, err
		}
	}

	req, err := http.NewRequest("POST", b.Upload.UploadURL, data)
	if err != nil {
		return nil, err
	}

	if contentType == "" {
		contentType = "b2/x-auto"
	}

	req.Header.Add("Authorization", b.Upload.AuthToken)
	req.Header.Add("X-Bz-File-Name", fileName)
	req.Header.Add("Content-Type", contentType)
	req.Header.Add("X-Bz-Content-Sha1", sha1)
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

	decoder := json.NewDecoder(resp.Body)

	if resp.StatusCode == GoodStatus {
		info := &FileInfo{}

		err := readResp(decoder, info)
		if err != nil {
			return nil, err
		}

		return info, nil
	}

	errb2 := &b2Err{}
	err = readResp(decoder, errb2)
	if err != nil {
		return nil, err
	}

	return nil, b2ErrToErr(errb2)
}
