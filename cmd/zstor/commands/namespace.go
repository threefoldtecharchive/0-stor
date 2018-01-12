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
	"encoding/json"
	"fmt"
	"os"

	"github.com/zero-os/0-stor/client/itsyouonline"

	log "github.com/Sirupsen/logrus"
	"github.com/spf13/cobra"
)

// namespaceCmd represents the namespace for all namespace subcommands
var namespaceCmd = &cobra.Command{
	Use:   "namespace",
	Short: "Manage namespaces and their permissions.",
}

var namespaceCmdCfg struct {
	namespace string
}

func preRunNamespaceCommands(requiredArgs int) func(*cobra.Command, []string) error {
	return func(_cmd *cobra.Command, args []string) error {
		n := len(args)
		if n < requiredArgs {
			return fmt.Errorf("required at least %d arg(s), received %d", requiredArgs, n)
		}
		maxArgs := requiredArgs + 1
		if n > maxArgs {
			return fmt.Errorf("accepting maximum %d arg(s), received %d", maxArgs, n)
		}

		if n == maxArgs {
			namespace := args[requiredArgs]

			config, err := getClientConfig()
			if err != nil {
				return err
			}
			log.Infof("overwrote namespace config property to '%s' (was: '%s')",
				namespace, config.Namespace)
			// Override namespace settings
			config.Namespace = namespace
		}

		return nil
	}
}

// namespaceCreateCmd represents the namespace-create command
var namespaceCreateCmd = &cobra.Command{
	Use:   "create [namespace]",
	Short: "Create a namespace.",
	Long: `Create namespace in IYO within an organization for a given name.

If no namespace is given as positional argument, it is assumed that the namespace
as defined in the (client) config file, is to be used.`,
	PreRunE: preRunNamespaceCommands(0),
	RunE: func(_cmd *cobra.Command, args []string) error {
		iyoCl, err := getNamespaceManager()
		if err != nil {
			return err
		}

		name := args[0]

		err = iyoCl.CreateNamespace(name)
		if err != nil {
			return fmt.Errorf("creation of namespace %q failed: %v", name, err)
		}

		log.Infof("namespace %q created", name)
		return nil
	},
}

// namespaceDeleteCmd represents the namespace-delete command
var namespaceDeleteCmd = &cobra.Command{
	Use:   "delete [namespace]",
	Short: "Delete a namespace.",
	Long: `Delete namespace in IYO existing within an organization under a given name.

If no namespace is given as positional argument, it is assumed that the namespace
as defined in the (client) config file, is to be used.`,
	PreRunE: preRunNamespaceCommands(0),
	RunE: func(_cmd *cobra.Command, args []string) error {
		iyoCl, err := getNamespaceManager()
		if err != nil {
			return err
		}

		name := args[0]

		err = iyoCl.DeleteNamespace(name)
		if err != nil {
			return fmt.Errorf("deletion of namespace %q failed: %v", name, err)
		}

		log.Infof("namespace %q deleted", name)
		return nil
	},
}

// namespacePermissionCmd represents the namespace for all namespace-permission subcommands
var namespacePermissionCmd = &cobra.Command{
	Use:   "permission",
	Short: "Manage permissions of namespaces.",
}

// namespaceSetPermissionCmd represents the namespace-permission-set command
var namespaceSetPermissionCmd = &cobra.Command{
	Use:   "set <userID> [namespace]",
	Short: "Set permissions.",
	Long: `Set permissions for a given user and namespace.

If no namespace is given as positional argument, it is assumed that the namespace
as defined in the (client) config file, is to be used.`,
	PreRunE: preRunNamespaceCommands(1),
	RunE: func(_cmd *cobra.Command, args []string) error {
		iyoCl, err := getNamespaceManager()
		if err != nil {
			return err
		}

		userID, namespace := args[0], args[1]
		currentpermissions, err := iyoCl.GetPermission(namespace, userID)
		if err != nil {
			return fmt.Errorf("fail to retrieve permission(s) for %s:%s: %v",
				userID, namespace, err)
		}

		// remove permission if needed
		toRemove := itsyouonline.Permission{
			Read:   currentpermissions.Read && !namespaceSetPermissionCfg.Read,
			Write:  currentpermissions.Write && !namespaceSetPermissionCfg.Write,
			Delete: currentpermissions.Delete && !namespaceSetPermissionCfg.Delete,
			Admin:  currentpermissions.Admin && !namespaceSetPermissionCfg.Admin,
		}
		if err := iyoCl.RemovePermission(namespace, userID, toRemove); err != nil {
			return fmt.Errorf("fail to remove permission(s) for %s:%s: %v",
				userID, namespace, err)
		}

		// add permission if needed
		toAdd := itsyouonline.Permission{
			Read:   !currentpermissions.Read && namespaceSetPermissionCfg.Read,
			Write:  !currentpermissions.Write && namespaceSetPermissionCfg.Write,
			Delete: !currentpermissions.Delete && namespaceSetPermissionCfg.Delete,
			Admin:  !currentpermissions.Admin && namespaceSetPermissionCfg.Admin,
		}

		// Give requested permission
		if err := iyoCl.GivePermission(namespace, userID, toAdd); err != nil {
			return fmt.Errorf("fail to give permission(s) for %s:%s: %v",
				userID, namespace, err)
		}

		return nil
	},
}

