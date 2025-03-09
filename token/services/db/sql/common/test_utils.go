/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package common

import (
	"context"
	"fmt"
	"io"
	"os"
	"runtime/debug"
	"testing"
	"time"

	"github.com/docker/docker/api/types/container"
	containertypes "github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/image"
	"github.com/docker/docker/client"
	postgres2 "github.com/hyperledger-labs/fabric-smart-client/platform/view/services/db/driver/sql/postgres"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/modules/postgres"
	"github.com/testcontainers/testcontainers-go/wait"
)

type PostgresConfig = postgres2.ContainerConfig
type DataSourceProvider = postgres2.DataSourceProvider

func DefaultPostgresConfig(node string) *PostgresConfig {
	return postgres2.DefaultConfig(node)
}

// https://testcontainers.com/guides/getting-started-with-testcontainers-for-go/
// Note: Before running tests: docker pull postgres:16.2-alpine
// Test may time out if image is not present on machine.
func StartPostgresContainer(t *testing.T) (func(), string) {
	if os.Getenv("TESTCONTAINERS") != "true" {
		t.Skip("set environment variable TESTCONTAINERS to true to include postgres test")
	}
	if testing.Short() {
		t.Skip("skipping postgres test in short mode")
	}

	ctx := context.Background()
	pg, err := postgres.Run(ctx, "postgres:16.2-alpine",
		testcontainers.WithWaitStrategy(
			wait.ForExposedPort().WithStartupTimeout(30*time.Second)),
		postgres.WithDatabase("testdb"),
		postgres.WithUsername("postgres"),
		postgres.WithPassword("example"),
	)
	if err != nil {
		t.Fatal(err)
	}
	pgConnStr, err := pg.ConnectionString(ctx, "sslmode=disable")
	if err != nil {
		t.Fatal(err)
	}

	return func() {
		if err := pg.Terminate(ctx); err != nil {
			logger.Errorf("failed to terminate [%s][%s]", err, debug.Stack())
		}
	}, pgConnStr
}

func StartHashicorpVaultContainer(t *testing.T) func() {
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
	io.Copy(os.Stdout, out)

	resp, err := cli.ContainerCreate(ctx, &container.Config{
		Image: imageName,
	}, nil, nil, nil, "")
	if err != nil {
		t.Fatal(err)
	}

	if err := cli.ContainerStart(ctx, resp.ID, container.StartOptions{}); err != nil {
		t.Fatal(err)
	}

	fmt.Println(resp.ID)

	return func() {
		noWaitTimeout := 0 // to not wait for the container to exit gracefully
		if err := cli.ContainerStop(ctx, resp.ID, containertypes.StopOptions{Timeout: &noWaitTimeout}); err != nil {
			logger.Errorf("failed to terminate hashicorp/vault [%s][%s]", err, debug.Stack())
		}
		fmt.Println("Success")
	}
}
