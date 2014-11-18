// Copyright (c) 2014 - Max Persson <max@looplab.se>
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package testing

import (
	"gopkg.in/check.v1"
)

// Contains is a checker for gocheck that asserts if an item is in a slice.
var Contains = &containsChecker{
	&check.CheckerInfo{
		Name: "Contains",
		Params: []string{
			"Container",
			"Expected to contain",
		},
	},
}

type containsChecker struct {
	*check.CheckerInfo
}

func (c *containsChecker) Check(params []interface{}, names []string) (result bool, error string) {
	slice, ok := params[0].([]interface{})
	if !ok {
		return false, "First parameter is not a []interface{}"
	}
	value, ok := params[1].(interface{})
	if !ok {
		return false, "Second parameter is not an int"
	}
	for _, v := range slice {
		if v == value {
			return true, ""
		}
	}
	return false, ""
}
