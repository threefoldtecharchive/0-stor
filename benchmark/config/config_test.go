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

package config

import (
	"io/ioutil"
	"math"
	"testing"

	"github.com/zero-os/0-stor/client"

	"github.com/stretchr/testify/require"
)

const (
	validFile              = "../../fixtures/benchmark/testconfigs/valid_conf.yaml"
	emptyFile              = "../../fixtures/benchmark/testconfigs/empty_conf.yaml"
	invalidDurOpsConfFile  = "../../fixtures/benchmark/testconfigs/invalid_dur_ops_conf.yaml"
	invalidKeySizeConfFile = "../../fixtures/benchmark/testconfigs/invalid_keysize_conf.yaml"
)

func TestClientConfig(t *testing.T) {
	require := require.New(t)

	yamlFile, err := ioutil.ReadFile(validFile)
	require.NoError(err)

	_, err = UnmarshalYAML(yamlFile)
	require.NoError(err)
}

func TestInvalidClientConfig(t *testing.T) {
	require := require.New(t)

	// empty config
	yamlFile, err := ioutil.ReadFile(emptyFile)
	require.NoError(err)

	_, err = UnmarshalYAML(yamlFile)
	require.Error(err)

	// invalid ops/duration
	yamlFile, err = ioutil.ReadFile(invalidDurOpsConfFile)
	require.NoError(err)

	_, err = UnmarshalYAML(yamlFile)
	require.Error(err)

	// invalid keysize
	yamlFile, err = ioutil.ReadFile(invalidKeySizeConfFile)
	require.NoError(err)

	_, err = UnmarshalYAML(yamlFile)
	require.Error(err)
}

func TestSetupClientConfig(t *testing.T) {
	require := require.New(t)
	c := client.Config{}

	SetupClientConfig(&c)
	require.NotEmpty(c.Namespace, "Namespace should be set")

	const testNamespace = "test_namespace"
	c = client.Config{
		Namespace: testNamespace,
	}

	SetupClientConfig(&c)
	require.Equal(testNamespace, c.Namespace)
}

func TestMarshalBencherMethod(t *testing.T) {
	require := require.New(t)

	testCases := []struct {
		Type     BencherMethod
		Expected string
	}{
		{BencherRead, "read"},
		{BencherWrite, "write"},
		{math.MaxUint8, ""},
	}
	for _, tc := range testCases {
		b, err := tc.Type.MarshalText()
		if tc.Expected == "" {
			require.Error(err)
			require.Nil(b)
		} else {
			require.NoError(err)
			require.Equal(tc.Expected, string(b))
		}
	}
}

func TestUnmarshalBencherMethod(t *testing.T) {
	require := require.New(t)

	testCases := []struct {
		String   string
		Expected BencherMethod
		Err      bool
	}{
		{"read", BencherRead, false},
		{"Read", BencherRead, false},
		{"READ", BencherRead, false},
		{"write", BencherWrite, false},
		{"Write", BencherWrite, false},
		{"WRITE", BencherWrite, false},
		{"foo", math.MaxUint8, true},
	}
	for _, tc := range testCases {
		var bm BencherMethod
		err := bm.UnmarshalText([]byte(tc.String))
		if tc.Err {
			require.Error(err)
		} else {
			require.NoError(err)
			require.Equal(tc.Expected, bm)
		}
	}
}
