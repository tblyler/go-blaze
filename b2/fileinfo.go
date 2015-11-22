package b2

import (
	"io"
)

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
	conn      *B2
}

// Download downloads this file ID's content
func (f *FileInfo) Download(output io.Writer) (*FileInfo, error) {
	return f.conn.DownloadFileByID(f.ID, output)
}

// Delete deletes this version of the file
func (f *FileInfo) Delete() (*FileInfo, error) {
	return f.conn.DeleteFileVersion(f.Name, f.ID)
}

// Hide hides a file so that downloading by name will not find the file, but previous versions of the file are still stored. See File Versions about what it means to hide a file
func (f *FileInfo) Hide() (*FileName, error) {
	return f.conn.HideFile(f.BucketID, f.Name)
}
