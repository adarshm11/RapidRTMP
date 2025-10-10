package main

import (
	"fmt"
	"rapidrtmp/httpServer"
)

func main() {
	fmt.Println("RapidRTMP")
	api := httpServer.SetupRouter()
	api.Run()
}
