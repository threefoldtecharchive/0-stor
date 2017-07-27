package main

import (
	"io/ioutil"
	"log"
	"os"

	"github.com/zero-os/0-stor/client"
)

func main() {
	log.Printf("args=%v", os.Args)
	if len(os.Args) != 4 && len(os.Args) != 5 {
		log.Println("usage:")
		log.Println("./cli conf_file upload file_name")
		log.Println("./cli conf_file download key result_file_name")
		return
	}
	confFile := os.Args[1]
	command := os.Args[2]

	c, err := client.New(confFile)
	if err != nil {
		log.Fatalf("failed to create client: %v", err)
	}
	switch command {
	case "upload":
		fileName := os.Args[3]
		err = uploadFile(c, fileName)
	case "download":
		key := os.Args[3]
		resultFile := os.Args[4]
		err = downloadFile(c, key, resultFile)
	}
	if err != nil {
		log.Fatalf("%v failed: %v", command, err)
	}
	log.Println("Everything looks OK")
}

func uploadFile(c *client.Client, fileName string) error {
	b, err := ioutil.ReadFile(fileName)
	if err != nil {
		return err
	}
	return c.Store([]byte(fileName), b)
}

func downloadFile(c *client.Client, key, resultFile string) error {
	b, err := c.Get([]byte(key))
	if err != nil {
		return err
	}
	return ioutil.WriteFile(resultFile, b, 0666)
}
