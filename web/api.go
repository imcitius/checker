package web

import (
	"bytes"
	"encoding/json"
	"fmt"
	"github.com/cristalhq/jwt/v3"
	"github.com/pkg/errors"
	"github.com/prometheus/common/log"
	"io"
	"my/checker/config"
	projects "my/checker/projects"
	"my/checker/reports"
	"my/checker/status"
	"net/http"
	"strings"
	"time"
)

type IncomingAlertMessage struct {
	Project  string
	Text     string
	Severity string
}

func incomingAlert(w http.ResponseWriter, r *http.Request) {

	var alert IncomingAlertMessage

	if r.URL.Path != "/alert" {
		http.Error(w, "Not found", http.StatusNotFound)
		return
	}

	if r.Method != "POST" {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	buf := new(bytes.Buffer)
	_, err := buf.ReadFrom(r.Body)
	if err == nil {
		log.Debugf("Alert body is: %s", buf.String())

		err = json.Unmarshal(buf.Bytes(), &alert)
		if err != nil {
			log.Infof("Cannot parse alert body: %s", buf.String())
		}

	} else {
		log.Infof("Cannot parse http request: %s", buf.String())
	}

	if alert.Severity == "critical" {
		projects.GetProjectByName(alert.Project).ProjectCritAlert(errors.Errorf(alert.Text))
	} else {
		projects.GetProjectByName(alert.Project).ProjectAlert(errors.Errorf(alert.Text))
	}
}

func checkPing(w http.ResponseWriter, r *http.Request) {

	if ok := strings.HasPrefix(r.URL.Path, "/check/ping"); !ok {
		http.Error(w, "Not found", http.StatusNotFound)
		return
	}

	if r.Method != "GET" {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	uuid := strings.Split(r.URL.Path, "/")[3]
	if uuid == "" {
		http.Error(w, "Pinged check's UUID not defined", http.StatusMethodNotAllowed)
		config.Log.Debugf("Pinged check's UUID not defined")
		return
	}

	if _, ok := status.Statuses.Checks[uuid]; !ok {
		check := config.Check{UUid: uuid}
		status.InitCheckStatus(&check)
	}

	status.Statuses.Checks[uuid].LastResult = true
	status.Statuses.Checks[uuid].When = time.Now()

	config.Log.Debugf("Passive check %s ping received: %s", uuid, status.Statuses.Checks[uuid].When)

	io.WriteString(w, "Pong\n")
}

func list(w http.ResponseWriter, r *http.Request) {

	if ok := strings.HasPrefix(r.URL.Path, "/list"); !ok {
		http.Error(w, "Not found", http.StatusNotFound)
		return
	}

	if r.Method != "GET" {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	if r.Header.Get("Authorization") == "" {
		config.Log.Info("Web: need auth")
		io.WriteString(w, "Web: need auth")
		return
	} else {
		verifier, err := jwt.NewVerifierHS(jwt.HS256, config.TokenEncryptionKey)
		if err != nil {
			io.WriteString(w, "cannot construct jwt verifier")
			config.Log.Info("cannot construct jwt verifier")
			return
		}

		newToken, err := jwt.ParseAndVerifyString(r.Header.Get("Authorization"), verifier)
		if err != nil {
			io.WriteString(w, "Web: token invalid")
			config.Log.Info("Web: token invalid")
			return
		}

		var newClaims jwt.StandardClaims
		err = json.Unmarshal(newToken.RawClaims(), &newClaims)
		if err != nil {
			io.WriteString(w, "Web: token decoding failed")
			config.Log.Info("Web: token decoding failed")
			return

		}

		var claimed = newClaims.IsForAudience("admin")
		var valid = newClaims.IsValidAt(time.Now())

		if claimed && valid {
			io.WriteString(w, reports.ListElements())
		} else {
			io.WriteString(w, "Web: unauthorized")
			config.Log.Info("Web: unauthorized")
		}
	}
}

func checkStatus(w http.ResponseWriter, r *http.Request) {

	if ok := strings.HasPrefix(r.URL.Path, "/check/status"); !ok {
		http.Error(w, "Not found", http.StatusNotFound)
		return
	}

	if r.Method != "GET" {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	uuid := strings.Split(r.URL.Path, "/")[3]
	if uuid == "" {
		http.Error(w, "Status check's UUID not defined", http.StatusMethodNotAllowed)
		config.Log.Debugf("Status check's UUID not defined")
		return
	}

	if _, ok := status.Statuses.Checks[uuid]; !ok {
		io.WriteString(w, "Empty\n")
		return
	}

	config.Log.Debugf("Check status requested %s", uuid)

	s, err := json.Marshal(status.Statuses.Checks[uuid])
	if err != nil {
		io.WriteString(w, fmt.Sprintf("%s", err))
		return
	}
	io.WriteString(w, fmt.Sprintf("%s", s))
}
