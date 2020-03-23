package main

import (
	"fmt"
	"log"
	"net"
	"time"

	"github.com/sparrc/go-ping"
)

func runICMPPingChecks(project project) error {
	var (
		healthy      uint
		checkNum     uint
		failedChecks []string
		checkGood    bool
	)

	projectFails := Runtime.Fails.ICMPPing[project.Name]

	for _, check := range project.Checks.ICMPPingChecks {
		fmt.Println("test: ", check.Host)
		checkNum++
		pinger, err := ping.NewPinger(check.Host)
		pinger.Count = int(check.Count)
		pinger.Timeout = check.Timeout * 1000 * 1000 //milliseconds
		pinger.Run()
		stats := pinger.Statistics()

		// log.Printf("Ping host %s, res: %+v (err: %+v, stats: %+v)", check.Host, pinger, err, stats)

		if stats.PacketLoss == 0 && stats.AvgRtt > check.Timeout {
			checkGood = true
		}
		if checkGood {
			healthy++
			continue
		} else {
			fmt.Printf("Ping host %v error %v\n", check.Host, err)
			message := nonCriticalPING(project.Name, check.Host, check.uuID)

			if Config.Defaults.Parameters.Mode == "loud" && project.Parameters.Mode == "loud" {
				log.Printf("Project Loud mode,")
				if check.Mode != "quiet" {
					log.Printf("Check Loud mode:\n%v\n", message)
					// log.Printf("Ask to send alert: channel %d, token %s, message %s", project.Parameters.ProjectChannel, project.Parameters.BotToken, message)
					sendAlert(project.Parameters.ProjectChannel, project.Parameters.BotToken, message)
				} else {
					log.Printf("Check Quiet mode:\n%v\n", message)
				}
			} else {
				log.Printf("Project Quiet mode:\n%v\n", message)
			}
			failedChecks = append(failedChecks, fmt.Sprintf("{Host: %s}\n", check.Host))
		}

		fmt.Printf("Ping stats: %+v\n\n", stats)
		fmt.Printf("Healthy %d of minimum %d, its %d fail (%d fails allowed)\n", healthy, project.Parameters.MinHealth, projectFails, project.Parameters.AllowFails)
		if healthy >= project.Parameters.MinHealth {
			if projectFails > 0 {
				projectFails--
			}
			continue
		} else {
			if project.Parameters.AllowFails > projectFails {
				projectFails++
				continue
			} else {
				message := criticalPING(project.Name, healthy, checkNum, project.Parameters.MinHealth, failedChecks)
				sendAlert(project.Parameters.CriticalChannel, project.Parameters.BotToken, message)
			}
		}
	}

	return nil
}

func runTCPPingChecks(project project) error {
	var (
		healthy       uint
		checkNum      uint
		checkAttempts uint
		failedChecks  []string
		checkGood     bool
	)

	projectFails := Runtime.Fails.TCPPing[project.Name]

	for _, check := range project.Checks.TCPPingChecks {
		checkNum++
		checkhost := fmt.Sprintf("%s:%d", check.Host, check.Port)
		timeout := check.Timeout * 1000 * 1000 // millisecond
		fmt.Printf("test: %s\n", checkhost)

		for checkAttempts < check.Attempts {
			startTime := time.Now()
			conn, err := net.DialTimeout("tcp", checkhost, timeout)
			endTime := time.Now()

			if err != nil {

				log.Printf("connection failed: %v (attempt %d)\n", err, checkAttempts)
				checkAttempts++
			} else {
				defer conn.Close()
				var t = float64(endTime.Sub(startTime)) / float64(time.Millisecond)
				log.Printf("Connection to host %v succeed, took %v millisec", conn.RemoteAddr().String(), t)
				checkGood = true
				break
			}

		}
		// log.Printf("Ping host %s, res: %+v (err: %+v, stats: %+v)", check.Host, pinger, err, stats)

		if checkGood {
			healthy++
			continue
		} else {
			fmt.Printf("TCP ping host error %v in %d attempts\n", check.Host, check.Attempts)
			message := nonCriticalPING(project.Name, check.Host, check.uuID)

			if Config.Defaults.Parameters.Mode == "loud" && project.Parameters.Mode == "loud" {
				log.Printf("Project Loud mode,")
				if check.Mode != "quiet" {
					log.Printf("Check Loud mode:\n%v\n", message)
					// log.Printf("Ask to send alert: channel %d, token %s, message %s", project.Parameters.ProjectChannel, project.Parameters.BotToken, message)
					sendAlert(project.Parameters.ProjectChannel, project.Parameters.BotToken, message)
				} else {
					log.Printf("Check Quiet mode:\n%v\n", message)
				}
			} else {
				log.Printf("Project Quiet mode:\n%v\n", message)
			}
			failedChecks = append(failedChecks, fmt.Sprintf("{Host: %s}\n", check.Host))
		}

		// fmt.Printf("Ping stats: %+v\n\n", stats)
		fmt.Printf("Healthy %d of minimum %d, its %d fail (%d fails allowed)\n", healthy, project.Parameters.MinHealth, projectFails, project.Parameters.AllowFails)
		if healthy >= project.Parameters.MinHealth {
			if projectFails > 0 {
				projectFails--
			}
			continue
		} else {
			if project.Parameters.AllowFails > projectFails {
				projectFails++
				continue
			} else {
				message := criticalPING(project.Name, healthy, checkNum, project.Parameters.MinHealth, failedChecks)
				sendAlert(project.Parameters.CriticalChannel, project.Parameters.BotToken, message)
			}
		}
	}

	return nil
}
