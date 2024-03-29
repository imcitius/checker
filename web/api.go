package web

import (
	"bytes"
	"encoding/json"
	"fmt"
	"github.com/cristalhq/jwt/v3"
	"github.com/pkg/errors"
	"github.com/prometheus/common/log"
	"io"
	checks "my/checker/checks"
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

	err := status.PingCheck(config.GetCheckByUUID(uuid))
	if err != nil {
		http.Error(w, fmt.Sprintf("Unable to ping check %s", uuid), http.StatusMethodNotAllowed)
		config.Log.Infof("Unable to ping check %s", uuid)
		return
	}

	config.Log.Infof("Passive check %s ping received: %s", uuid, status.Statuses.Checks[uuid].When)

	_, _ = io.WriteString(w, "Pong\n")
}

func listChecks(w http.ResponseWriter, r *http.Request) {

	if ok := strings.HasPrefix(r.URL.Path, "/listChecks"); !ok {
		http.Error(w, "Not found", http.StatusNotFound)
		return
	}

	if r.Method != "GET" {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	if r.Header.Get("Authorization") == "" {
		config.Log.Info("Web: need auth")
		_, _ = io.WriteString(w, "Web: need auth")
		return
	} else {
		claimed, valid := checkWebAuth(r.Header.Get("Authorization"))
		if claimed && valid {
			list := reports.List()
			_, _ = io.WriteString(w, list)
		} else {
			_, _ = io.WriteString(w, "Web: unauthorized")
			config.Log.Info("Web: unauthorized")
		}
	}
}

func checkWebAuth(authHeader string) (bool, bool) {

	verifier, err := jwt.NewVerifierHS(jwt.HS256, config.TokenEncryptionKey)
	if err != nil {
		config.Log.Info("cannot construct jwt verifier")
		return false, false
	}

	newToken, err := jwt.ParseAndVerifyString(authHeader, verifier)
	if err != nil {
		config.Log.Info("Web: token invalid")
		return false, false
	}

	var newClaims jwt.StandardClaims
	err = json.Unmarshal(newToken.RawClaims(), &newClaims)
	if err != nil {
		config.Log.Info("Web: token decoding failed")
		return false, false
	}

	claimed := newClaims.IsForAudience("admin")
	valid := newClaims.IsValidAt(time.Now())

	return claimed, valid
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

	if config.GetCheckByUUID(uuid) == nil {
		http.Error(w, "Check not found", http.StatusNotFound)
		config.Log.Debugf("Check with UUID %s not found", uuid)
		return
	}

	if _, ok := status.Statuses.Checks[uuid]; !ok {
		_, _ = io.WriteString(w, "Empty\n")
		return
	}

	config.Log.Debugf("Check status requested %s", uuid)

	s, err := json.Marshal(status.Statuses.Checks[uuid])
	if err != nil {
		_, _ = io.WriteString(w, fmt.Sprintf("%s", err))
		return
	}
	_, _ = io.WriteString(w, fmt.Sprintf("%s", s))
}

type answer struct {
	Name     string
	UUID     string
	Status   string
	Error    string
	Duration string
}

func checkFire(w http.ResponseWriter, r *http.Request) {

	if ok := strings.HasPrefix(r.URL.Path, "/check/fire"); !ok {
		http.Error(w, "Not found", http.StatusNotFound)
		return
	}

	if r.Method != "GET" {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	uuid := strings.Split(r.URL.Path, "/")[3]
	if uuid == "" {
		http.Error(w, "Execution check's UUID not defined", http.StatusMethodNotAllowed)
		config.Log.Debugf("Execution check's UUID not defined")
		return
	}

	if config.GetCheckByUUID(uuid) == nil {
		http.Error(w, "Check not found", http.StatusNotFound)
		config.Log.Debugf("Check with UUID %s not found", uuid)
		return
	}

	check := config.GetCheckByUUID(uuid)
	project := new(projects.Project)
	project.Project = *config.GetProjectByCheckUUID(uuid)
	config.Log.Debugf("Check execution requested '%s', '%s'", check.Name, uuid)

	duration, tempErr := checks.Execute(project, check)

	a := answer{
		Name:     check.Name,
		UUID:     check.UUid,
		Error:    "",
		Duration: duration.String(),
		Status:   "Ok",
	}
	if tempErr != nil {
		a.Status = "Failed"
		a.Error = tempErr.Error()
	}

	s, err := json.Marshal(a)
	if err != nil {
		_, _ = io.WriteString(w, fmt.Sprintf("%s", err))
		return
	}
	_, _ = io.WriteString(w, fmt.Sprintf("%s", s))
}

func authHandler(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {

		if r.Header.Get("Authorization") == "" {
			config.Log.Info("Web: need auth")
			_, _ = io.WriteString(w, "Web: need auth")
			return
		} else {
			claimed, valid := checkWebAuth(r.Header.Get("Authorization"))
			if claimed && valid {
				config.Log.Info("auth pass")
				next.ServeHTTP(w, r)
			} else {
				_, _ = io.WriteString(w, "Web: unauthorized")
				config.Log.Info("Web: unauthorized")
			}
		}
	})
}
