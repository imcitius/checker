package main

import (
	//"encoding/json"
	"fmt"
	redis "github.com/go-redis/redis/v7"
	"time"
)

func runRedisPubSubCheck(c *Check, p *Project) error {

	var dbPort int

	dbHost := c.Host
	if c.Port == 0 {
		dbPort = 6379
	} else {
		dbPort = c.Port
	}
	dbConnectTimeout := c.Timeout * time.Millisecond
	dbPassword := c.PubSub.Password

	connStr := fmt.Sprintf("%s:%d", dbHost, dbPort)

	client := redis.NewClient(&redis.Options{
		Addr:        connStr,
		Password:    dbPassword, // no password set
		DB:          0,          // use default DB
		DialTimeout: dbConnectTimeout,
		ReadTimeout: dbConnectTimeout,
	})

	_, err := client.Ping().Result()
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
					//log.Printf("Received Subscription message on channel %s\n", channel)
					continue
				case *redis.Pong:
					//log.Printf("Received Pong message on channel %s\n", channel)
					continue
				case *redis.Message:
					//log.Printf("Received Data message on channel %s\n", channel)
					//log.Println(msg.Payload, "\n\n")
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
