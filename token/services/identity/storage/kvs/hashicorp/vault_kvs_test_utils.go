/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package hashicorp

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"os"
	"runtime/debug"
	"strings"
	"testing"
	"time"

	container "github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/image"
	"github.com/docker/docker/client"
	"github.com/docker/go-connections/nat"
)

func StartHashicorpVaultContainer(t *testing.T) (func(), string, string) {
	ctx := context.Background()
	cli, err := client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation())
	if err != nil {
		t.Fatal(err)
	}
	defer cli.Close()

	imageName := "hashicorp/vault"

	out, err := cli.ImagePull(ctx, imageName, image.PullOptions{})
	if err != nil {
		t.Fatal(err)
	}
	defer out.Close()
	_, _ = io.Copy(os.Stdout, out)

	// Define the container configuration
	token := "00000000-0000-0000-0000-000000000000"
	address := "0.0.0.0:8200"
	containerConfig := &container.Config{
		Image: imageName,
		Env: []string{
			"VAULT_DEV_ROOT_TOKEN_ID=" + token,
			"VAULT_DEV_LISTEN_ADDRESS=" + address,
		},
	}
	// Define the host configuration
	hostConfig := &container.HostConfig{
		CapAdd: []string{"IPC_LOCK"},
		PortBindings: nat.PortMap{ // Use nat.PortMap for port bindings
			"8200/tcp": []nat.PortBinding{
				{
					HostIP:   "0.0.0.0",
					HostPort: "8200",
				},
			},
		},
	}
	// Create the container
	resp, err := cli.ContainerCreate(ctx, containerConfig, hostConfig, nil, nil, "")
	if err != nil {
		t.Fatal(err)
	}

	if err := cli.ContainerStart(ctx, resp.ID, container.StartOptions{}); err != nil {
		t.Fatal(err)
	}

	fmt.Println(resp.ID)

	// Wait for Vault to be ready
	vaultURL := fmt.Sprintf("http://%s", address)
	if err := waitForVault(vaultURL, token); err != nil {
		t.Fatal(err)
	}

	// Enable the kv secret engine version 1
	if err := enableKVSecretEngine(vaultURL, token, "kv1"); err != nil {
		t.Fatal(err)
	}

	return func() {
		noWaitTimeout := 0 // to not wait for the container to exit gracefully
		if err := cli.ContainerStop(ctx, resp.ID, container.StopOptions{Timeout: &noWaitTimeout}); err != nil {
			logger.Errorf("failed to terminate hashicorp/vault [%s][%s]", err, debug.Stack())
		}
		fmt.Println("Success")
	}, vaultURL, token
}

func waitForVault(vaultURL, token string) error {
	client := &http.Client{}
	req, err := http.NewRequest("GET", fmt.Sprintf("%s/v1/sys/health", vaultURL), nil)
	if err != nil {
		return err
	}
	req.Header.Set("X-Vault-Token", token)

	for i := 0; i < 30; i++ { // Try for 30 seconds
		resp, err := client.Do(req)
		if err == nil && resp.StatusCode == http.StatusOK {
			return nil
		}
		time.Sleep(1 * time.Second)
	}
	return fmt.Errorf("vault did not become ready in time")
}

func enableKVSecretEngine(vaultURL, token, path string) error {
	client := &http.Client{}
	reqBody := `{"type": "kv", "options": {"version": "1"}}`
	req, err := http.NewRequest("POST", fmt.Sprintf("%s/v1/sys/mounts/%s", vaultURL, path), strings.NewReader(reqBody))
	if err != nil {
		return err
	}
	req.Header.Set("X-Vault-Token", token)
	req.Header.Set("Content-Type", "application/json")

	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusNoContent {
		return fmt.Errorf("failed to enable kv secret engine: %s", resp.Status)
	}
	return nil
}
