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

package processing

import (
	"fmt"
	"strings"
)

// CompressionMode represents a compression mode.
//
// A compression mode is a hint to the compression type's constructor,
// it is not guaranteed that the returned compressor is as ideal as you'll hope,
// the implementation should however try to fit the requested mode
// to a possible configuration which respects the particalar mode as much as possible.
type CompressionMode uint8

const (
	// CompressionModeDisabled is the enum constant which identifies
	// the disabled compression mode.
	// Meaning we do not want any compression type at all.
	CompressionModeDisabled CompressionMode = iota
	// CompressionModeDefault is the enum constant which identifies
	// the default compression mode. What this means exactly
	// is up to the compression algorithm to define.
	CompressionModeDefault
	// CompressionModeBestSpeed is the enum constant which identifies
	// the request to use the compression with a configuration,
	// which aims for the best possible speed, using that algorithm.
	//
	// What this exactly means is up to the compression type,
	// and it is entirely free to ignore this compression mode all together,
	// if it doesn't make sense for that type in particular.
	CompressionModeBestSpeed
	// CompressionModeBestCompression is the enum constant which identifies
	// the request to use the compression with a configuration,
	// which aims for the best possible compression (ratio), using that algorithm.
	//
	// What this exactly means is up to the compression type,
	// and it is entirely free to ignore this compression mode all together,
	// if it doesn't make sense for that type in particular.
	CompressionModeBestCompression
)

// String implements Stringer.String
func (cm CompressionMode) String() string {
	return _CompressionModeValueToStringMapping[cm]
}

var (
	_CompressionModeValueToStringMapping = map[CompressionMode]string{
		CompressionModeDisabled:        "",
		CompressionModeDefault:         _CompressionModeStrings[:7],
		CompressionModeBestSpeed:       _CompressionModeStrings[7:17],
		CompressionModeBestCompression: _CompressionModeStrings[17:],
	}
	_CompressionModeStringToValueMapping = map[string]CompressionMode{
		"":                            CompressionModeDisabled,
		_CompressionModeStrings[:7]:   CompressionModeDefault,
		_CompressionModeStrings[7:17]: CompressionModeBestSpeed,
		_CompressionModeStrings[17:]:  CompressionModeBestCompression,
	}
)

const _CompressionModeStrings = "defaultbest_speedbest_compression"

// MarshalText implements encoding.TextMarshaler.MarshalText
func (cm CompressionMode) MarshalText() ([]byte, error) {
	return []byte(cm.String()), nil
}

// UnmarshalText implements encoding.TextUnmarshaler.UnmarshalText
func (cm *CompressionMode) UnmarshalText(text []byte) error {
	var ok bool
	*cm, ok = _CompressionModeStringToValueMapping[strings.ToLower(string(text))]
	if !ok {
		return fmt.Errorf("'%s' is not a valid CompressionMode string", text)
	}
	return nil
}
