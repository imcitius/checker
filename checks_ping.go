package main

import (
	"fmt"
)

func runICMPPingChecks(project project) error {
	var (
		healthy      uint
		failedChecks []string
		alert        TgAlert
		check        UniversalCheck = icmpPingCheck{}
	)
	projectFails := Runtime.Fails.ICMPPing[project.Name]

	for _, check = range project.Checks.ICMPPingChecks {
		_, err := check.Execute()
		if err != nil {
			fmt.Printf("Ping host %v error %v\n", check.HostName(), err)
			Runtime.Fails.ICMPPing[check.UUID()]++

			alert.Message = nonCriticalPING(project.Name, check.HostName(), check.UUID())
			alert.SendNonCrit(project, check)

		} else {
			healthy++
			continue

		}
		failedChecks = append(failedChecks, fmt.Sprintf("{Host: %s}\n", check.HostName()))

		// fmt.Printf("Ping stats: %+v\n\n", stats)
		fmt.Printf("Healthy %d of minimum %d, its %d fail (%d fails allowed)\n", healthy, project.Parameters.MinHealth, projectFails, project.Parameters.AllowFails)
		checkHealth(project, projectFails, healthy, failedChecks)
	}

	fmt.Printf("Healthy %d of minimum %d, its %d fail (%d fails allowed)\n", healthy, project.Parameters.MinHealth, projectFails, project.Parameters.AllowFails)

	return nil
}

func runTCPPingChecks(project project) error {
	var (
		healthy      uint
		failedChecks []string
		alert        TgAlert
		check        UniversalCheck = tcpPingCheck{}
	)

	projectFails := Runtime.Fails.TCPPing[project.Name]

	for _, check = range project.Checks.TCPPingChecks {
		_, err := check.Execute()

		if err != nil {

			fmt.Printf("Ping host %v error %v\n", check.HostName(), err)
			Runtime.Fails.TCPPing[check.UUID()]++

			alert.Message = nonCriticalPING(project.Name, check.HostName(), check.UUID())
			alert.SendNonCrit(project, check)

		} else {
			healthy++
			continue

		}
		failedChecks = append(failedChecks, fmt.Sprintf("{Host: %s}\n", check.HostName()))

		// log.Printf("Ping host %s, res: %+v (err: %+v, stats: %+v)", check.Host, pinger, err, stats)
		// fmt.Printf("Ping stats: %+v\n\n", stats)
		fmt.Printf("Healthy %d of minimum %d, its %d fail (%d fails allowed)\n", healthy, project.Parameters.MinHealth, projectFails, project.Parameters.AllowFails)
		checkHealth(project, projectFails, healthy, failedChecks)

	}

	return nil
}
