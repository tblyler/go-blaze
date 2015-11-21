package b2

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"net/url"
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

// FileName B2 file name
type FileName struct {
	ID        string `json:"fileId"`
	Name      string `json:"fileName"`
	Action    string `json:"action"`
	Size      int64  `json:"size"`
	Timestamp int64  `json:"uploadTimestamp"`
}

// Err B2 error information
type Err struct {
	Code    string `json:"code"`
	Message string `json:"message"`
	Status  int    `json:"status"`
}

func (b *Err) Error() string {
	return fmt.Sprintf("code: '%s' status: '%d' message: '%s'", b.Code, b.Status, b.Message)
}

func readResp(resp *http.Response, output interface{}) error {
	data, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return err
	}

	if resp.StatusCode == GoodStatus {
		err = json.Unmarshal(data, output)
		if err != nil {
			return err
		}

		return nil
	}

	errb2 := &Err{}
	err = json.Unmarshal(data, errb2)
	if err != nil {
		return err
	}

	return errb2
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

	b2 := &B2{}

	err = readResp(resp, b2)
	if err != nil {
		return nil, err
	}

	return b2, nil
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

	bucket := &Bucket{}
	err = readResp(resp, bucket)
	if err != nil {
		return nil, err
	}

	return bucket, nil
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

	bucket := &Bucket{}

	err = readResp(resp, bucket)
	if err != nil {
		return nil, err
	}

	return bucket, nil
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

	upload := &Upload{}
	err = readResp(resp, upload)
	if err != nil {
		return nil, err
	}

	return upload, nil
}

// UploadFile uploads one file to B2
func (b *B2) UploadFile(data io.Reader, fileName string, fileSize int64, contentType string, sha1 string, mtime *time.Time, info map[string]string) (*FileInfo, error) {
	if b.Upload == nil {
		return nil, errors.New("Must run GetUploadURL and set B2.Upload to upload")
	}

	req, err := http.NewRequest("POST", b.Upload.UploadURL, data)
	if err != nil {
		return nil, err
	}

	req.ContentLength = fileSize

	if contentType == "" {
		contentType = "b2/x-auto"
	}

	fileEncoded, err := url.Parse(fileName)
	if err != nil {
		return nil, err
	}

	fileName = fileEncoded.String()

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

	fileInfo := &FileInfo{}
	err = readResp(resp, fileInfo)
	if err != nil {
		return nil, err
	}

	return fileInfo, nil
}

// DownloadFileByID Downloads one file from B2
func (b *B2) DownloadFileByID(fileID string, output io.Writer) (http.Header, error) {
	req, err := http.NewRequest("GET", b.DownloadURL+APIsuffix+"/b2_download_file_by_id", nil)
	if err != nil {
		return nil, err
	}

	req.Header.Add("Authorization", b.AuthToken)

	q := req.URL.Query()
	q.Add("fileId", fileID)
	req.URL.RawQuery = q.Encode()

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}

	if resp.StatusCode != GoodStatus {
		return nil, readResp(resp, nil)
	}

	defer resp.Body.Close()

	_, err = io.Copy(output, resp.Body)
	if err != nil {
		return nil, err
	}

	return resp.Header, nil
}

// DownloadFileByName downloads one file by providing the name of the bucket and the name of the file
func (b *B2) DownloadFileByName(bucketName string, fileName string, output io.Writer) (http.Header, error) {
	urlFileName, err := url.Parse(fileName)
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequest("GET", b.DownloadURL+"/file/"+bucketName+"/"+urlFileName.String(), nil)
	if err != nil {
		return nil, err
	}

	req.Header.Add("Authorization", b.AuthToken)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}

	if resp.StatusCode != GoodStatus {
		return nil, readResp(resp, nil)
	}

	defer resp.Body.Close()

	_, err = io.Copy(output, resp.Body)
	if err != nil {
		return nil, err
	}

	return resp.Header, nil
}

// UpdateBucket update an existing bucket
func (b *B2) UpdateBucket(bucketID string, bucketType string) (*Bucket, error) {
	data, err := json.Marshal(map[string]string{
		"accountId":  b.AccountID,
		"bucketId":   bucketID,
		"bucketType": bucketType,
	})
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequest("POST", b.APIUrl+APIsuffix+"/b2_update_bucket", bytes.NewReader(data))
	if err != nil {
		return nil, err
	}

	req.Header.Add("Authorization", b.AuthToken)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}

	bucket := &Bucket{}
	err = readResp(resp, bucket)
	if err != nil {
		return nil, err
	}

	return bucket, nil
}

