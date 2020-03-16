package main

// chakavaka bot's token
const token string = "201865937:AAHBSXrIlEFSbVfUCvkkd3y4kbvJNgNIJuM"

func main() {

	go runListenBot(token)
	urlChecks(token)

}
