// Copyright 2020 The Cockroach Authors.
//
// Use of this software is governed by the Business Source License
// included in the file licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with
// the Business Source License, use of this software will be governed
// by the Apache License, Version 2.0, included in the file
// licenses/APL.txt.

package sql

import (
	"errors"
	"fmt"

	"github.com/cockroachdb/cockroach/pkg/clusterversion"
	"github.com/cockroachdb/cockroach/pkg/sql/parser"
	"github.com/cockroachdb/cockroach/pkg/sql/pgwire/pgcode"
	"github.com/cockroachdb/cockroach/pkg/sql/pgwire/pgerror"
	"github.com/cockroachdb/cockroach/pkg/sql/schemachange"
	"github.com/cockroachdb/cockroach/pkg/sql/sem/tree"
	"github.com/cockroachdb/cockroach/pkg/sql/sqlbase"
	"github.com/cockroachdb/cockroach/pkg/sql/types"
	"github.com/cockroachdb/cockroach/pkg/util/errorutil/unimplemented"
)

var usingExpressionNotSupportedErr = unimplemented.NewWithIssuef(
	47706, "alter column type using expression is not supported")

var colInIndexNotSupportedErr = unimplemented.NewWithIssuef(
	47636, "alter column type requiring rewrite of on-disk "+
		"data is currently not supported for columns that are part of an index")

var colOwnsSequenceNotSupportedErr = unimplemented.NewWithIssuef(
	48244, "alter column type for a column that owns a sequence "+
		"is currently not supported")

// AlterColumnType takes an AlterTableAlterColumnType, determines
// which conversion to use and applies the type conversion.
func AlterColumnType(
	tableDesc *sqlbase.MutableTableDescriptor,
	col *sqlbase.ColumnDescriptor,
	t *tree.AlterTableAlterColumnType,
	params runParams,
) error {
	typ, err := tree.ResolveType(t.ToType, params.p.semaCtx.GetTypeResolver())
	if err != nil {
		return err
	}

	version := params.ExecCfg().Settings.Version.ActiveVersionOrEmpty(params.ctx)
	if supported, err := isTypeSupportedInVersion(version, typ); err != nil {
		return err
	} else if !supported {
		return pgerror.Newf(
			pgcode.FeatureNotSupported,
			"type %s is not supported until version upgrade is finalized",
			typ.SQLString(),
		)
	}

	// Special handling for STRING COLLATE xy to verify that we recognize the language.
	if t.Collation != "" {
		if types.IsStringType(typ) {
			typ = types.MakeCollatedString(typ, t.Collation)
		} else {
			return pgerror.New(pgcode.Syntax, "COLLATE can only be used with string types")
		}
	}

	err = sqlbase.ValidateColumnDefType(typ)
	if err != nil {
		return err
	}

	// No-op if the types are Identical.  We don't use Equivalent here because
	// the user may be trying to change the type of the column without changing
	// the type family.
	if col.Type.Identical(typ) {
		return nil
	}

	kind, err := schemachange.ClassifyConversion(&col.Type, typ)
	if err != nil {
		return err
	}

	switch kind {
	case schemachange.ColumnConversionDangerous, schemachange.ColumnConversionImpossible:
		// We're not going to make it impossible for the user to perform
		// this conversion, but we do want them to explicit about
		// what they're going for.
		return pgerror.Newf(pgcode.CannotCoerce,
			"the requested type conversion (%s -> %s) requires an explicit USING expression",
			col.Type.SQLString(), typ.SQLString())
	case schemachange.ColumnConversionTrivial:
		col.Type = *typ
	case schemachange.ColumnConversionGeneral:
		if err := alterColumnTypeGeneral(tableDesc, col, t, params); err != nil {
			return err
		}
		if err := params.p.createOrUpdateSchemaChangeJob(params.ctx, tableDesc, "alter column type", tableDesc.ClusterVersion.NextMutationID); err != nil {
			return err
		}
		params.p.SendClientNotice(params.ctx, errors.New("alter column type changes are finalized asynchronously; "+
			"further schema changes on this table may be restricted until the job completes; "+
			"some inserts into the altered column may be rejected until the schema change is finalized"))
	default:
		return fmt.Errorf("unknown conversion for %s -> %s",
			col.Type.SQLString(), typ.SQLString())
	}

	return nil
}

