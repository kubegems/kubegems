// Copyright 2022 The kubegems.io Authors
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

package argo

import (
	"bytes"
	"crypto/tls"
	"encoding/json"
	"errors"
	"io/ioutil"
	"net/http"
	"net/url"

	argocdcli "github.com/argoproj/argo-cd/v2/pkg/apiclient"
	gemlabels "kubegems.io/kubegems/pkg/apis/gems"
)

func NewArgoCDCli(options *Options) (*argocdcli.Client, error) {
	parsedurl, err := url.Parse(options.Addr)
	if err != nil {
		return nil, err
	}

	token := options.Token
	if token == "" {
		// create from user and password
		tk, err := GetTokenFromUserPassword(options.Addr, options.Username, options.Password)
		if err != nil {
			return nil, err
		}
		token = tk
	}

	cliopt := &argocdcli.ClientOptions{
		ServerAddr: parsedurl.Host,
		Insecure:   true, // Same with tls.SkipTLSVerify
		AuthToken:  token,
		// https://argo-cd.readthedocs.io/en/stable/faq/#why-am-i-getting-rpc-error-code-unavailable-desc-transport-is-closing-when-using-the-cli
		GRPCWeb: true,
	}

	if options.Addr == "" {
		cliopt.PortForward = true
		cliopt.PortForwardNamespace = gemlabels.NamespaceSystem
	}

	cli, err := argocdcli.NewClient(cliopt)
	if err != nil {
		return nil, err
	}
	return &cli, nil
}

func GetTokenFromUserPassword(addr string, username, password string) (string, error) {
	bts, err := json.Marshal(struct {
		Username string `json:"username,omitempty"`
		Password string `json:"password,omitempty"`
	}{
		Username: username,
		Password: password,
	})
	if err != nil {
		return "", err
	}

	client := http.Client{
		Transport: &http.Transport{
			TLSClientConfig: &tls.Config{
				InsecureSkipVerify: true,
			},
		},
	}

	resp, err := client.Post(addr+"/api/v1/session", "application/json", bytes.NewReader(bts))
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		bt, err := ioutil.ReadAll(resp.Body)
		if err != nil {
			return "", err
		}
		return "", errors.New(string(bt))
	}
	tokenresp := &struct {
		Token string `json:"token"`
	}{}
	if err := json.NewDecoder(resp.Body).Decode(tokenresp); err != nil {
		return "", err
	}
	return tokenresp.Token, nil
}
