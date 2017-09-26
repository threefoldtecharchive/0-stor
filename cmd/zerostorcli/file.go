package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/urfave/cli"
)

func upload(c *cli.Context) error {
	cl, err := getClient(c)
	if err != nil {
		return cli.NewExitError(err, 1)
	}

	if len(c.Args()) < 1 {
		return cli.NewExitError("need to give the path to the file to upload", 1)
	}

	fileName := c.Args().First()

	f, err := os.Open(fileName)
	if err != nil {
		return cli.NewExitError(fmt.Errorf("can't read the file: %v", err), 1)
	}
	defer f.Close()

	var (
		key        = c.String("key")
		references = c.String("reference")
	)

	if key == "" {
		key = filepath.Base(fileName)
	}

	var refList []string
	if len(references) > 0 {
		refList = strings.Split(references, ",")
	}

	_, err = cl.WriteF([]byte(key), f, refList)
	if err != nil {
		return cli.NewExitError(fmt.Errorf("upload failed : %v", err), 1)
	}
	fmt.Printf("file uploaded, key = %v\n", key)
	return nil
}

func download(c *cli.Context) error {
	cl, err := getClient(c)
	if err != nil {
		return cli.NewExitError(err, 1)
	}

	if len(c.Args()) < 2 {
		return cli.NewExitError(fmt.Errorf("need to give the path to the key of file to download and the destination"), 1)
	}

	key := c.Args().Get(0)
	output := c.Args().Get(1)
	fOutput, err := os.Create(output)
	if err != nil {
		return cli.NewExitError(fmt.Errorf("can't create output file: %v", err), 1)
	}

	refList, err := cl.ReadF([]byte(key), fOutput)
	if err != nil {
		return cli.NewExitError(fmt.Errorf("download file failed: %v", err), 1)
	}

	fmt.Printf("file downloaded to %s. referenceList=%v\n", output, refList)

	return nil
}

func delete(c *cli.Context) error {
	cl, err := getClient(c)
	if err != nil {
		return cli.NewExitError(err, 1)
	}

	if len(c.Args()) < 1 {
		return cli.NewExitError(fmt.Errorf("need to give the key of the file to delete"), 1)
	}

	key := c.Args().Get(0)
	if key == "" {
		return cli.NewExitError(fmt.Errorf("need to give the key of the file to delete"), 1)
	}

	err = cl.Delete([]byte(key))
	if err != nil {
		return cli.NewExitError(fmt.Errorf("fail to delete file: %v", err), 1)
	}
	fmt.Println("file deleted successfully")

	return nil
}

func metadata(c *cli.Context) error {
	cl, err := getClient(c)
	if err != nil {
		return cli.NewExitError(err, 1)
	}

	if len(c.Args()) < 1 {
		return cli.NewExitError(fmt.Errorf("need to give the key of the object to inspect"), 1)
	}

	key := c.Args().Get(0)
	if key == "" {
		return cli.NewExitError("key cannot be empty", 1)
	}

	meta, err := cl.GetMeta([]byte(key))
	if err != nil {
		return cli.NewExitError(fmt.Sprintf("fail to get metadata: %v", err), 1)
	}

	b, err := json.Marshal(meta)
	if err != nil {
		return cli.NewExitError("error encoding metadata into json", 1)
	}
	fmt.Print(string(b))

	return nil
}

func repair(c *cli.Context) error {
	cl, err := getClient(c)
	if err != nil {
		return cli.NewExitError(err, 1)
	}

	if len(c.Args()) < 1 {
		return cli.NewExitError(fmt.Errorf("need to give the key of the object to inspect"), 1)
	}

	key := c.Args().Get(0)
	if key == "" {
		return cli.NewExitError("key cannot be empty", 1)
	}

	if err := cl.Repair([]byte(key)); err != nil {
		return cli.NewExitError(fmt.Sprintf("error during repair: %v", err), 1)
	}

	fmt.Println("file properly restored")
	return nil
}