func alterColumnTypeGeneral(
	tableDesc *sqlbase.MutableTableDescriptor,
	col *sqlbase.ColumnDescriptor,
	t *tree.AlterTableAlterColumnType,
	params runParams,
) error {
	// Make sure that all nodes in the cluster are able to perform general alter column type conversions.
	if !params.p.ExecCfg().Settings.Version.IsActive(params.ctx, clusterversion.VersionAlterColumnTypeGeneral) {
		return pgerror.Newf(pgcode.FeatureNotSupported,
			"all nodes are not the correct version for alter column type general")
	}
	if !params.SessionData().AlterColumnTypeGeneral {
		return pgerror.Newf(pgcode.FeatureNotSupported,
			"alter column type general is experimental; "+
				"you can enable alter column type general support by running `SET enable_experimental_alter_column_type_general = true`")
	}
	// Disallow ALTER COLUMN ... TYPE ... USING EXPRESSION.
	// Todo(richardjcai): Todo, need to handle "inverse" expression
	// during state after column swap but the old column has not been dropped.
	// Can allow the user to provide an inverse expression.
	if t.Using != nil {
		return usingExpressionNotSupportedErr
	}

	// Disallow ALTER COLUMN TYPE general for columns that own sequences.
	if len(col.OwnsSequenceIds) != 0 {
		return colOwnsSequenceNotSupportedErr
	}

	// Disallow ALTER COLUMN TYPE general for columns that are
	// part of indexes.
	for i := range tableDesc.Indexes {
		for _, id := range append(
			tableDesc.Indexes[i].ColumnIDs,
			tableDesc.Indexes[i].ExtraColumnIDs...) {
			if col.ID == id {
				return colInIndexNotSupportedErr
			}
		}
	}

	for _, id := range append(
		tableDesc.PrimaryIndex.ColumnIDs,
		tableDesc.PrimaryIndex.ExtraColumnIDs...) {
		if col.ID == id {
			return colInIndexNotSupportedErr
		}
	}

	currentMutationID := tableDesc.ClusterVersion.NextMutationID
	for i := range tableDesc.Mutations {
		mut := &tableDesc.Mutations[i]
		if mut.MutationID < currentMutationID {
			return unimplemented.NewWithIssuef(
				47137, "table %s is currently undergoing a schema change", tableDesc.Name)
		}
	}

	nameExists := func(name string) bool {
		_, _, err := tableDesc.FindColumnByName(tree.Name(name))
		return err == nil
	}

	shadowColName := sqlbase.GenerateUniqueConstraintName(col.Name, nameExists)

	toType, err := tree.ResolveType(t.ToType, params.p.semaCtx.GetTypeResolver())
	if err != nil {
		return err
	}

	// The default computed expression is casting the column to the new type.
	newComputedExpr := tree.CastExpr{
		Expr:       &tree.ColumnItem{ColumnName: tree.Name(col.Name)},
		Type:       toType,
		SyntaxMode: tree.CastShort,
	}
	s := tree.Serialize(&newComputedExpr)
	newColComputeExpr := &s

	// Create the default expression for the new column.
	hasDefault := col.HasDefault()
	var newColDefaultExpr *string
	if hasDefault {
		if col.HasNullDefault() {
			s := tree.Serialize(tree.DNull)
			newColDefaultExpr = &s
		} else {
			// The default expression for the new column is applying the
			// computed expression to the previous default expression.
			expr, err := parser.ParseExpr(col.DefaultExprStr())
			if err != nil {
				return err
			}
			newDefaultComputedExpr := tree.CastExpr{Expr: expr, Type: t.ToType, SyntaxMode: tree.CastShort}
			s := tree.Serialize(&newDefaultComputedExpr)
			newColDefaultExpr = &s
		}
	}

	newCol := sqlbase.ColumnDescriptor{
		Name:            shadowColName,
		Type:            *toType,
		Nullable:        col.Nullable,
		DefaultExpr:     newColDefaultExpr,
		UsesSequenceIds: col.UsesSequenceIds,
		OwnsSequenceIds: col.OwnsSequenceIds,
		ComputeExpr:     newColComputeExpr,
	}

	// Ensure new column is created in the same column family as the original
	// so backfiller writes to the same column family.
	family, err := tableDesc.GetFamilyOfColumn(col.ID)
	if err != nil {
		return err
	}

	if err := tableDesc.AddColumnToFamilyMaybeCreate(
		newCol.Name, family.Name, false, false); err != nil {
		return err
	}

	tableDesc.AddColumnMutation(&newCol, sqlbase.DescriptorMutation_ADD)

	if err := tableDesc.AllocateIDs(); err != nil {
		return err
	}

	swapArgs := &sqlbase.ComputedColumnSwap{
		OldColumnId: col.ID,
		NewColumnId: newCol.ID,
	}

	tableDesc.AddComputedColumnSwapMutation(swapArgs)

	return nil
}
