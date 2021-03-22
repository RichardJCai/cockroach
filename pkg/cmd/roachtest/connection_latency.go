// Copyright 2021 The Cockroach Authors.
//
// Use of this software is governed by the Business Source License
// included in the file licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with
// the Business Source License, use of this software will be governed
// by the Apache License, Version 2.0, included in the file
// licenses/APL.txt.

package main

import (
	"context"
	"fmt"
	"github.com/stretchr/testify/require"
)

func registerConnectionLatencyTest(r *testRegistry) {
	runConnectionLatencyTest := func(ctx context.Context, t *test, c *cluster, numNodes int) {
		err := c.PutE(ctx, t.l, cockroach, "./cockroach")
		require.NoError(t, err)

		err = c.PutE(ctx, t.l, workload, "./workload")
		require.NoError(t, err)

		err = c.StartE(ctx, startArgs("--secure"))
		require.NoError(t, err)

		err = c.RunE(ctx, c.Node(1), `./cockroach sql --certs-dir certs -e "CREATE USER testuser"`)
		require.NoError(t, err)

		err = c.RunE(ctx, c.Node(1), `./cockroach cert create-client testuser --certs-dir certs --ca-key=certs/ca.key`)
		require.NoError(t, err)

		err = c.RunE(ctx, c.All(), "./workload init connectionlatency")
		require.NoError(t, err)

		workloadCmd := fmt.Sprintf(
			`./workload run connectionlatency --duration 30s --histograms=%s/stats.json`,
			perfArtifactsDir)
		err = c.RunE(ctx, c.All(), workloadCmd)
		require.NoError(t, err)
	}

	//geoZones := []string{"us-east1-b", "us-west1-b", "europe-west2-b"}
	//if cloud == aws {
	//	geoZones = []string{"us-east-2b", "us-west-1a", "eu-west-1a"}
	//}
	//geoZonesStr := strings.Join(geoZones, ",")

	nodesConfig := []int{1}
	//nodesConfig := []int{1, 3, 5}
	for _, numNodes := range nodesConfig {
		clusterSpec := makeClusterSpec(numNodes)
		clusterSpec.Secure = true
		r.Add(testSpec{
			MinVersion: "v20.1.0",
			Name:       fmt.Sprintf("connection_latency/nodes=%d", numNodes),
			Owner:      OwnerSQLExperience,
			Cluster:  clusterSpec  ,
			Run: func(ctx context.Context, t *test, c *cluster) {
				runConnectionLatencyTest(ctx, t, c, numNodes)
			},
		})
	}

	// Copying over multiregion configuration from indexes.go
	//numMultiRegionNodes := 6
	//r.Add(testSpec{
	//	MinVersion: "v20.1.0",
	//	Name:       fmt.Sprintf("connection_latency/nodes=%d/multiregion", numMultiRegionNodes),
	//	Owner:      OwnerSQLExperience,
	//	Cluster:    makeClusterSpec(numMultiRegionNodes, geo(), zones(geoZonesStr)),
	//	Run: func(ctx context.Context, t *test, c *cluster) {
	//		runConnectionLatencyTest(ctx, t, c, numMultiRegionNodes)
	//	},
	//})
}
