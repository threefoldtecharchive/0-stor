package commands

import (
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"

	log "github.com/Sirupsen/logrus"
	"github.com/spf13/cobra"
	"github.com/zero-os/0-stor/cmd"
)

// fileCmd represents the namespace for all file subcommands
var fileCmd = &cobra.Command{
	Use:   "file",
	Short: "Upload or download files to/from (a) 0-stor server(s).",
}

// fileUploadCmd represents the file-upload command
var fileUploadCmd = &cobra.Command{
	Use:   "upload [path]",
	Short: "Upload a file.",
	Long:  "Upload a file securely onto (a) 0-stor server(s).",
	Args:  cobra.MaximumNArgs(1),
	RunE: func(_cmd *cobra.Command, args []string) error {
		cl, err := getClient()
		if err != nil {
			return err
		}

		var (
			inputName string
			input     io.Reader
		)

		// collect option flags
		key := fileUploadCfg.Key
		refList := fileUploadCfg.References.Strings()

		// parse optional pos arg and create input reader
		if len(args) == 1 {
			inputName = args[0]
			file, err := os.Open(inputName)
			if err != nil {
				return fmt.Errorf("can't read the file %q: %v", inputName, err)
			}
			defer file.Close()
			input = file
			if key == "" {
				key = filepath.Base(inputName)
			}
		} else { // len(args) == 0
			if key == "" {
				return errors.New("key flag is required when uploading from the STDIN")
			}

			inputName, input = "STDIN", os.Stdin
		}

		// upload the content from the input reader as the given/set key
		_, err = cl.WriteF([]byte(key), input, refList)
		if err != nil {
			return fmt.Errorf("uploading data from %q as %q failed: %v", inputName, key, err)
		}
		log.Infof("data from %q uploaded as key = %q", inputName, key)
		return nil
	},
}

var fileUploadCfg struct {
	Key        string
	References cmd.Strings
}

// fileDownloadCmd represents the file-download command
var fileDownloadCmd = &cobra.Command{
	Use:   "download <key>",
	Short: "Download a file.",
	Long:  "Download a file which is stored securely onto (a) 0-Stor server(s).",
	Args:  cobra.ExactArgs(1),
	RunE: func(_cmd *cobra.Command, args []string) error {
		cl, err := getClient()
		if err != nil {
			return err
		}

		key := args[0]

		var output io.Writer
		if fileDownloadCfg.Output != "" {
			output, err = os.Create(fileDownloadCfg.Output)
			if err != nil {
				return fmt.Errorf("can't create output file %q: %v",
					fileDownloadCfg.Output, err)
			}
		} else {
			output = os.Stdout
		}

		refList, err := cl.ReadF([]byte(key), output)
		if err != nil {
			return fmt.Errorf("downloading file (key: %s) failed: %v", key, err)
		}

		log.Infof("file (key: %s) downloaded, referenceList=%v\n", key, refList)
		return nil
	},
}

var fileDownloadCfg struct {
	Output string
}

// fileDeleteCmd represents the file-delete command
var fileDeleteCmd = &cobra.Command{
	Use:   "delete <key>",
	Short: "Delete a file.",
	Long:  "Delete a file which is stored securely onto (a) 0-Stor server(s).",
	Args:  cobra.ExactArgs(1),
	RunE: func(_cmd *cobra.Command, args []string) error {
		cl, err := getClient()
		if err != nil {
			return err
		}

		key := args[0]
		err = cl.Delete([]byte(key))
		if err != nil {
			return fmt.Errorf("failed to delete file %q: %v", key, err)
		}

		log.Infoln("file deleted successfully")
		return nil
	},
}

// fileMetadataCmd represents the file-print-metadata command
var fileMetadataCmd = &cobra.Command{
	Use:   "metadata <key>",
	Short: "Print the metadata from a file.",
	Long:  "Print the metadata from a file which is stored securely onto (a) 0-Stor server(s).",
	Args:  cobra.ExactArgs(1),
	RunE: func(_cmd *cobra.Command, args []string) error {
		cl, err := getClient()
		if err != nil {
			return err
		}

		key := args[0]
		meta, err := cl.GetMeta([]byte(key))
		if err != nil {
			return fmt.Errorf("failed to get metadata for %q: %v", key, err)
		}

		output := os.Stdout
		switch {
		case fileMetadataCfg.JSONPrettyFormat:
			log.Debugf("Printing metadata for %q in pretty JSON Format", key)
			writeMetaAsJSON(output, meta, true)

		case fileMetadataCfg.JSONFormat:
			log.Debugf("Printing metadata for %q in JSON Format", key)
			writeMetaAsJSON(output, meta, false)

		default:
			log.Debugf("Printing metadata for %q in custom human-readable Format", key)
			writeMetaAsHumanReadableFormat(output, meta)
		}

		return nil
	},
}

var fileMetadataCfg struct {
	JSONFormat       bool
	JSONPrettyFormat bool
}

// fileRepairCmd represents the file-repair command
var fileRepairCmd = &cobra.Command{
	Use:   "repair <key>",
	Short: "Repair a file.",
	Long:  "Repair a file which is stored securely onto (a) 0-Stor server(s).",
	Args:  cobra.ExactArgs(1),
	RunE: func(_cmd *cobra.Command, args []string) error {
		cl, err := getClient()
		if err != nil {
			return err
		}

		key := args[0]

		err = cl.Repair([]byte(key))
		if err != nil {
			return fmt.Errorf("repair file %q failed: %v", key, err)
		}

		log.Infof("file %q properly restored", key)
		return nil
	},
}

func init() {
	fileCmd.AddCommand(
		fileUploadCmd,
		fileDownloadCmd,
		fileDeleteCmd,
		fileMetadataCmd,
		fileRepairCmd,
	)

	fileUploadCmd.Flags().StringVarP(
		&fileUploadCfg.Key, "key", "k", "",
		"Key to use to store the file, required when uploading from STDIN, if empty use the name of the file as the key")
	fileUploadCmd.Flags().VarP(
		&fileUploadCfg.References, "ref", "r",
		"references for this file, split by comma for multiple values")

	fileDownloadCmd.Flags().StringVarP(
		&fileDownloadCfg.Output, "output", "o", "",
		"Download the file to the given file, otherwise it will be streamed to the STDOUT.")

	fileMetadataCmd.Flags().BoolVar(
		&fileMetadataCfg.JSONFormat, "json", false,
		"Print the metadata in JSON format instead of a custom human readable format.")
	fileMetadataCmd.Flags().BoolVar(
		&fileMetadataCfg.JSONPrettyFormat, "json-pretty", false,
		"Print the metadata in prettified JSON format instead of a custom human readable format.")
}
