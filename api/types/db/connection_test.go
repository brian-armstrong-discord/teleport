/*

 Copyright 2022 Gravitational, Inc.

 Licensed under the Apache License, Version 2.0 (the "License");
 you may not use this file except in compliance with the License.
 You may obtain a copy of the License at

     http://www.apache.org/licenses/LICENSE-2.0

 Unless required by applicable law or agreed to in writing, software
 distributed under the License is distributed on an "AS IS" BASIS,
 WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 See the License for the specific language governing permissions and
 limitations under the License.


*/

package db

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func Test_parseRedisURI(t *testing.T) {
	tests := []struct {
		name   string
		uri    string
		want   *RedisConnectionOptions
		errStr string
	}{
		{
			name: "correct URI",
			uri:  "redis://localhost:6379",
			want: &RedisConnectionOptions{
				Mode: Standalone,
				Host: "localhost",
				Port: "6379",
			},
			errStr: "",
		},
		{
			name: "correct host:Port",
			uri:  "localhost:6379",
			want: &RedisConnectionOptions{
				Mode: Standalone,
				Host: "localhost",
				Port: "6379",
			},
			errStr: "",
		},
		{
			name: "rediss schema is accepted",
			uri:  "rediss://localhost:6379",
			want: &RedisConnectionOptions{
				Mode: Standalone,
				Host: "localhost",
				Port: "6379",
			},
			errStr: "",
		},
		{
			name: "IP Address passes",
			uri:  "rediss://1.2.3.4:6379",
			want: &RedisConnectionOptions{
				Mode: Standalone,
				Host: "1.2.3.4",
				Port: "6379",
			},
			errStr: "",
		},
		{
			name: "single instance explicit",
			uri:  "redis://localhost:6379?mode=standalone",
			want: &RedisConnectionOptions{
				Mode: Standalone,
				Host: "localhost",
				Port: "6379",
			},
			errStr: "",
		},
		{
			name: "cluster enabled",
			uri:  "redis://localhost:6379?mode=cluster",
			want: &RedisConnectionOptions{
				Mode: Cluster,
				Host: "localhost",
				Port: "6379",
			},
			errStr: "",
		},
		{
			name:   "invalid connection Mode",
			uri:    "redis://localhost:6379?mode=foo",
			want:   nil,
			errStr: "incorrect connection Mode",
		},
		{
			name:   "invalid connection string",
			uri:    "localhost:6379?mode=foo",
			want:   nil,
			errStr: "failed to parse Redis URL",
		},
		{
			name: "only Address default Port",
			uri:  "localhost",
			want: &RedisConnectionOptions{
				Mode: Standalone,
				Host: "localhost",
				Port: "6379",
			},
			errStr: "",
		},
		{
			name: "default Port",
			uri:  "redis://localhost",
			want: &RedisConnectionOptions{
				Mode: Standalone,
				Host: "localhost",
				Port: "6379",
			},
			errStr: "",
		},
		{
			name:   "incorrect URI schema is rejected",
			uri:    "http://localhost",
			want:   nil,
			errStr: "invalid Redis URI scheme",
		},
		{
			name:   "empty Address",
			uri:    "",
			want:   nil,
			errStr: "Address is empty",
		},
	}
	for _, tt := range tests {
		tt := tt

		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got, err := ParseRedisAddress(tt.uri)
			if err != nil {
				if tt.errStr == "" {
					require.FailNow(t, "unexpected error: %v", err)
					return
				}
				require.Contains(t, err.Error(), tt.errStr)
				return
			}

			require.Equal(t, tt.want, got)
		})
	}
}