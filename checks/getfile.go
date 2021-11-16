package check

import (
	"crypto/md5"
	"encoding/hex"
	"errors"
	"fmt"
	"github.com/cavaliercoder/grab"
	"io"
	"io/ioutil"
	"my/checker/config"
	projects "my/checker/projects"
	"os"
)

func init() {
	Checks["getfile"] = func(c *config.Check, p *projects.Project) error {
		var (
			errorMessage string
		)

		errorHeader := fmt.Sprintf("File get error at project: %s\nCheck url: %s\nCheck UUID: %s\n", p.Name, c.Host, c.UUid)
		config.Log.Infof("Get file test: %s\n", c.Host)

		//config.Log.Panic(timeout)

		tmpfile, err := ioutil.TempFile("", "getfile")
		defer func() { _ = os.Remove(tmpfile.Name()) }()

		resp, err := grab.Get(tmpfile.Name(), c.Host)
		if err != nil {
			errorMessage = errorHeader + fmt.Sprintf("Error downloading %s: %v\n", c.Host, err)
			return errors.New(errorMessage)
		}
		if resp.HTTPResponse.Status != "200" {
			errorMessage = errorHeader + fmt.Sprintf("Error downloading file, code %s", resp.HTTPResponse.Status)
			return errors.New(errorMessage)
		}

		hash := md5.New()

		if _, err := io.Copy(hash, tmpfile); err != nil {
			errorMessage = errorHeader + fmt.Sprintf("Cannot read downloaded file %s: %s\n", tmpfile.Name(), err)
			return errors.New(errorMessage)
		}

		fileHash := hex.EncodeToString(hash.Sum(nil))
		if c.Hash != fileHash {
			errorMessage = errorHeader + fmt.Sprintf("File hash mismatch: config hash %s, downloaded hash: %s\n", c.Hash, fileHash)
			//config.Log.Infof("File hash mismatch: config hash %s, downloaded hash: %s\n", c.Hash, fileHash)
			return errors.New(errorMessage)
		}

		return nil
	}
}