// DeleteFileVersion deletes one version of a file from B2
func (b *B2) DeleteFileVersion(fileName string, fileID string) (*FileInfo, error) {
	data, err := json.Marshal(map[string]string{
		"fileName": fileName,
		"fileId":   fileID,
	})
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequest("POST", b.APIUrl+APIsuffix+"/b2_delete_file_version", bytes.NewReader(data))
	if err != nil {
		return nil, err
	}

	req.Header.Add("Authorization", b.AuthToken)

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

// ListBuckets lists buckets associated with an account, in alphabetical order by bucket ID
func (b *B2) ListBuckets() ([]Bucket, error) {
	data, err := json.Marshal(map[string]string{
		"accountId": b.AccountID,
	})
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequest("POST", b.APIUrl+APIsuffix+"/b2_list_buckets", bytes.NewReader(data))
	if err != nil {
		return nil, err
	}

	req.Header.Add("Authorization", b.AuthToken)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}

	buckets := &struct {
		Buckets []Bucket `json:"buckets"`
	}{}
	err = readResp(resp, buckets)
	if err != nil {
		return nil, err
	}

	return buckets.Buckets, nil
}

// ListFileNames Lists the names of all files in a bucket, starting at a given name<Paste>
func (b *B2) ListFileNames(bucketID string, startFileName string, maxFileCount int) ([]FileName, string, error) {
	data, err := json.Marshal(struct {
		BucketID      string `json:"bucketId"`
		StartFileName string `json:"startFileName,omitempty"`
		MaxFileCount  int    `json:"maxFileCount,omitempty"`
	}{
		BucketID:      bucketID,
		StartFileName: startFileName,
		MaxFileCount:  maxFileCount,
	})
	if err != nil {
		return nil, "", err
	}

	req, err := http.NewRequest("POST", b.APIUrl+APIsuffix+"/b2_list_file_names", bytes.NewReader(data))
	if err != nil {
		return nil, "", err
	}

	req.Header.Add("Authorization", b.AuthToken)
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, "", err
	}

	list := &struct {
		Files        []FileName `json:"files"`
		NextFileName string     `json:"nextFileName"`
	}{}
	err = readResp(resp, list)
	if err != nil {
		return nil, "", err
	}

	return list.Files, list.NextFileName, nil
}

// ListFileVersions lists all of the versions of all of the files contained in one bucket, in alphabetical order by file name, and by reverse of date/time uploaded for versions of files with the same name
func (b *B2) ListFileVersions(bucketID string, startFileName string, startFileID string, maxFileCount int) ([]FileName, string, string, error) {
	data, err := json.Marshal(struct {
		BucketID      string `json:"bucketId"`
		StartFileName string `json:"startFileName,omitempty"`
		StartFileID   string `json:"startFileId,omitempty"`
		MaxFileCount  int    `json:"maxFileCount,omitempty"`
	}{
		BucketID:      bucketID,
		StartFileName: startFileName,
		StartFileID:   startFileID,
		MaxFileCount:  maxFileCount,
	})
	if err != nil {
		return nil, "", "", err
	}

	req, err := http.NewRequest("POST", b.APIUrl+APIsuffix+"/b2_list_file_versions", bytes.NewReader(data))
	if err != nil {
		return nil, "", "", err
	}

	req.Header.Add("Authorization", b.AuthToken)
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, "", "", err
	}

	list := &struct {
		Files        []FileName `json:"files"`
		NextFileID   string     `json:"nextFileId"`
		NextFileName string     `json:"nextFileName"`
	}{}
	err = readResp(resp, list)
	if err != nil {
		return nil, "", "", err
	}

	return list.Files, list.NextFileID, list.NextFileName, nil
}

// GetFileInfo Gets information about one file stored in B2
func (b *B2) GetFileInfo(fileID string) (*FileInfo, error) {
	data, err := json.Marshal(map[string]string{
		"fileId": fileID,
	})
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequest("POST", b.APIUrl+APIsuffix+"/b2_get_file_info", bytes.NewReader(data))
	if err != nil {
		return nil, err
	}

	req.Header.Add("Authorization", b.AuthToken)
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}

	info := &FileInfo{}
	err = readResp(resp, info)
	if err != nil {
		return nil, err
	}

	return info, nil
}

// HideFile hides a file so that downloading by name will not find the file, but previous versions of the file are still stored. See File Versions about what it means to hide a file
func (b *B2) HideFile(bucketID string, fileName string) (*FileName, error) {
	data, err := json.Marshal(map[string]string{
		"bucketId": bucketID,
		"fileName": fileName,
	})
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequest("POST", b.APIUrl+APIsuffix+"/b2_hide_file", bytes.NewReader(data))
	if err != nil {
		return nil, err
	}

	req.Header.Add("Authorization", b.AuthToken)
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}

	info := &FileName{}
	err = readResp(resp, info)
	if err != nil {
		return nil, err
	}

	return info, nil
}
