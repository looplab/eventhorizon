// Copyright (c) 2015 - Max Ekman <max@looplab.se>
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

package mongodb

import (
	"context"
	"os"
	"testing"

	eh "github.com/looplab/eventhorizon"
	"github.com/looplab/eventhorizon/mocks"
	"github.com/looplab/eventhorizon/projectordriver/testutil"
)

func TestProjectorDriver(t *testing.T) {
	// Support Wercker testing with MongoDB.
	host := os.Getenv("MONGO_PORT_27017_TCP_ADDR")
	port := os.Getenv("MONGO_PORT_27017_TCP_PORT")

	url := "localhost"
	if host != "" && port != "" {
		url = host + ":" + port
	}

	driver, err := NewProjectorDriver(url, "test", "mocks.Model")
	if err != nil {
		t.Error("there should be no error:", err)
	}
	if driver == nil {
		t.Error("there should be a driver")
	}
	defer driver.Close()
	driver.SetModelFactory(func() interface{} {
		return &mocks.Model{}
	})

	ctx := eh.NewContextWithNamespace(context.Background(), "ns")

	defer func() {
		t.Log("clearing db")
		if err = driver.Clear(context.Background()); err != nil {
			t.Fatal("there should be no error:", err)
		}
		if err = driver.Clear(ctx); err != nil {
			t.Fatal("there should be no error:", err)
		}
	}()

	// Run the actual test suite.

	t.Log("driver with default namespace")
	testutil.ProjectorDriverCommonTests(t, context.Background(), driver)

	t.Log("driver with other namespace")
	testutil.ProjectorDriverCommonTests(t, ctx, driver)
}
