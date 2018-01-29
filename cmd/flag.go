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
	"crypto/tls"
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

// ListenAddress is a string representing either a TCP addr (a host and a port),
// or a Unix socket (file path), which can be used as a flag for a Cobra command.
type ListenAddress struct {
	addr  string
	proto netProto
}

const (
	defaultTCPListenAddress = ":8080"
)

type netProto uint8

const (
	netProtoTCP netProto = iota
	netProtoUnix
)

// String implements spf13/pflag.Value.String
func (b *ListenAddress) String() string {
	switch b.proto {
	case netProtoUnix:
		return b.addr
	default:
		if len(b.addr) == 0 {
			return defaultTCPListenAddress
		}
		return b.addr
	}
}

// Set implements spf13/pflag.Value.Set
func (b *ListenAddress) Set(str string) error {
	if len(str) == 0 {
		b.addr, b.proto = defaultTCPListenAddress, netProtoTCP
		return nil
	}

	if i := strings.IndexAny(str, "/:"); i == -1 || str[i] == '/' {
		b.addr, b.proto = str, netProtoUnix
		return nil
	}

	host, _, err := net.SplitHostPort(str)
	if err != nil {
		return err
	}

	if host != "" {
		if ip := net.ParseIP(host); ip == nil {
			return errors.New("host not valid")
		}
	}

	b.addr, b.proto = str, netProtoTCP
	return nil
}

// Description prints the flag description for this flag
func (b *ListenAddress) Description() string {
	return "Bind the server to the given unix socket path or tcp address." +
		" Format has to be either host:port, with host optional, or a valid unix (socket) path."
}

// Type implements spf13/pflag.Value.Type
func (b *ListenAddress) Type() string {
	return "listenAddress"
}

// NetworkProtocol returns the network protocol to be used for the set ListenAddress.
func (b *ListenAddress) NetworkProtocol() string {
	switch b.proto {
	case netProtoTCP:
		return "tcp"
	case netProtoUnix:
		return "unix"
	default:
		return ""
	}
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
			return nil
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

// TLSVersion defines a TLS Version,
// usable to restrict the possible TLS Versions.
type TLSVersion uint8

const (
	// TLSVersion12 defines TLS version 1.2,
	// and is also the current default TLS Version.
	TLSVersion12 TLSVersion = iota
	// TLSVersion11 defines TLS version 1.1
	TLSVersion11
	// TLSVersion10 defines TLS version 1.0,
	// but should not be used, unless you have no other option.
	TLSVersion10

	_MaxTLSVersion = TLSVersion10
)

var (
	// important that this order stays in sync with
	// the order of constants definitions from above!
	_TLSVersionStrings = []string{
		"TLS12",
		"TLS11",
		"TLS10",
	}
	_TlsVersionValues = []uint16{
		tls.VersionTLS12,
		tls.VersionTLS11,
		tls.VersionTLS10,
	}
)

// String implements spf13/pflag.Value.String
func (v TLSVersion) String() string {
	if v > _MaxTLSVersion {
		return ""
	}
	return _TLSVersionStrings[v]
}

// Set implements spf13/pflag.Value.Set
func (v *TLSVersion) Set(str string) error {
	str = strings.ToUpper(str)
	for index, verStr := range _TLSVersionStrings {
		if verStr == str {
			*v = TLSVersion(index)
			return nil
		}
	}
	return fmt.Errorf("TLS version %s not recognized or supported", str)
}

// Type implements spf13/pflag.Value.Type
func (v TLSVersion) Type() string {
	return "TLSVersion"
}

// Description prints the flag description for this flag
func (v TLSVersion) Description(desc string) string {
	// print with disabled/invalid default
	return fmt.Sprintf("%s, one of [%s].",
		desc, strings.Join(_TLSVersionStrings, ", "))
}

// VersionTLS returns the tls.VersionTLS value,
// which corresponds to this TLSVersion.
func (v TLSVersion) VersionTLS() uint16 {
	if v > _MaxTLSVersion {
		panic(fmt.Sprintf("invalid TLS Version: %d", v))
	}
	return _TlsVersionValues[v]
}
