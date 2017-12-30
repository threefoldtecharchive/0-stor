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

package jwt

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestMethodString(t *testing.T) {
	cases := []struct {
		Value    Method
		Expected string
	}{
		{MethodRead, "read"},
		{MethodWrite, "write"},
		{MethodDelete, "delete"},
		{MethodAdmin, "admin"},
		{Expected: ""},
		{42, ""},
	}

	for _, c := range cases {
		assert.Equalf(t, c.Expected, c.Value.String(), "Method: %d", c.Value)
		assert.Equalf(t, c.Expected, fmt.Sprint(c.Value), "Method: %d", c.Value)
		assert.Equalf(t, c.Expected, fmt.Sprintf("%s", c.Value), "Method: %d", c.Value)
	}
}
