// Copyright (c) 2021 Gitpod GmbH. All rights reserved.
// Licensed under the GNU Affero General Public License (AGPL).
// See License-AGPL.txt in the project root for license information.

package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"net/url"
	"os"
	"time"

	supervisor "github.com/gitpod-io/gitpod/supervisor/api"
	"golang.org/x/xerrors"
	"google.golang.org/grpc"
)

// proxy for the Code With Me status endpoints that transforms it into the supervisor status format.
func main() {
	if len(os.Args) < 2 {
		fmt.Printf("Usage: %s <port> [<link label>]\n", os.Args[0])
		os.Exit(1)
	}
	port := os.Args[1]
	label := "Open JetBrains IDE"
	if len(os.Args) > 2 {
		label = os.Args[2]
	}

	errlog := log.New(os.Stderr, "JetBrains IDE status: ", log.LstdFlags)

	http.HandleFunc("/joinLink", func(w http.ResponseWriter, r *http.Request) {
		jsonLink, err := resolveJsonLink()
		if err != nil {
			http.Error(w, err.Error(), http.StatusServiceUnavailable)
			return
		}
		fmt.Fprint(w, jsonLink)
	})
	http.HandleFunc("/status", func(w http.ResponseWriter, r *http.Request) {
		wsInfo, err := resolveWorkspaceInfo(context.Background())
		if err != nil {
			errlog.Printf("cannot get workspace info: %v\n", err)
			http.Error(w, err.Error(), http.StatusServiceUnavailable)
		}
		gitpodUrl, err := url.Parse(wsInfo.GitpodHost)
		if err != nil {
			errlog.Printf("cannot parse gitpod url: %v\n", err)
			http.Error(w, err.Error(), http.StatusServiceUnavailable)
		}
		link := url.URL{
			Scheme:   "jetbrains-gateway",
			Host:     "connect",
			RawQuery: fmt.Sprintf("gitpodHost=%s&workspaceId=%s", url.QueryEscape(gitpodUrl.Hostname()), url.QueryEscape(wsInfo.WorkspaceId)),
		}
		response := make(map[string]string)
		response["link"] = link.String()
		response["label"] = label
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(response)
	})

	fmt.Printf("Starting status proxy for desktop IDE at port %s\n", port)
	if err := http.ListenAndServe(fmt.Sprintf(":%s", port), nil); err != nil {
		log.Fatal(err)
	}
}

type Projects struct {
	JoinLink string `json:"joinLink"`
}
type Response struct {
	Projects []Projects `json:"projects"`
}

func resolveJsonLink() (string, error) {
	var (
		hostStatusUrl = "http://localhost:63342/codeWithMe/unattendedHostStatus?token=gitpod"
		client        = http.Client{Timeout: 1 * time.Second}
	)
	resp, err := client.Get(hostStatusUrl)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	bodyBytes, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}
	if resp.StatusCode != http.StatusOK {
		return "", xerrors.Errorf("failed to resolve project status: %s (%d)", bodyBytes, resp.StatusCode)
	}
	jsonResp := &Response{}
	err = json.Unmarshal(bodyBytes, &jsonResp)
	if err != nil {
		return "", err
	}
	if len(jsonResp.Projects) != 1 {
		return "", xerrors.Errorf("project is not found")
	}
	return jsonResp.Projects[0].JoinLink, nil
}

func resolveWorkspaceInfo(ctx context.Context) (*supervisor.WorkspaceInfoResponse, error) {
	supervisorAddr := os.Getenv("SUPERVISOR_ADDR")
	if supervisorAddr == "" {
		supervisorAddr = "localhost:22999"
	}
	supervisorConn, err := grpc.Dial(supervisorAddr, grpc.WithInsecure())
	if err != nil {
		return nil, xerrors.Errorf("failed connecting to supervisor: %w", err)
	}
	defer supervisorConn.Close()
	wsinfo, err := supervisor.NewInfoServiceClient(supervisorConn).WorkspaceInfo(ctx, &supervisor.WorkspaceInfoRequest{})
	if err != nil {
		return nil, xerrors.Errorf("failed getting workspace info from supervisor: %w", err)
	}
	return wsinfo, nil
}
