/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package hashicorp

import (
	"context"
	"fmt"
	"net/http"
	"runtime/debug"
	"strings"
	"testing"
	"time"

	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/client"
	"github.com/docker/go-connections/nat"
	vault "github.com/hashicorp/vault/api"
	docker2 "github.com/hyperledger-labs/fabric-smart-client/integration/nwo/common/docker"
	"github.com/pkg/errors"
)

const (
	ImageName           = "hashicorp/vault"
	ContainerNamePrefix = "dev-hashicorp-vault-container-"
)

// NewVaultClient creates a new Vault client
func NewVaultClient(address, token string) (*vault.Client, error) {
	config := vault.DefaultConfig()
	config.Address = address

	client, err := vault.NewClient(config)
	if err != nil {
		return nil, errors.Wrap(err, "failed to create Vault client")
	}

	client.SetToken(token)

	return client, nil
}

func StartHashicorpVaultContainer(t *testing.T, port int) (func(), string, string) {
	docker, err := docker2.GetInstance()
	if err != nil {
		t.Fatalf("failed to connect to docker daemon: %v", err)
	}
	err = docker.CheckImagesExist(ImageName)
	if err != nil {
		t.Fatal(err)
	}

	cli, err := client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation())
	if err != nil {
		t.Fatal(err)
	}
	// Define the container configuration
	portStr := fmt.Sprintf("%d", port)
	token := "00000000-0000-0000-0000-000000000000"
	address := "0.0.0.0:" + portStr
	portBinding := nat.Port(fmt.Sprintf("%d/tcp", port))
	containerConfig := &container.Config{
		Image: ImageName,
		Env: []string{
			"VAULT_DEV_ROOT_TOKEN_ID=" + token,
			"VAULT_DEV_LISTEN_ADDRESS=" + address,
		},
		ExposedPorts: map[nat.Port]struct{}{
			portBinding: {},
		},
	}

	// Define the host configuration
	hostConfig := &container.HostConfig{
		CapAdd: []string{"IPC_LOCK"},
		PortBindings: nat.PortMap{ // Use nat.PortMap for port bindings
			portBinding: []nat.PortBinding{
				{
					HostIP:   "0.0.0.0",
					HostPort: portStr,
				},
			},
		},
	}
	// Create the container
	ctx := context.Background()
	containerName := ContainerNamePrefix + portStr
	resp, err := cli.ContainerCreate(
		ctx,
		containerConfig,
		hostConfig,
		nil,
		nil,
		containerName,
	)
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
		time.Sleep(2 * time.Second)
	}
	return errors.Errorf("vault did not become ready in time")
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
		return errors.Errorf("failed to enable kv secret engine: %s", resp.Status)
	}
	return nil
}
