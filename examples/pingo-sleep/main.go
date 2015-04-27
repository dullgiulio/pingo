package main

import "time"

func main() {
	<-time.After(10 * time.Second)
}
