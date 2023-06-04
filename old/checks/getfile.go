package check

import (
	"crypto/md5"
	"encoding/hex"
	"errors"
	"fmt"
	"github.com/cavaliercoder/grab"
	"github.com/sirupsen/logrus"
	"io"
	"io/ioutil"
	"my/checker/config"
	projects "my/checker/projects"
	"os"
	"strings"
	"time"
)

func init() {
	Checks["getfile"] = func(c *config.Check, p *projects.Project) error {
		var (
			errorMessage string
			size         int64

			logger = *logrus.New()
		)

		if c.DebugLevel != config.Log.Level.String() {
			lvl, err := logrus.ParseLevel(c.DebugLevel)
			if err != nil {
				config.Log.Infof("Cannot parse debug level: '%s'", c.DebugLevel)
			}
			config.Log.Infof("Set log level: '%s'", c.DebugLevel)
			logger.SetLevel(lvl)
		} else {
			logger.SetLevel(config.Log.Level)
		}

		// cleanup url
		c.Host = strings.TrimRight(c.Host, "/r/n")

		errorHeader := fmt.Sprintf("File get error at project: %s\nCheck url: %s\nCheck UUID: %s\n", p.Name, c.Host, c.UUid)
		logger.Infof("Get file test: %s\n", c.Host)

		tmpfile, err := ioutil.TempFile("", fmt.Sprintf("getfile-%s", c.Name))
		logger.Debugf("tmp file name: '%s'", tmpfile.Name())
		defer func() { _ = os.Remove(tmpfile.Name()) }()

		client := grab.NewClient()
		req, err := grab.NewRequest(tmpfile.Name(), c.Host)
		resp := client.Do(req)

		// start UI loop
		t := time.NewTicker(500 * time.Millisecond)
		defer t.Stop()
	Loop:
		for {
			select {
			case <-t.C:
				//fmt.Printf("  transferred %v / %v bytes (%.2f%%)\n",
				//	resp.BytesComplete(),
				//	resp.Size,
				//	100*resp.Progress())
				//
				//
			case <-resp.Done:
				// download is complete
				break Loop
			}
		}

		// check for errors
		if err := resp.Err(); err != nil {
			errorMessage = errorHeader + fmt.Sprintf("Error downloading file, error: '%s', code: '%d'", err, resp.HTTPResponse.StatusCode)
			logger.Debugf("\n\nhttp response:\n\n%+v\n\n", resp)
			return errors.New(errorMessage)
		}

		hash := md5.New()
		if size, err = io.Copy(hash, tmpfile); err != nil {
			errorMessage = errorHeader + fmt.Sprintf("Cannot read downloaded file %s: %s\n", tmpfile.Name(), err)
			logger.Debugf("Cannot read downloaded file %s: %s\n", tmpfile.Name(), err)
			return errors.New(errorMessage)
		}

		if c.Size > 0 && size != c.Size {
			errorMessage = errorHeader + fmt.Sprintf("File size mismatch: config size %d, downloaded size: %d\n", c.Size, size)
			logger.Infof("File size mismatch: config size %d, downloaded size: %d\n", c.Size, size)
			return errors.New(errorMessage)
		}

		fileHash := hex.EncodeToString(hash.Sum(nil))
		if c.Hash != "" && c.Hash != fileHash {
			errorMessage = errorHeader + fmt.Sprintf("File hash mismatch: config hash %s, downloaded hash: %s\n", c.Hash, fileHash)
			logger.Infof("File hash mismatch: config hash %s, downloaded hash: %s\n", c.Hash, fileHash)
			return errors.New(errorMessage)
		}

		return nil
	}
}
