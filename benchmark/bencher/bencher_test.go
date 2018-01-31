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

package bencher

import (
	"math"
	"testing"

	"github.com/zero-os/0-stor/benchmark/config"

	"github.com/stretchr/testify/require"
)

func TestBencherRuns(t *testing.T) {
	require := require.New(t)

	// setup test servers
	servers, cleanupZstor := newTestZstorServers(t, 4)
	defer cleanupZstor()

	shards := make([]string, len(servers))
	for i, server := range servers {
		shards[i] = server.Address()
	}

	clientConfig := newDefaultZstorConfig(shards, nil, 64)

	const (
		runs   = 100
		testID = "testScenario"
	)
	sc := config.Scenario{
		ZstorConf: clientConfig,
		BenchConf: config.BenchmarkConfig{
			Method:     config.BencherRead,
			Operations: runs,
			KeySize:    5,
			ValueSize:  25,
		},
	}

	// run read benchmark
	bencher, err := NewBencher(testID, &sc)
	require.NoError(err)

	r, err := bencher.RunBenchmark()
	require.NoError(err)
	require.Equal(int64(runs), r.Count)

	// run write benchmark
	sc.BenchConf.Method = config.BencherWrite
	bencher, err = NewBencher(testID, &sc)
	require.NoError(err)

	r, err = bencher.RunBenchmark()
	require.NoError(err)
	require.Equal(int64(runs), r.Count)
}

func TestBencherDuration(t *testing.T) {
	require := require.New(t)

	// setup test servers
	servers, cleanupZstor := newTestZstorServers(t, 4)
	defer cleanupZstor()

	shards := make([]string, len(servers))
	for i, server := range servers {
		shards[i] = server.Address()
	}

	clientConfig := newDefaultZstorConfig(shards, nil, 64)

	const (
		duration = 2
		testID   = "testScenario"
	)
	sc := config.Scenario{
		ZstorConf: clientConfig,
		BenchConf: config.BenchmarkConfig{
			Method:    config.BencherRead,
			Duration:  duration,
			KeySize:   5,
			ValueSize: 25,
			Output:    "per_second",
		},
	}

	// run read  benchmark
	bencher, err := NewBencher(testID, &sc)
	require.NoError(err)

	r, err := bencher.RunBenchmark()
	require.NoError(err)

	// check if it ran for about requested duration
	runDur := r.Duration.Seconds()
	require.Equal(float64(duration), math.Floor(runDur),
		"rounded run duration should be equal to the requested duration")

	// run write  benchmark
	sc.BenchConf.Method = config.BencherWrite
	bencher, err = NewBencher(testID, &sc)
	require.NoError(err)

	r, err = bencher.RunBenchmark()
	require.NoError(err)

	// check if it ran for about requested duration
	runDur = r.Duration.Seconds()
	require.Equal(float64(duration), math.Floor(runDur),
		"rounded run duration should be equal to the requested duration")
}
