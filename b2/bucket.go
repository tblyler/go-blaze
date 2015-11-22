package b2

import (
	"io"
	"time"
)

// Bucket B2 bucket type
type Bucket struct {
	AccountID string `json:"accountId"`
	ID        string `json:"bucketId"`
	Name      string `json:"bucketName"`
	Type      string `json:"bucketType"`
	conn      *B2
	upload    *Upload
}

// Delete deletes this bucket
func (b *Bucket) Delete() error {
	_, err := b.conn.DeleteBucket(b.ID)
	return err
}

// Update updates this bucket
func (b *Bucket) Update(bucketType string) error {
	bucket, err := b.conn.UpdateBucket(b.ID, bucketType)
	if err != nil {
		return err
	}

	b.AccountID = bucket.AccountID
	b.ID = bucket.ID
	b.Name = bucket.Name
	b.Type = bucket.Type

	return nil
}

// ListFileNames Lists the names of all files in a bucket, starting a given name
func (b *Bucket) ListFileNames(startFileName string, maxFileCount int) ([]FileName, string, error) {
	return b.conn.ListFileNames(b.ID, startFileName, maxFileCount)
}

// ListFileVersions lists all of the versions of all of the files contained in one bucket, in alphabetical order by file name, and by reverse of date/time uploaded for versions of files with the same name
func (b *Bucket) ListFileVersions(startFileName string, startFileID string, maxFileCount int) ([]FileName, string, string, error) {
	return b.conn.ListFileVersions(b.ID, startFileName, startFileID, maxFileCount)
}

// HideFile hides a file so that downloading by name will not find the file, but previous versions of the file are still stored. See File Versions about what it means to hide a file
func (b *Bucket) HideFile(fileName string) (*FileName, error) {
	return b.conn.HideFile(b.ID, fileName)
}

// UploadFile uploads one file to B2
func (b *Bucket) UploadFile(data io.Reader, fileName string, fileSize int64, contentType string, sha1 string, mtime *time.Time, info map[string]string) (*FileInfo, error) {
	if b.upload == nil {
		var err error
		b.upload, err = b.conn.GetUploadURL(b.ID)
		if err != nil {
			return nil, err
		}
	}

	return b.upload.UploadFile(data, fileName, fileSize, contentType, sha1, mtime, info)
}
