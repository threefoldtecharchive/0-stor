/*
 * Copyright (C) 2017-2018 GIG Technology NV and Contributors
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *    http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

package commands

import (
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"

	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
)

// fileCmd represents the namespace for all file subcommands
var fileCmd = &cobra.Command{
	Use:   "file",
	Short: "Upload or download files to/from (a) 0-stor server(s).",
	// overwrite the namespace config property, if it is given as a flag by the user
	PersistentPreRunE: func(_cmd *cobra.Command, _args []string) error {
		// Override namespace settings
		if fileCfg.Namespace != "" {
			config, err := getClientConfig()
			if err != nil {
				return err
			}
			log.Infof("overwrote namespace config property to '%s' (was: '%s')",
				fileCfg.Namespace, config.Namespace)
			config.Namespace = fileCfg.Namespace
		}
		return nil
	},
}

// Used to hold common file config flags
var fileCfg struct {
	Namespace string
}

// fileUploadCmd represents the file-upload command
var fileUploadCmd = &cobra.Command{
	Use:   "upload [path]",
	Short: "Upload a file.",
	Long:  "Upload a file securely onto (a) 0-stor server(s).",
	Args:  cobra.MaximumNArgs(1),
	RunE: func(_cmd *cobra.Command, args []string) error {
		cl, _, err := getClient()
		if err != nil {
			return err
		}

		var (
			inputName string
			input     io.Reader
		)

		// collect option flags
		key := fileUploadCfg.Key

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
		_, err = cl.Write([]byte(key), input)
		if err != nil {
			return fmt.Errorf("uploading data from %q as %q failed: %v", inputName, key, err)
		}
		log.Infof("data from %q uploaded as key = %q", inputName, key)
		return nil
	},
}

var fileUploadCfg struct {
	Key string
}

// fileDownloadCmd represents the file-download command
var fileDownloadCmd = &cobra.Command{
	Use:   "download <key>",
	Short: "Download a file.",
	Long:  "Download a file which is stored securely onto (a) 0-Stor server(s).",
	Args:  cobra.ExactArgs(1),
	RunE: func(_cmd *cobra.Command, args []string) error {
		cl, metaCli, err := getClient()
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

		md, err := metaCli.GetMetadata([]byte(key))
		if err != nil {
			return fmt.Errorf("downloading file (key: %s) failed to get metadata: %v", key, err)
		}

		err = cl.Read(*md, output)
		if err != nil {
			return fmt.Errorf("downloading file (key: %s) failed: %v", key, err)
		}

		log.Infof("file (key: %s) downloaded", key)
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
		cl, metaCli, err := getClient()
		if err != nil {
			return err
		}

		key := args[0]
		md, err := metaCli.GetMetadata([]byte(key))
		if err != nil {
			return fmt.Errorf("failed to delete file %q. failed to get metadata: %v", key, err)
		}
		err = cl.Delete(*md)
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
		cl, err := getMetaClient()
		if err != nil {
			return err
		}

		key := args[0]
		meta, err := cl.GetMetadata([]byte(key))
		if err != nil {
			return fmt.Errorf("failed to get metadata for %q: %v", key, err)
		}

		output := os.Stdout
		switch {
		case fileMetadataCfg.JSONPrettyFormat:
			log.Debugf("Printing metadata for %q in pretty JSON Format", key)
			writeMetaAsJSON(output, *meta, true)

		case fileMetadataCfg.JSONFormat:
			log.Debugf("Printing metadata for %q in JSON Format", key)
			writeMetaAsJSON(output, *meta, false)

		default:
			log.Debugf("Printing metadata for %q in custom human-readable Format", key)
			writeMetaAsHumanReadableFormat(output, *meta)
		}

		return nil
	},
}

var fileListCmdCfg struct {
	HexFormat bool
}

// fileMetadataCmd represents the file-print-metadata command
var fileListCmd = &cobra.Command{
	Use:   "list",
	Short: "Print all files. ",
	Long:  "Print all files in this namespace",
	Args:  cobra.ExactArgs(0),
	RunE: func(_cmd *cobra.Command, args []string) error {
		cl, err := getMetaClient()
		if err != nil {
			return err
		}

		return cl.ListKeys(func(key []byte) error {
			if fileListCmdCfg.HexFormat {
				fmt.Printf("0x%X\n", key)
			} else {
				fmt.Println(string(key))
			}
			return nil
		})

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
		cl, metaCli, err := getClient()
		if err != nil {
			return err
		}

		key := args[0]

		md, err := metaCli.GetMetadata([]byte(key))
		if err != nil {
			return fmt.Errorf("repair file %q failed. failed to get metadata: %v", key, err)
		}
		_, err = cl.Repair(*md)
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
		fileListCmd,
		fileRepairCmd,
	)

	fileCmd.PersistentFlags().StringVar(
		&fileCfg.Namespace, "namespace", "", "Overrides Namespace (client) config property.")

	fileUploadCmd.Flags().StringVarP(
		&fileUploadCfg.Key, "key", "k", "",
		"Key to use to store the file, required when uploading from STDIN, if empty use the name of the file as the key")

	fileDownloadCmd.Flags().StringVarP(
		&fileDownloadCfg.Output, "output", "o", "",
		"Download the file to the given file, otherwise it will be streamed to the STDOUT.")

	fileListCmd.Flags().BoolVar(
		&fileListCmdCfg.HexFormat, "hex", false,
		"Print the keys in hex format.")

	fileMetadataCmd.Flags().BoolVar(
		&fileMetadataCfg.JSONFormat, "json", false,
		"Print the metadata in JSON format instead of a custom human readable format.")
	fileMetadataCmd.Flags().BoolVar(
		&fileMetadataCfg.JSONPrettyFormat, "json-pretty", false,
		"Print the metadata in prettified JSON format instead of a custom human readable format.")
}
