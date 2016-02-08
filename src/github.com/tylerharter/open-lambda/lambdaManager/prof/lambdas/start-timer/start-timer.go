package main

import (
	"encoding/json"
	"fmt"
	"os"
	"time"
)

type Timestamp struct {
	Time string `json:"startup"`
}

func main() {
	t := Timestamp{
		Time: fmt.Sprint(time.Now().UTC()),
	}

	b, err := json.Marshal(t)
	if err != nil {
		fmt.Println("error")
	} else {
		os.Stdout.Write(b)
	}
}