var namespaceSetPermissionCfg struct {
	Read, Write, Delete, Admin bool
}

var namespaceGetPermissionCfg struct {
	JSONFormat       bool
	JSONPrettyFormat bool
}

// namespaceGetPermissionCmd represents the namespace-permission-get command
var namespaceGetPermissionCmd = &cobra.Command{
	Use:   "get <userID> [namespace]",
	Short: "Get permissions.",
	Long: `Get permissions for a given user and namespace.

If no namespace is given as positional argument, it is assumed that the namespace
as defined in the (client) config file, is to be used.`,
	PreRunE: preRunNamespaceCommands(1),
	RunE: func(_cmd *cobra.Command, args []string) error {
		iyoCl, err := getNamespaceManager()
		if err != nil {
			return err
		}

		userID, namespace := args[0], args[1]
		perm, err := iyoCl.GetPermission(namespace, userID)
		if err != nil {
			return fmt.Errorf("failed to retrieve permission for %s:%s: %v",
				userID, namespace, err)
		}

		switch {
		case namespaceGetPermissionCfg.JSONPrettyFormat, namespaceGetPermissionCfg.JSONFormat:
			encoder := json.NewEncoder(os.Stdout)
			if namespaceGetPermissionCfg.JSONPrettyFormat {
				encoder.SetIndent("", "\t")
			}
			err := encoder.Encode(struct {
				Read   bool `json:"read"`
				Write  bool `json:"write"`
				Delete bool `json:"delete"`
				Admin  bool `json:"admin"`
			}{perm.Read, perm.Write, perm.Delete, perm.Admin})
			if err != nil {
				return fmt.Errorf(
					"failed to encode permission for %s:%s into JSON format: %v",
					userID, namespace, err)
			}

		default:
			fmt.Printf("Read: %v\n", perm.Read)
			fmt.Printf("Write: %v\n", perm.Write)
			fmt.Printf("Delete: %v\n", perm.Delete)
			fmt.Printf("Admin: %v\n", perm.Admin)
		}

		return nil
	},
}

func init() {
	namespaceCmd.AddCommand(
		namespaceCreateCmd,
		namespaceDeleteCmd,
		namespacePermissionCmd,
	)

	namespacePermissionCmd.AddCommand(
		namespaceSetPermissionCmd,
		namespaceGetPermissionCmd,
	)

	namespaceSetPermissionCmd.Flags().BoolVarP(
		&namespaceSetPermissionCfg.Read, "read", "r", false,
		"Set read permissions for the given user and namespace.")
	namespaceSetPermissionCmd.Flags().BoolVarP(
		&namespaceSetPermissionCfg.Write, "write", "w", false,
		"Set write permissions for the given user and namespace.")
	namespaceSetPermissionCmd.Flags().BoolVarP(
		&namespaceSetPermissionCfg.Delete, "delete", "d", false,
		"Set delete permissions for the given user and namespace.")
	namespaceSetPermissionCmd.Flags().BoolVarP(
		&namespaceSetPermissionCfg.Admin, "admin", "a", false,
		"Set admin permissions for the given user and namespace.")

	namespaceGetPermissionCmd.Flags().BoolVar(
		&namespaceGetPermissionCfg.JSONFormat, "json", false,
		"Print the permissions in JSON format instead of a custom human readable format.")
	namespaceGetPermissionCmd.Flags().BoolVar(
		&namespaceGetPermissionCfg.JSONPrettyFormat, "json-pretty", false,
		"Print the permissions in prettified JSON format instead of a custom human readable format.")
}
