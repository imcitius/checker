package check

import (
	//"encoding/json"
	"fmt"
	redis "github.com/go-redis/redis/v7"
	"my/checker/config"
	"time"
)

func init() {
	config.Checks["redis_pubsub"] = func (c *config.Check, p *config.Project) error {

		var dbPort int

		dbHost := c.Host
		if c.Port == 0 {
			dbPort = 6379
		} else {
			dbPort = c.Port
		}
		dbConnectTimeout, err := time.ParseDuration(c.Timeout)

		dbPassword := c.PubSub.Password

		connStr := fmt.Sprintf("%s:%d", dbHost, dbPort)

		client := redis.NewClient(&redis.Options{
			Addr:    connStr,
			Password:  dbPassword, // no password set
			DB:     0,     // use default DB
			DialTimeout: dbConnectTimeout,
			ReadTimeout: dbConnectTimeout,
		})

		_, err = client.Ping().Result()
		if err != nil {
			msg := fmt.Errorf("redis connect error %+v", err)
			return msg
		}

		for _, channel := range c.PubSub.Channels {

			pubsub := client.Subscribe(channel)
		loop:

			for {
				msgi, err := pubsub.ReceiveTimeout(dbConnectTimeout)
				if err != nil {
					return err
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
						return err
					}
				}
			}
		}
		return nil
	}
}