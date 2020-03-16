package main

import (
	"fmt"
	"log"
	"net/http"
	"net/url"
)

func testurls(probes []urlCheck) {

	// fmt.Println(probes)
	for _, s := range probes {
		fmt.Println("test: ", s.URL)
		_, err := url.Parse(s.URL)
		if err != nil {
			log.Fatal(err)
		}

		response, err := http.Get(s.URL)

		switch code := response.StatusCode; {
		default:
			continue
		case code > 400 && code < 500:
			fmt.Printf("The HTTP request failed with error %d\n", response.StatusCode)
			message := fmt.Sprintf("URL: %s\nError code: %d\n", s.URL, response.StatusCode)
			postChannel(1390752, token, message)
		}
	}
}
