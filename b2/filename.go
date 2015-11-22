package b2

// FileName B2 file name
type FileName struct {
	ID        string `json:"fileId"`
	Name      string `json:"fileName"`
	Action    string `json:"action"`
	Size      int64  `json:"size"`
	Timestamp int64  `json:"uploadTimestamp"`
	conn      *B2
}

// GetFileInfo Gets information about one file stored in B2
func (f *FileName) GetFileInfo() (*FileInfo, error) {
	return f.conn.GetFileInfo(f.ID)
}
