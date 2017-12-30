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
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestVersion_PrintAndLog(t *testing.T) {
	// no build commit, or date
	PrintVersion()
	LogVersion()

	// set build date allone
	BuildDate = "1st of January"
	PrintVersion()
	LogVersion()

	// set commit hash allone
	BuildDate, CommitHash = "", "abcdefghijk"
	PrintVersion()
	LogVersion()

	// set both commit hash and build date
	BuildDate = "31st of December"
	PrintVersion()
	LogVersion()
}

func TestVersionString(t *testing.T) {
	testCases := []struct {
		version  Version
		expected string
	}{
		{NewVersion(0, 0, 0, nil), "0.0.0"},
		{NewVersion(0, 0, 1, nil), "0.0.1"},
		{NewVersion(0, 1, 0, nil), "0.1.0"},
		{NewVersion(1, 0, 0, nil), "1.0.0"},
		{NewVersion(1, 0, 0, versionLabel("beta-1")), "1.0.0-beta-1"},
		{NewVersion(1, 1, 0, nil), "1.1.0"},
		{NewVersion(1, 1, 0, versionLabel("abcdefgh")), "1.1.0-abcdefgh"},
		{NewVersion(1, 1, 0, nil), "1.1.0"},
		{NewVersion(1, 2, 3, nil), "1.2.3"},
		{NewVersion(1, 2, 3, versionLabel("alpha-8")), "1.2.3-alpha-8"},
		{NewVersion(4, 2, 0, nil), "4.2.0"},
	}

	for _, testCase := range testCases {
		str := testCase.version.String()
		assert.Equal(t, testCase.expected, str)
	}
}

func TestVersionTextMarshalUnmarshal(t *testing.T) {
	testCases := []struct {
		version  Version
		expected string
	}{
		{NewVersion(0, 0, 0, nil), "0.0.0"},
		{NewVersion(0, 0, 1, nil), "0.0.1"},
		{NewVersion(0, 1, 0, nil), "0.1.0"},
		{NewVersion(1, 0, 0, nil), "1.0.0"},
		{NewVersion(1, 0, 0, versionLabel("beta-1")), "1.0.0-beta-1"},
		{NewVersion(1, 1, 0, nil), "1.1.0"},
		{NewVersion(1, 1, 0, versionLabel("abcdefgh")), "1.1.0-abcdefgh"},
		{NewVersion(1, 1, 0, nil), "1.1.0"},
		{NewVersion(1, 2, 3, nil), "1.2.3"},
		{NewVersion(1, 2, 3, versionLabel("alpha-8")), "1.2.3-alpha-8"},
		{NewVersion(4, 2, 0, nil), "4.2.0"},
	}

	for _, testCase := range testCases {
		// marshal
		text, err := testCase.version.MarshalText()
		if testCase.expected == "" {
			// if expected is empty, it means we're expecting an error
			if assert.Error(t, err) {
				assert.Empty(t, text)
			}
			continue
		}

		// we're not expecting an error, so we should the correct string and no error
		if !assert.NoError(t, err) || !assert.Equal(t, testCase.expected, string(text)) {
			continue
		}

		// unmarshal, marshalled text we received
		var v Version
		v.UnmarshalText(text)
		assert.Equal(t, testCase.version, v)
	}
}

func TestUnmarshalVersionErrors(t *testing.T) {
	bad := []string{
		"1.1-alpha",             //no patch number
		"abcd",                  //rubbish
		"1.1.1.alpha-2",         //label separated by . instead of -
		"123671.0.0",            //numbers out of rage of uint8,
		"0.1234.0",              //numbers out of rage of uint8,
		"0.0.12345",             //numbers out of rage of uint8,
		"0.0.0-very-long-label", //label is longer than 8 char
	}

	var v Version
	for _, s := range bad {
		err := v.UnmarshalText([]byte(s))
		if assert.Error(t, err) {
			assert.Equal(t, NilVersion, v)
		}
	}
}

func TestVersionUint32(t *testing.T) {
	testCases := []struct {
		version  Version
		expected uint32
	}{
		{NewVersion(0, 0, 0, nil), 0x0},
		{NewVersion(0, 0, 1, nil), 0x1},
		{NewVersion(0, 0, 1, versionLabel("foo")), 0x1},
		{NewVersion(0, 0, 42, versionLabel("beta-1")), 0x2A},
		{NewVersion(0, 1, 1, nil), 0x101},
		{NewVersion(0, 1, 1, versionLabel("beta-1")), 0x101},
		{NewVersion(1, 1, 1, nil), 0x10101},
		{NewVersion(1, 1, 1, versionLabel("beta-1")), 0x10101},
		{NewVersion(255, 15, 1, nil), 0xFF0F01},
		{NewVersion(255, 15, 1, versionLabel("beta-1")), 0xFF0F01},
		{NewVersion(255, 17, 255, nil), 0xFF11FF},
		{NewVersion(255, 17, 255, versionLabel("beta-1")), 0xFF11FF},
	}

	for _, testCase := range testCases {
		vn := testCase.version.UInt32()
		// test if our uint32 value is correct
		if !assert.Equal(t, testCase.expected, vn) {
			continue
		}
		v := VersionFromUInt32(vn)

		// remove label, as we lose that information here
		expected := testCase.version
		expected.Label = nil

		// now test if we can go back to the correct version (minus the label)
		assert.Equal(t, expected, v)
	}
}

