// Copyright 2021 The Cockroach Authors.
//
// Use of this software is governed by the Business Source License
// included in the file licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with
// the Business Source License, use of this software will be governed
// by the Apache License, Version 2.0, included in the file
// licenses/APL.txt.

package connectionlatency

import (
	"context"
	"strings"

	"github.com/cockroachdb/cockroach/pkg/util/timeutil"
	"github.com/cockroachdb/cockroach/pkg/workload"
	"github.com/cockroachdb/cockroach/pkg/workload/histogram"
	"github.com/cockroachdb/errors"
	"github.com/jackc/pgx/v4"
	"github.com/spf13/pflag"
)

type connectionLatency struct {
	flags     workload.Flags
	connFlags *workload.ConnFlags
	hists     *histogram.Histograms
}

func init() {
	workload.Register(connectionLatencyMeta)
}

var connectionLatencyMeta = workload.Meta{
	Name:         `connectionlatency`,
	Description:  `Testing Connection Latencies`,
	Version:      `1.0.0`,
	PublicFacing: false,
	New: func() workload.Generator {
		c := &connectionLatency{}
		c.flags.FlagSet = pflag.NewFlagSet(`connectionlatency`, pflag.ContinueOnError)
		c.connFlags = workload.NewConnFlags(&c.flags)
		return c
	},
}

// Meta implements the Generator interface.
func (connectionLatency) Meta() workload.Meta { return connectionLatencyMeta }

// Tables implements the Generator interface.
func (connectionLatency) Tables() []workload.Table {
	return nil
}

// Ops implements the Opser interface.
func (c *connectionLatency) Ops(
	ctx context.Context, urls []string, reg *histogram.Registry,
) (workload.QueryLoad, error) {
	println("urls:")
	println(urls)
	ql := workload.QueryLoad{}
	if len(urls) != 1 {
		return workload.QueryLoad{}, errors.New("expected urls to be length 1")
	}
	op := &connectionOp{
		url:   urls[0],
		hists: reg.GetHandle(),
	}
	ql.WorkerFns = append(ql.WorkerFns, op.run)
	return ql, nil
}

type connectionOp struct {
	url   string
	hists *histogram.Histograms
}

func (o *connectionOp) run(ctx context.Context) error {
	start := timeutil.Now()
	println(o.url)
	x := strings.Replace(o.url, "root","testuser", -1)
	newUrl := strings.Replace(x, "verify-full", "require", -1)
	println(newUrl)
	conn, err := pgx.Connect(ctx, newUrl)
	if err != nil {
		return err
	}
	defer conn.Close(ctx)
	elapsed := timeutil.Since(start)
	o.hists.Get(`connect`).Record(elapsed)

	if _, err = conn.Exec(ctx, "SELECT 1"); err != nil {
		return err
	}
	// Record the time it takes to do a select after connecting for reference.
	elapsed = timeutil.Since(start)
	o.hists.Get(`select`).Record(elapsed)

	//var nodeId int
	//var sessionId string
	//var user string
	//var clientAddress string

	var applicationName string
	row := conn.QueryRow(ctx, "SELECT user FROM crdb_internal.cluster_sessions")
	err = row.Scan(&applicationName)
	if err != nil {
		return err
	}
	//println("stuff:")
	println(applicationName)

	//var n int
	//if err := conn.QueryRow(ctx, "SELECT 1").Scan(&n); err != nil {
	//	return err
	//}
	//if n != 1 {
	//	return errors.Errorf("expected 1 got %d", n)
	//}
	return nil
}
