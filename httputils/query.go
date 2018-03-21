// Copyright (c) 2017 - The Event Horizon authors.
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

package httputils

import (
	"encoding/json"
	"net/http"
	"path"

	eh "github.com/looplab/eventhorizon"
)

// QueryHandler returns one or all items from a eventhorizon.ReadRepo. If the
// URL ends with a / it will return all items, otherwise it will try to use the
// last part of the path as an ID to return one item.
func QueryHandler(repo eh.ReadRepo) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "GET" {
			http.Error(w, "unsuported method: "+r.Method, http.StatusMethodNotAllowed)
			return
		}

		var (
			data interface{}
			err  error
		)
		// If there is a trailing slash in the URL we return all items,
		// otherwise we try to parse an ID from the last part to return one item.
		_, idStr := path.Split(r.URL.Path)
		if idStr == "" {
			if data, err = repo.FindAll(r.Context()); err != nil {
				http.Error(w, "could not find items: "+err.Error(), http.StatusInternalServerError)
				return
			}
		} else {
			id, err := eh.ParseUUID(idStr)
			if err != nil {
				http.Error(w, "could not parse ID: "+err.Error(), http.StatusBadRequest)
				return
			}

			if data, err = repo.Find(r.Context(), id); err != nil {
				if rrErr, ok := err.(eh.RepoError); ok && rrErr.Err == eh.ErrEntityNotFound {
					http.Error(w, "could not find item", http.StatusNotFound)
					return
				}

				http.Error(w, "could not find item: "+err.Error(), http.StatusInternalServerError)
				return
			}
		}

		b, err := json.Marshal(data)
		if err != nil {
			http.Error(w, "could not encode result: "+err.Error(), http.StatusInternalServerError)
			return
		}
		w.Write(b)
	})
}
