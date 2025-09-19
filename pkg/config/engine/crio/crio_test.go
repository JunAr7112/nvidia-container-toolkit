/**
# Copyright 2024 NVIDIA CORPORATION
#
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
#     http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.
**/

package crio

import (
	"testing"

	testlog "github.com/sirupsen/logrus/hooks/test"
	"github.com/stretchr/testify/require"

	"github.com/NVIDIA/nvidia-container-toolkit/pkg/config/toml"
)

func TestAddRuntime(t *testing.T) {
	logger, _ := testlog.NewNullLogger()
	testCases := []struct {
		description    string
		config         string
		setAsDefault   bool
		expectedConfig string
		expectedError  error
	}{
		{
			description: "empty config not default runtime",
			expectedConfig: `
			[crio]
			[crio.runtime.runtimes.test]
			runtime_path = "/usr/bin/test"
			runtime_type = "oci"
			`,
			expectedError: nil,
		},
		{
			description:  "empty config, set as default runtime",
			setAsDefault: true,
			expectedConfig: `
			[crio]
			[crio.runtime]
			default_runtime = "test"
			[crio.runtime.runtimes.test]
			runtime_path = "/usr/bin/test"
			runtime_type = "oci"
			`,
			expectedError: nil,
		},
		{
			description: "options from runc are imported",
			config: `
			[crio]
			[crio.runtime.runtimes.runc]
			runtime_path = "/usr/bin/runc"
			runtime_type = "runcoci"
			runc_option = "option"
			`,
			expectedConfig: `
			[crio]
			[crio.runtime.runtimes.test]
			runtime_path = "/usr/bin/test"
			runtime_type = "oci"
			runc_option = "option"
			`,
		},
		{
			description: "options from default runtime are imported",
			config: `
			[crio]
			[crio.runtime]
			default_runtime = "default"
			[crio.runtime.runtimes.default]
			runtime_path = "/usr/bin/default"
			runtime_type = "defaultoci"
			default_option = "option"
			`,
			expectedConfig: `
			[crio]
			[crio.runtime]
			[crio.runtime.runtimes.test]
			runtime_path = "/usr/bin/test"
			runtime_type = "oci"
			default_option = "option"
			`,
		},
		{
			description: "options from the default runtime take precedence over runc",
			config: `
			[crio]
			[crio.runtime]
			default_runtime = "default"
			[crio.runtime.runtimes.default]
			runtime_path = "/usr/bin/default"
			runtime_type = "defaultoci"
			default_option = "option"
			[crio.runtime.runtimes.runc]
			runtime_path = "/usr/bin/runc"
			runtime_type = "runcoci"
			runc_option = "option"
			`,
			expectedConfig: `
			[crio]
			[crio.runtime]
			[crio.runtime.runtimes.test]
			runtime_path = "/usr/bin/test"
			runtime_type = "oci"
			default_option = "option"
			`,
		},
		{
			description:  "runtime already exists in config, default runtime",
			setAsDefault: true,
			config: `
			[crio]
			[crio.runtime]
			default_runtime = "test"
			[crio.runtime.runtimes.test]
			runtime_path = "/usr/bin/test"
			runtime_type = "oci"
			`,
			expectedConfig: `
			[crio]
			[crio.runtime]
			default_runtime = "test"
			[crio.runtime.runtimes.test]
			runtime_path = "/usr/bin/test"
			runtime_type = "oci"
			`,
			expectedError: nil,
		},
		{
			description:  "runtime already exists in config, not default runtime",
			setAsDefault: false,
			config: `
			[crio]
			[crio.runtime]
			default_runtime = "test"
			[crio.runtime.runtimes.test]
			runtime_path = "/usr/bin/test"
			runtime_type = "oci"
			`,
			expectedConfig: `
			[crio]
			[crio.runtime]
			[crio.runtime.runtimes.test]
			runtime_path = "/usr/bin/test"
			runtime_type = "oci"
			`,
			expectedError: nil,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.description, func(t *testing.T) {
			expectedConfig, err := toml.Load(tc.expectedConfig)
			require.NoError(t, err)

			c, err := New(
				WithLogger(logger),
				WithConfigSource(toml.FromString(tc.config)),
			)
			require.NoError(t, err)

			err = c.AddRuntime("test", "/usr/bin/test", tc.setAsDefault)
			require.NoError(t, err)

			require.EqualValues(t, expectedConfig.String(), c.String())
		})
	}
}

func TestGetRuntimeConfig(t *testing.T) {
	logger, _ := testlog.NewNullLogger()
	config := `
[crio.image]
signature_policy = "/etc/crio/policy.json"

[crio.runtime]
default_runtime = "crun"

[crio.runtime.runtimes.crun]
runtime_path = "/usr/libexec/crio/crun"
runtime_root = "/run/crun"
monitor_path = "/usr/libexec/crio/conmon"
allowed_annotations = [
    "io.containers.trace-syscall",
]

[crio.runtime.runtimes.runc]
runtime_path = "/usr/libexec/crio/runc"
runtime_root = "/run/runc"
monitor_path = "/usr/libexec/crio/conmon"
`
	testCases := []struct {
		description   string
		runtime       string
		expected      string
		expectedError error
	}{
		{
			description:   "valid runtime config, existing runtime",
			runtime:       "crun",
			expected:      "/usr/libexec/crio/crun",
			expectedError: nil,
		},
		{
			description:   "valid runtime config, non-existing runtime",
			runtime:       "some-other-runtime",
			expected:      "",
			expectedError: nil,
		},
	}
	for _, tc := range testCases {
		t.Run(tc.description, func(t *testing.T) {
			c, err := New(
				WithLogger(logger),
				WithConfigSource(toml.FromString(config)),
			)
			require.NoError(t, err)

			rc, err := c.GetRuntimeConfig(tc.runtime)
			require.Equal(t, tc.expectedError, err)
			require.Equal(t, tc.expected, rc.GetBinaryPath())
		})
	}
}
