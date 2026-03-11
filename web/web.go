package web

import (
	"fmt"
	"io"
	"net/http"
	_ "net/http/pprof"

	"github.com/prometheus/client_golang/prometheus/promhttp"
	"golang.org/x/sync/semaphore"

	"my/checker/config"
	"my/checker/internal/db"
	internalslack "my/checker/internal/slack"
	internalweb "my/checker/internal/web"
	"my/checker/slack"
)

var Config = &config.Config

var (
	slackClient *internalslack.SlackClient
	repo        db.Repository
)

// SetSlackClient sets the Slack client used for interactive message handling.
func SetSlackClient(c *internalslack.SlackClient) {
	slackClient = c
}

// SetRepository sets the database repository used for interactive message handling.
func SetRepository(r db.Repository) {
	repo = r
}

func serveHome(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != "/" {
		http.Error(w, "Not found", http.StatusNotFound)
		return
	}
	if r.Method != "GET" {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
}

func healthCheck(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != "/healthcheck" {
		http.Error(w, "Not found", http.StatusNotFound)
		return
	}
	if r.Method != "GET" {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	_, _ = io.WriteString(w, "Ok!\n")
}

func Serve(_ chan bool, sem *semaphore.Weighted) {
	defer sem.Release(1)

	var (
		server *http.Server
		addr   string
	)

	if Config.Defaults.HTTPEnabled != "" {
		return
	}

	addr = fmt.Sprintf(":%s", config.Koanf.String("defaults.http.port"))

	server = new(http.Server)
	server.Addr = addr
	config.Log.Debugf("HTTP listen on: %s", addr)

	http.HandleFunc("/", serveHome)
	http.HandleFunc("/alert", incomingAlert)
	http.HandleFunc("/check/ping/", checkPing)
	http.HandleFunc("/healthcheck", healthCheck)

	http.Handle("/check/status/", authHandler(http.HandlerFunc(checkStatus)))
	http.Handle("/check/fire/", authHandler(http.HandlerFunc(checkFire)))
	http.Handle("/listChecks", authHandler(http.HandlerFunc(listChecks)))
	http.Handle("/metrics", promhttp.Handler())

	// Slack slash command endpoint
	if Config.SlackApp.SigningSecret != "" {
		slackHandler := slack.NewInteractionHandler(Config.SlackApp.SigningSecret)
		http.HandleFunc("/api/slack/commands", slackHandler.HandleSlashCommand)
		config.Log.Info("Slack slash command endpoint registered at /api/slack/commands")

		// Slack interactive message endpoint (button clicks from alert messages)
		interactiveHandler := internalweb.NewSlackInteractiveHandler(
			Config.SlackApp.SigningSecret,
			slackClient,
			repo,
		)
		http.HandleFunc("/api/slack/interactive", interactiveHandler.HandleInteraction)
		config.Log.Info("Slack interactive endpoint registered at /api/slack/interactive")
	}

	if err := server.ListenAndServe(); err != nil {
		config.Log.Fatalf("ListenAndServe: %s", err)
	}
}
