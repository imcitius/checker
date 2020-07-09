package check

import (
	//"encoding/json"
	"fmt"
	redis "github.com/go-redis/redis/v7"
	"my/checker/config"
	projects "my/checker/projects"
	"time"
)

func init() {
	Checks["redis_pubsub"] = func(c *config.Check, p *projects.Project) error {

		var dbPort int

		errorHeader := fmt.Sprintf("Redis PubSub error at project: %s\nCheck Host: %s\nCheck UUID: %s\n", p.Name, c.Host, c.UUid)

		dbHost := c.Host
		if c.Port == 0 {
			dbPort = 6379
		} else {
			dbPort = c.Port
		}

		if c.Timeout == "" {
			c.Timeout = config.Config.Defaults.Parameters.ConnectTimeout
		}
		dbConnectTimeout, err := time.ParseDuration(c.Timeout)

		if err != nil {
			config.Log.Warnf("Cannot parse timeout duration: %s (%s)", c.Timeout, c.Type)
		}

		dbPassword := c.PubSub.Password

		connStr := fmt.Sprintf("%s:%d", dbHost, dbPort)

		client := redis.NewClient(&redis.Options{
			Addr:        connStr,
			Password:    dbPassword, // no password set
			DB:          0,          // use default DB
			DialTimeout: dbConnectTimeout,
			ReadTimeout: dbConnectTimeout,
		})

		_, err = client.Ping().Result()
		if err != nil {
			return fmt.Errorf(errorHeader+"redis connect error %+v", err)
		}
		defer client.Close()

		for _, channel := range c.PubSub.Channels {

			pubsub := client.Subscribe(channel)
		loop:

			for {
				msgi, err := pubsub.ReceiveTimeout(dbConnectTimeout)
				if err != nil {
					return fmt.Errorf(errorHeader+"redis pub/sub receive timeout error:\n %+v\n on channel %s", err, channel)
				} else {
					switch msg := msgi.(type) {
					case *redis.Subscription:
						//config.Log.Printf("Received Subscription message on channel %s\n", channel)
						continue
					case *redis.Pong:
						//config.Log.Printf("Received Pong message on channel %s\n", channel)
						continue
					case *redis.Message:
						//config.Log.Printf("Received Data message on channel %s\n", channel)
						//config.Log.Println(msg.Payload, "\n\n")
						break loop
					default:
						err := fmt.Errorf("redis: unknown message: %T on channel %s", msg, channel)
						return fmt.Errorf(errorHeader + err.Error())
					}
				}
			}
		}
		return nil
	}
}
