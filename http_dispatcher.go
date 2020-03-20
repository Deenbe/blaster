package main

import (
	"bytes"
	"encoding/json"
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

	_, err = http.Post(d.HandlerUrl, "application/json", bytes.NewReader(output))
	if err != nil {
		return errors.WithStack(err)
	}

	return nil
}

func NewHttpDispatcher(handlerUrl string) *HttpDispatcher {
	return &HttpDispatcher{
		HandlerUrl: handlerUrl,
	}
}