func TestVersionCompare(t *testing.T) {
	testCases := []struct {
		verA, verB Version
		expected   int
	}{
		// equal
		{NewVersion(0, 0, 0, nil), NewVersion(0, 0, 0, nil), 0},
		{NewVersion(0, 0, 1, nil), NewVersion(0, 0, 1, nil), 0},
		{NewVersion(0, 1, 0, nil), NewVersion(0, 1, 0, nil), 0},
		{NewVersion(0, 1, 0, versionLabel("foo")), NewVersion(0, 1, 0, nil), 0},
		{NewVersion(0, 1, 0, nil), NewVersion(0, 1, 0, versionLabel("foo")), 0},
		{NewVersion(0, 1, 0, versionLabel("foo")), NewVersion(0, 1, 0, versionLabel("foo")), 0},
		{NewVersion(1, 0, 0, nil), NewVersion(1, 0, 0, nil), 0},
		{NewVersion(1, 1, 0, nil), NewVersion(1, 1, 0, nil), 0},
		{NewVersion(3, 2, 1, nil), NewVersion(3, 2, 1, nil), 0},
		// different
		{NewVersion(2, 0, 0, nil), NewVersion(1, 12, 19, nil), 1},
		{NewVersion(1, 0, 0, nil), NewVersion(0, 1, 1, nil), 1},
		{NewVersion(1, 0, 1, nil), NewVersion(1, 0, 0, nil), 1},
		{NewVersion(1, 1, 1, nil), NewVersion(1, 1, 0, nil), 1},
		{NewVersion(0, 1, 0, nil), NewVersion(0, 0, 1, nil), 1},
		{NewVersion(0, 1, 1, nil), NewVersion(0, 1, 0, nil), 1},
		{NewVersion(0, 0, 1, nil), NewVersion(0, 0, 0, nil), 1},
		{NewVersion(1, 12, 19, nil), NewVersion(2, 0, 0, nil), -1},
		{NewVersion(0, 1, 1, nil), NewVersion(1, 0, 0, nil), -1},
		{NewVersion(1, 0, 0, nil), NewVersion(1, 0, 1, nil), -1},
		{NewVersion(1, 1, 0, nil), NewVersion(1, 1, 1, nil), -1},
		{NewVersion(0, 0, 1, nil), NewVersion(0, 1, 0, nil), -1},
		{NewVersion(0, 1, 0, nil), NewVersion(0, 1, 1, nil), -1},
		{NewVersion(0, 0, 0, nil), NewVersion(0, 0, 1, nil), -1},
	}

	for _, testCase := range testCases {
		result := testCase.verA.Compare(testCase.verB)
		assert.Equalf(t, testCase.expected, result, "%s v %s", testCase.verA, testCase.verB)
	}
}

func TestVersionFromString(t *testing.T) {
	versions := []Version{
		NewVersion(1, 2, 0, nil),
		NewVersion(2, 1, 1, versionLabel("test")),
		NewVersion(2, 0, 0, versionLabel("test")),
	}

	for _, v := range versions {
		pv, err := VersionFromString(v.String())
		if err != nil {
			t.Fatal(err)
		}

		if ok := assert.Zero(t, v.Compare(pv)); !ok {
			t.Error()
		}

		if ok := assert.Equal(t, pv.Label, v.Label); !ok {
			t.Error()
		}
	}

	//default version
	if dv, err := VersionFromString(""); err == nil {
		if ok := assert.Zero(t, DefaultVersion.Compare(dv)); !ok {
			t.Fatal()
		}
	} else {
		t.Fatal(err)
	}

	//faulty version numbers
	bad := []string{
		"1.1-alpha",             //no patch number
		"abcd",                  //rubbish
		"1.1.1.alpha-2",         //label separated by . instead of -
		"123671.0.0",            //numbers out of rage of uint8,
		"0.1234.0",              //numbers out of rage of uint8,
		"0.0.12345",             //numbers out of rage of uint8,
		"0.0.0-very-long-label", //label is longer than 8 char
	}

	for _, s := range bad {
		_, err := VersionFromString(s)
		if ok := assert.Error(t, err); !ok {
			t.Error()
		}
	}
}
