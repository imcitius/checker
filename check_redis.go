package main

import (
	//"encoding/json"
	"fmt"
	redis "github.com/go-redis/redis/v7"
	"log"
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

	pubsub := client.Subscribe(c.PubSub.Channel)

	for {
		msgi, err := pubsub.ReceiveTimeout(dbConnectTimeout)
		if err != nil {
			return err
		} else {
			switch msg := msgi.(type) {
			case *redis.Subscription:
				log.Println("Received Subscription", msg.Channel, "retry")
				continue
			case *redis.Pong:
				log.Println("Received Pong ", msg.Payload, " on channel")
				continue
			case *redis.Message:
				//log.Println("Received ", msg.Payload, " on channel")
				return nil
			default:
				err := fmt.Errorf("redis: unknown message: %T", msg)
				return err
			}
		}
	}

}
