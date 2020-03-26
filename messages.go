package main

import (
	"fmt"
	"log"
)

func nonCriticalHTTP(name, url, uuid string, code int) string {
	log.Println("2")
	return fmt.Sprintf("Http check error\nProject: %s\nURL: %s, code: %d\nUUID: %s\n", name, url, code, uuid)
}

func criticalHTTP(name string, healthy, num, minnum uint, failed []string) string {
	return fmt.Sprintf("Project: %s critical (healthy %d of %d, need %d)\nFailed checks: %v", name, healthy, num, minnum, failed)
}

func nonCriticalPING(name, host, uuid string) string {
	return fmt.Sprintf("Ping check error\nProject: %s\nHOST: %s\nUUID: %s\n", name, host, uuid)
}

func criticalPING(name string, healthy, minnum uint, failed []string) string {
	return fmt.Sprintf("Project: %s critical (healthy %d, need %d)\nFailed checks: %v", name, healthy, minnum, failed)
}
