package main

import "fmt"

func nonCritical(name, url string, code int) string {

	return fmt.Sprintf("Project: %s;\nURL: %s, code: %d\n", name, url, code)

}

func critical(name string, healthy, num, minnum int, failed []string) string {

	return fmt.Sprintf("Project: %s critical (healthy %d of %d, need %d)\nFailed checks: %v", name, healthy, num, minnum, failed)

}
