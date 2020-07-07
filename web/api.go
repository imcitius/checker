package web

import (
	"bytes"
	"encoding/json"
	"fmt"
	"github.com/cristalhq/jwt/v3"
	"github.com/pkg/errors"
	"github.com/prometheus/common/log"
	"io"
	"my/checker/alerts"
	"my/checker/config"
	projects "my/checker/projects"
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
		log.Debugf("Alert body is: %s", string(buf.Bytes()))

		err = json.Unmarshal(buf.Bytes(), &alert)
		if err != nil {
			log.Infof("Cannot parse alert body: %s", string(buf.Bytes()))
		}

	} else {
		log.Infof("Cannot parse http request: %s", string(buf.Bytes()))
	}

	if alert.Severity == "critical" {
		alerts.ProjectCritAlert(projects.GetProjectByName(alert.Project), errors.Errorf(alert.Text))
	} else {
		alerts.ProjectAlert(projects.GetProjectByName(alert.Project), errors.Errorf(alert.Text))
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

	io.WriteString(w, "Pong\n")

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

	s := *status.Statuses.Checks[uuid]
	s.LastResult = true
	s.When = time.Now()

	config.Log.Infof("Result: %+v", s)

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
		io.WriteString(w, "Web: need auth	")
		return
	} else {
		verifier, err := jwt.NewVerifierHS(jwt.HS256, key)
		newToken, err := jwt.ParseAndVerifyString(r.Header.Get("Authorization"), verifier)
		if err != nil {
			io.WriteString(w, "Web: token invalid")
			config.Log.Info("Web: token invalid")
			return
		}

		var newClaims jwt.StandardClaims
		errClaims := json.Unmarshal(newToken.RawClaims(), &newClaims)
		checkErr(errClaims)

		var claimed = newClaims.IsForAudience("admin")
		var valid = newClaims.IsValidAt(time.Now())

		if claimed && valid {
			list := ""
			for _, p := range config.Config.Projects {
				list = list + fmt.Sprintf("Project: %s\n", p.Name)
				for _, h := range p.Healthchecks {
					list = list + fmt.Sprintf("\tHealthcheck: %s\n", h.Name)
					for _, c := range h.Checks {
						list = list + fmt.Sprintf("\t\tUUID: %s\n", c.UUid)
					}
				}
			}
			io.WriteString(w, list)
		} else {
			io.WriteString(w, "Web: token invalid")
			config.Log.Info("Web: token invalid")
		}

	}

}
