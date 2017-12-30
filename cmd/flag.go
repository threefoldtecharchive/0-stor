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

package cmd

import (
	"errors"
	"fmt"
	"net"
	"strings"
)

// Strings is a slice of strings which can be used as
// a flag for a Cobra command.
type Strings []string

// String implements spf13/pflag.Value.String
func (s Strings) String() string {
	if len(s) == 0 {
		return ""
	}
	return strings.Join([]string(s), ",")
}

// Strings returns this Slice as a []string value.
func (s Strings) Strings() []string {
	return s
}

// Set implements spf13/pflag.Value.Set
func (s *Strings) Set(str string) error {
	if len(str) == 0 {
		*s = nil
	} else {
		*s = strings.Split(str, ",")
	}
	return nil
}

// Type implements spf13/pflag.Value.Type
func (s Strings) Type() string {
	return "strings"
}

// ListenAddress is a string representing a host and a port
// which can be used as a flag for a Cobra command.
type ListenAddress string

const defaultListenAddress = ":8080"

// String implements spf13/pflag.Value.String
func (b *ListenAddress) String() string {
	if len(*b) == 0 {
		return defaultListenAddress
	}
	return string(*b)
}

// Set implements spf13/pflag.Value.Set
func (b *ListenAddress) Set(str string) error {
	host, _, err := net.SplitHostPort(str)
	if err != nil {
		return err
	}

	if host != "" {
		if ip := net.ParseIP(host); ip == nil {
			return errors.New("host not valid")
		}
	}

	*b = ListenAddress(str)
	return nil
}

// Description prints the flag description for this flag
func (b *ListenAddress) Description() string {
	return "Bind the server to the given host and port." +
		" Format has to be host:port, with host optional"
}

// Type implements spf13/pflag.Value.Type
func (b *ListenAddress) Type() string {
	return "listenAddress"
}

//ProfileMode is a string representing a profiling mode
//which can be used as a flag for Cobra command
type ProfileMode uint8

const (
	//ProfileModeDisabled disables profiling
	ProfileModeDisabled ProfileMode = iota
	//ProfileModeCPU enables cpu profiling
	ProfileModeCPU
	//ProfileModeMem enables memory profiling
	ProfileModeMem
	//ProfileModeBlock enables blocking profiling
	ProfileModeBlock
	//ProfileModeTrace enables trace profiling
	ProfileModeTrace

	// constants just for our conversion functions that might need to know about this
	_MinProfileMode = ProfileModeCPU
	_MaxProfileMode = ProfileModeTrace
)

var (
	// important that this order stays in sync with
	// the order of constants definitions from above!
	_ProfileModeStrings = []string{
		"",
		"cpu",
		"mem",
		"block",
		"trace",
	}
)

// String implements spf13/pflag.Value.String
func (p ProfileMode) String() string {
	if p > _MaxProfileMode {
		return ""
	}
	return _ProfileModeStrings[p]
}

// Set implements spf13/pflag.Value.Set
func (p *ProfileMode) Set(str string) error {
	for index, modeStr := range _ProfileModeStrings {
		if modeStr == str {
			*p = ProfileMode(index)
		}
	}
	return fmt.Errorf("profile mode %s not recognized or supported", str)
}

// Type implements spf13/pflag.Value.Type
func (p ProfileMode) Type() string {
	return "profileMode"
}

// Description prints the flag description for this flag
func (p ProfileMode) Description() string {
	profileModes := _ProfileModeStrings[_MinProfileMode : _MinProfileMode+_MaxProfileMode]
	// print with acceptable default
	if str := p.String(); str != "" {
		return fmt.Sprintf("Enable profiling mode, one of [%s] (default %s)",
			strings.Join(profileModes, ", "), str)
	}
	// print with disabled/invalid default
	return fmt.Sprintf("Enable profiling mode, one of [%s]", strings.Join(profileModes, ", "))
}
