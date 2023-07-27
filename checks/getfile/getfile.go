package getfile

import (
	"crypto/md5"
	"encoding/hex"
	"errors"
	"fmt"
	"github.com/cavaliergopher/grab/v3"
	"io"
	"os"
	"strings"
	"time"
)

func (c TGetFileCheck) RealExecute() (time.Duration, error) {
	var (
		errorMessage string
		size         int64
	)

	start := time.Now()

	c.Url = strings.TrimRight(c.Url, "/r/n")

	errorHeader := fmt.Sprintf(ErrGetFile)

	tmpfile, err := os.CreateTemp("", fmt.Sprintf("getfile-%s", c.Hash))
	if err != nil {
		errorMessage = errorHeader + fmt.Sprintf(ErrOpenTempFile)
	}
	//logger.Debugf("tmp file name: '%s'", tmpfile.Name())

	// check temp file is not r/o
	err = checkFileWritable(tmpfile)
	if err != nil {
		errorMessage = errorHeader + fmt.Sprintf(ErrCheckWritable)
	}

	defer func() { _ = os.Remove(tmpfile.Name()) }()

	client := grab.NewClient()
	req, err := grab.NewRequest(tmpfile.Name(), c.Url)
	resp := client.Do(req)

	// start UI loop
	t := time.NewTicker(500 * time.Millisecond)
	defer t.Stop()
Loop:
	for {
		select {
		case <-t.C:
			logger.Debugf("  transferred %v / %v bytes (%.2f%%)\n",
				resp.BytesComplete(),
				resp.Size(),
				100*resp.Progress())

		case <-resp.Done:
			// download is complete
			break Loop
		}
	}

	if resp.IsComplete() {
		logger.Debugf("%d bytes transferred", resp.BytesComplete())
	}

	// check for errors
	if err := resp.Err(); err != nil {
		errorMessage = errorHeader + fmt.Sprintf(ErrDownload, err, resp.HTTPResponse.StatusCode)
		logger.Debugf("\n\nhttp response:\n\n%+v\n\n", resp)
		return time.Now().Sub(start), errors.New(errorMessage)
	}

	hash := md5.New()
	if size, err = io.Copy(hash, tmpfile); err != nil {
		errorMessage = errorHeader + fmt.Sprintf(ErrCantReadFile, tmpfile.Name(), err)
		logger.Infof(ErrCantReadFile, tmpfile.Name(), err)
		return time.Now().Sub(start), errors.New(errorMessage)
	}

	// if target file size is set in check, compare with one we get
	if c.Size > 0 {
		logger.Debugf("File size: %d", size)
		if size == 0 || (size != c.Size) {
			errorMessage = errorHeader + fmt.Sprintf(ErrFileSizeMismatch, c.Size, size)
			logger.Infof(ErrFileSizeMismatch, c.Size, size)
			return time.Now().Sub(start), errors.New(errorMessage)
		}
	}

	if c.Hash != "" {
		fileHash := hex.EncodeToString(hash.Sum(nil))
		logger.Debugf("File hash: %s", fileHash)
		if c.Hash != fileHash {
			errorMessage = errorHeader + fmt.Sprintf(ErrFileHashMismatch, c.Hash, fileHash)
			logger.Infof(ErrFileHashMismatch, c.Hash, fileHash)
			return time.Now().Sub(start), errors.New(errorMessage)
		}
	}

	return time.Now().Sub(start), nil
}

func checkFileWritable(tmpfile *os.File) error {
	file, err := os.OpenFile(tmpfile.Name(), os.O_WRONLY, 0666)

	if err != nil {
		logger.Debugf(ErrFileReadOnly, tmpfile.Name())
		return fmt.Errorf(fmt.Sprintf(ErrFileReadOnly, tmpfile.Name()))
	}

	err = file.Close()
	if err != nil {
		logger.Debugf(ErrCloseTempFile, file.Name())
		return fmt.Errorf(fmt.Sprintf(ErrCloseTempFile, tmpfile.Name()))
	}
	return nil
}
