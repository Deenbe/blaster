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
