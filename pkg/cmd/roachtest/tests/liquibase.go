// Copyright 2021 The Cockroach Authors.
//
// Use of this software is governed by the Business Source License
// included in the file licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with
// the Business Source License, use of this software will be governed
// by the Apache License, Version 2.0, included in the file
// licenses/APL.txt.

package tests

import (
	"context"
	"fmt"
	"strings"

	"github.com/cockroachdb/cockroach/pkg/cmd/roachtest/cluster"
	"github.com/cockroachdb/cockroach/pkg/cmd/roachtest/registry"
	"github.com/cockroachdb/cockroach/pkg/cmd/roachtest/test"
)

var supportedLiquibaseHarnessCommit = "05a91da4869b68f9e3a47c033f856acfb9f01106"

// This test runs the Liquibase test harness against a single cockroach node.
func registerLiquibase(r registry.Registry) {
	runLiquibase := func(
		ctx context.Context,
		t test.Test,
		c cluster.Cluster,
	) {
		if c.IsLocal() {
			t.Fatal("cannot be run in local mode")
		}
		node := c.Node(1)
		t.Status("setting up cockroach")
		c.Put(ctx, t.Cockroach(), "./cockroach", c.All())
		c.Start(ctx, c.All())

		version, err := fetchCockroachVersion(ctx, c, node[0], nil)
		if err != nil {
			t.Fatal(err)
		}

		if err := alterZoneConfigAndClusterSettings(ctx, version, c, node[0], nil); err != nil {
			t.Fatal(err)
		}

		if err := repeatRunE(
			ctx, t, c, node, "update apt-get", `sudo apt-get -qq update`,
		); err != nil {
			t.Fatal(err)
		}

		t.Status("cloning liquibase test harness and installing prerequisites")
		if err := repeatRunE(
			ctx, t, c, node, "update apt-get", `sudo apt-get -qq update`,
		); err != nil {
			t.Fatal(err)
		}

		// TODO(rafi): use openjdk-11-jdk-headless once we are off of Ubuntu 16.
		if err := repeatRunE(
			ctx,
			t,
			c,
			node,
			"install dependencies",
			`sudo apt-get -qq install default-jre openjdk-8-jdk-headless maven`,
		); err != nil {
			t.Fatal(err)
		}

		if err := repeatRunE(
			ctx, t, c, node, "remove old liquibase test harness",
			`rm -rf /mnt/data1/liquibase-test-harness`,
		); err != nil {
			t.Fatal(err)
		}

		// TODO(richardjcai): When liquibase-test-harness 1.0.3 is released and tagged,
		//    use the tag version instead of the commit.
		if err = c.RunE(ctx, node, "cd /mnt/data1/ && git clone https://github.com/liquibase/liquibase-test-harness.git"); err != nil {
			t.Fatal(err)
		}
		if err = c.RunE(ctx, node, fmt.Sprintf("cd /mnt/data1/liquibase-test-harness/ && git checkout %s",
			supportedLiquibaseHarnessCommit)); err != nil {
			t.Fatal(err)
		}

		// The liquibase harness comes with a script that sets up the database.
		// The script executes the cockroach binary from /cockroach/cockroach.sh
		// so we symlink that here.
		t.Status("creating database/user used by tests")
		if err = c.RunE(ctx, node, `sudo mkdir /cockroach && sudo ln -sf /home/ubuntu/cockroach /cockroach/cockroach.sh`); err != nil {
			t.Fatal(err)
		}
		if err = c.RunE(ctx, node, `/mnt/data1/liquibase-test-harness/src/test/resources/docker/setup_db.sh localhost`); err != nil {
			t.Fatal(err)
		}

		t.Status("running liquibase test harness")

		rawResults, err := c.RunWithBuffer(ctx, t.L(), node,
			`cd /mnt/data1/liquibase-test-harness/ && mvn test -Dtest=LiquibaseHarnessSuiteTest -DdbName=cockroachdb -DdbVersion=20.2`,
		)
		rawResultsStr := string(rawResults)
		t.L().Printf("Test Results: %s", rawResultsStr)
		if err != nil {
			if strings.Contains(rawResultsStr, "Failures: 1") &&
				strings.Contains(rawResultsStr, "Failed tests:   "+
					"apply addDefaultValueSequenceNext against cockroachdb 20.2; "+
					"verify generated SQL and DB snapshot(liquibase.harness.change.ChangeObjectTests): "+
					"Condition failed with Exception:(..)") {
				err = nil
			}
			if err != nil {
				t.Fatal(err)
			}
		}
	}

	r.Add(registry.TestSpec{
		Name:    "liquibase",
		Owner:   registry.OwnerSQLExperience,
		Cluster: r.MakeClusterSpec(1),
		Tags:    []string{`default`, `tool`},
		Run: func(ctx context.Context, t test.Test, c cluster.Cluster) {
			runLiquibase(ctx, t, c)
		},
	})
}
