package main

import (
	"bytes"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/urfave/cli"
	"github.com/zero-os/0-stor/client/meta"
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

	json := c.Bool("json")
	pretty := c.Bool("pretty")

	switch {
	case pretty:
		jsonStr, err := structPrettyJSONString(meta)
		if err != nil {
			return cli.NewExitError("error encoding metadata into json", 1)
		}
		fmt.Print(jsonStr)
	case json:
		jsonStr, err := structJSONString(meta)
		if err != nil {
			return cli.NewExitError("error encoding metadata into json", 1)
		}
		fmt.Print(jsonStr)
	default:
		fmt.Print(metaString(meta))
	}

	return nil
}

// metaString turns a meta.Meta struct into a human readable string
func metaString(m *meta.Meta) string {
	if m == nil {
		return "no metadata found"
	}
	var buffer bytes.Buffer

	buffer.WriteString(fmt.Sprintf("Key: %s\n", m.Key))
	buffer.WriteString(fmt.Sprintf("Epoch: %d\n", m.Epoch))
	buffer.WriteString(fmt.Sprintf("Encryption key: %s\n", m.EncrKey))
	buffer.WriteString("Chunks:\n")
	for _, chunk := range m.Chunks {
		buffer.WriteString("\t")
		buffer.WriteString(tabAfterNewLine(chunkString(chunk)))
		buffer.WriteString("\n")
	}
	if m.Previous != nil {
		buffer.WriteString(fmt.Sprintf("Previous: %s\n", m.Previous))
	}
	if m.Next != nil {
		buffer.WriteString(fmt.Sprintf("Next: %s\n", m.Next))
	}
	if m.ConfigPtr != nil {
		buffer.WriteString(fmt.Sprintf("Config pointer: %s\n", m.ConfigPtr))
	}

	return buffer.String()
}

func chunkString(c *meta.Chunk) string {
	if c == nil {
		return "no chunk found\n"
	}
	var buffer bytes.Buffer

	buffer.WriteString(fmt.Sprintf("Key: %s\n", hex.EncodeToString(c.Key)))
	buffer.WriteString(fmt.Sprintf("Size: %d\n", c.Size))
	buffer.WriteString("Shards:\n")
	for _, shard := range c.Shards {
		buffer.WriteString(fmt.Sprintf("\t%s\n", shard))
	}

	return buffer.String()
}

// tabAfterNewLine adds a tab After each `\n` newline character
func tabAfterNewLine(str string) string {
	return strings.Replace(str, "\n", "\n\t", -1)
}

// JSONString returns a flat JSON representation of provided struct
func structJSONString(i interface{}) (string, error) {
	return encodeJSON(i, "")
}

// structPrettyJSONString returns a prettified JSON representation of provided struct
func structPrettyJSONString(i interface{}) (string, error) {
	return encodeJSON(i, "\t")
}

// encodeJSON turns provided struct json string with provided indentation character(s)
func encodeJSON(data interface{}, indent string) (string, error) {
	buffer := new(bytes.Buffer)
	encoder := json.NewEncoder(buffer)
	encoder.SetIndent("", indent)

	err := encoder.Encode(data)
	if err != nil {
		return "", err
	}

	return buffer.String(), nil
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
