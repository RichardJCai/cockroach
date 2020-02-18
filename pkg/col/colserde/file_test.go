// Copyright 2019 The Cockroach Authors.
//
// Use of this software is governed by the Business Source License
// included in the file licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with
// the Business Source License, use of this software will be governed
// by the Apache License, Version 2.0, included in the file
// licenses/APL.txt.

package colserde_test

import (
	"bytes"
	"testing"

	"github.com/cockroachdb/cockroach/pkg/col/coldata"
	"github.com/cockroachdb/cockroach/pkg/col/colserde"
	"github.com/cockroachdb/cockroach/pkg/col/coltypes"
	"github.com/cockroachdb/cockroach/pkg/util/leaktest"
	"github.com/stretchr/testify/require"
)

func TestFileIndexing(t *testing.T) {
	defer leaktest.AfterTest(t)()

	const numInts = 10
	typs := []coltypes.T{coltypes.Int64}

	var buf bytes.Buffer
	s, err := colserde.NewFileSerializer(&buf, typs)
	require.NoError(t, err)

	for i := 0; i < numInts; i++ {
		b := coldata.NewMemBatchWithSize(typs, 1)
		b.SetLength(1)
		b.ColVec(0).Int64()[0] = int64(i)
		require.NoError(t, s.AppendBatch(b))
	}
	require.NoError(t, s.Finish())

	d, err := colserde.NewFileDeserializerFromBytes(buf.Bytes())
	require.NoError(t, err)
	defer func() { require.NoError(t, d.Close()) }()
	require.Equal(t, typs, d.Typs())
	require.Equal(t, numInts, d.NumBatches())
	for batchIdx := numInts - 1; batchIdx >= 0; batchIdx-- {
		b := coldata.NewMemBatchWithSize(nil, 0)
		require.NoError(t, d.GetBatch(batchIdx, b))
		require.Equal(t, uint16(1), b.Length())
		require.Equal(t, 1, b.Width())
		require.Equal(t, coltypes.Int64, b.ColVec(0).Type())
		require.Equal(t, int64(batchIdx), b.ColVec(0).Int64()[0])
	}
}
