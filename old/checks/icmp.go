package check

import (
	"errors"
	"fmt"
	"github.com/sparrc/go-ping"
	"my/checker/config"
	projects "my/checker/projects"
	"time"
)

func init() {
	Checks["icmp"] = func(c *config.Check, p *projects.Project) error {
		var (
			errorHeader, errorMessage string
		)

		errorHeader = fmt.Sprintf("ICMP error at project: %s\nCheck Host: %s\nCheck UUID: %s\nCheck name: %s\n", p.Name, c.Host, c.UUid, c.Name)

		fmt.Println("icmp ping test: ", c.Host)
		pinger, err := ping.NewPinger(c.Host)
		pinger.Count = c.Count
		pinger.Timeout, _ = time.ParseDuration(c.Timeout)
		pinger.Run()
		stats := pinger.Statistics()

		//config.Log.Printf("Ping host %s, res: %+v (err: %+v, stats: %+v)", c.Host, pinger, err, stats)

		if err == nil && stats.PacketLoss == 0 {
			return nil
		} else {
			switch {
			case stats.PacketLoss > 0:
				//config.Log.Printf("Ping stats: %+v", stats)
				errorMessage = errorHeader + fmt.Sprintf("ping error: %v percent packet loss\n", stats.PacketLoss)
			default:
				//config.Log.Printf("Ping stats: %+v", stats)
				errorMessage = errorHeader + fmt.Sprintf("other ping error: %+v\n", err)
			}
		}

		//config.Log.Println(errorMessage)
		return errors.New(errorMessage)

	}
}
