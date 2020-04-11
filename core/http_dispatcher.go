/*
Copyright © 2020 Blaster Contributors

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

package core

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"

	"github.com/pkg/errors"
)

type HttpDispatcher struct {
	HandlerUrl string
}

func (d *HttpDispatcher) Dispatch(m *Message) error {
	output, err := json.Marshal(m)
	if err != nil {
		return errors.WithStack(err)
	}

	response, err := http.Post(d.HandlerUrl, "application/json", bytes.NewReader(output))
	if err != nil {
		return errors.WithStack(err)
	}

	defer response.Body.Close()
	body, err := ioutil.ReadAll(response.Body)

	if err != nil {
		return errors.WithStack(err)
	}

	if response.StatusCode != 200 {
		return errors.WithStack(errors.New(fmt.Sprintf("error response from handler %d: %s", response.StatusCode, string(body))))
	}
	return nil
}

func NewHttpDispatcher(handlerUrl string) *HttpDispatcher {
	return &HttpDispatcher{
		HandlerUrl: handlerUrl,
	}
}
