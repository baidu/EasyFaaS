/*
 * Copyright (c) 2020 Baidu, Inc.
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

package client

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"net/url"
	"time"

	"github.com/spf13/pflag"

	"github.com/baidu/easyfaas/pkg/api"
	"github.com/baidu/easyfaas/pkg/rest"
)

const DefaultFuncletApiSocket = "/var/run/faas/.funcletapi.sock"

var (
	baseURL, _ = url.Parse("http://funclet")
	timeout    = 10 * time.Second
)

type FuncletClientOptions struct {
	ApiSock string
}

func NewFuncletClientOptions() *FuncletClientOptions {
	return &FuncletClientOptions{
		ApiSock: DefaultFuncletApiSocket,
	}
}

// AddFunctionCacheFlags xxx
func (o *FuncletClientOptions) AddFuncletClientFlags(fs *pflag.FlagSet) {
	fs.StringVar(&o.ApiSock, "funclet-api-sock", o.ApiSock, "The api socket of funclet")
}

type FuncletInterface interface {
	List(*api.FuncletClientListContainersInput) (*api.ListContainersResponse, error)
	Info(*api.FuncletClientContainerInfoInput) (*api.ContainerInfoResponse, error)
	IDEWarmUp(*api.FuncletClientWarmUpInput) (*api.WarmUpResponse, error)
	WarmUp(*api.FuncletClientWarmUpInput) (*api.WarmUpResponse, error)
	CoolDown(*api.FuncletClientCoolDownInput) (*api.ResetResponse, error)
	Reborn(*api.FuncletClientRebornInput) (*api.ResetResponse, error)
	NodeInfo(*api.FuncletClientListNodeInput) (*api.FuncletNodeInfo, error)
}

// FuncletClient is used to comm with funclet
type FuncletClient struct {
	client   *rest.RESTClient
	protocol string
}

// NewFuncletClient create a kubernetes client
func NewFuncletClient(option *FuncletClientOptions) FuncletInterface {
	version := "v1"
	config := rest.ContentConfig{
		BackendType: rest.BackendTypeInternal,
	}
	client := &http.Client{
		Transport: &http.Transport{
			DialContext: func(_ context.Context, _, _ string) (net.Conn, error) {
				return net.Dial("unix", option.ApiSock)
			},
		},
	}
	restCli, _ := rest.NewRESTClient(&url.URL{}, version, config, client)
	return &FuncletClient{
		client:   restCli,
		protocol: "http",
	}
}

func (f *FuncletClient) Info(input *api.FuncletClientContainerInfoInput) (out *api.ContainerInfoResponse, err error) {
	out = &api.ContainerInfoResponse{}
	req := f.client.Get().
		BaseURL(baseURL).
		Resource(fmt.Sprintf("funclet/container/%s", input.ID)).
		Timeout(timeout)

	if input.RequestID != "" {
		req = req.SetHeader(api.HeaderXRequestID, input.RequestID)
	}
	if err := req.Do().Into(out); err != nil {
		return nil, err
	}
	return out, nil
}

func (f *FuncletClient) List(input *api.FuncletClientListContainersInput) (out *api.ListContainersResponse, err error) {
	request := &api.ListContainerCriteria{
		input.Criteria,
	}

	out = &api.ListContainersResponse{}
	req := f.client.Get().
		BaseURL(baseURL).
		Resource("funclet/list").
		Body(request).
		Timeout(timeout)

	if input.RequestID != "" {
		req = req.SetHeader(api.HeaderXRequestID, input.RequestID)
	}

	if err := req.Do().Into(out); err != nil {
		return nil, err
	}
	return out, nil
}

func (f *FuncletClient) IDEWarmUp(input *api.FuncletClientWarmUpInput) (out *api.WarmUpResponse, err error) {
	out = &api.WarmUpResponse{}
	req := f.client.Post().
		BaseURL(baseURL).
		Resource("funclet/ide-warmup").
		Body(input).
		Timeout(timeout)

	if input.RequestID != "" {
		req = req.SetHeader(api.HeaderXRequestID, input.RequestID)
	}
	if err := req.Do().Into(out); err != nil {
		return nil, err
	}
	return out, nil
}

func (f *FuncletClient) WarmUp(input *api.FuncletClientWarmUpInput) (out *api.WarmUpResponse, err error) {
	out = &api.WarmUpResponse{}
	req := f.client.Post().
		BaseURL(baseURL).
		Resource("funclet/warmup").
		Body(input).
		Timeout(timeout)

	if input.RequestID != "" {
		req = req.SetHeader(api.HeaderXRequestID, input.RequestID)
	}
	if err := req.Do().Into(out); err != nil {
		return nil, err
	}
	return out, nil
}

func (f *FuncletClient) CoolDown(input *api.FuncletClientCoolDownInput) (out *api.ResetResponse, err error) {
	body := api.ResetRequest{
		ContainerID:             input.ContainerID,
		RequestID:               input.RequestID,
		ScaleDownRecommendation: input.ScaleDownRecommendation,
	}

	out = &api.ResetResponse{}
	req := f.client.Post().
		BaseURL(baseURL).
		Resource("funclet/cooldown").
		Body(body).
		Timeout(timeout * 3)
	if input.RequestID != "" {
		req = req.SetHeader(api.HeaderXRequestID, input.RequestID)
	}

	if err := req.Do().Into(out); err != nil {
		return nil, err
	}
	return out, nil
}

func (f *FuncletClient) Reborn(input *api.FuncletClientRebornInput) (out *api.ResetResponse, err error) {
	body := api.ResetRequest{
		ContainerID:             input.ContainerID,
		RequestID:               input.RequestID,
		ScaleDownRecommendation: input.ScaleDownRecommendation,
	}
	out = &api.ResetResponse{}
	req := f.client.Post().
		BaseURL(baseURL).
		Resource("funclet/reborn").
		Body(body).
		Timeout(timeout * 3)

	if input.RequestID != "" {
		req = req.SetHeader(api.HeaderXRequestID, input.RequestID)
	}

	if err := req.Do().Into(out); err != nil {
		return nil, err
	}
	return out, nil
}

func (f *FuncletClient) NodeInfo(input *api.FuncletClientListNodeInput) (out *api.FuncletNodeInfo, err error) {
	out = &api.FuncletNodeInfo{}
	req := f.client.Get().
		BaseURL(baseURL).
		Resource("funclet/node").
		Timeout(timeout)

	if input.RequestID != "" {
		req = req.SetHeader(api.HeaderXRequestID, input.RequestID)
	}

	if err := req.Do().Into(out); err != nil {
		return nil, err
	}
	return
}
