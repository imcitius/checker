package getfile

import (
	"time"
)

type IGetFileCheck interface {
	RealExecute() (time.Duration, error)
}

type TGetFileCheck struct {
	Project   string
	CheckName string

	Url     string
	Hash    string
	Timeout string
	Size    int64

	ErrorHeader string
}

const (
	ErrWrongCheckType   = "Wrong check type: %s (should be getfile)"
	ErrGetFile          = "File get error "
	ErrEmptyUrl         = "Url is empty"
	ErrEmptyHash        = "Hash of empty file"
	ErrFileReadOnly     = "Temp file '%s' appears to be read-only"
	ErrOpenTempFile     = "Can't open temp file"
	ErrCloseTempFile    = "Error closing temp file: '%s'"
	ErrCheckWritable    = "Temp file is not writable"
	ErrDownload         = "Error downloading file, error: '%s', code: '%d'"
	ErrCantReadFile     = "Cannot read downloaded file %s: %s"
	ErrFileSizeMismatch = "File size mismatch: config size %d, downloaded size: %d"
	ErrFileHashMismatch = "File hash mismatch: config hash %s, downloaded hash: %s"
	//ErrOther            = "other error: %s"
)
