// Code generated by "stringer -type=VersionKey"; DO NOT EDIT.

package clusterversion

import "strconv"

func _() {
	// An "invalid array index" compiler error signifies that the constant values have changed.
	// Re-run the stringer command to generate them again.
	var x [1]struct{}
	_ = x[Version19_1-0]
	_ = x[VersionStart19_2-1]
	_ = x[VersionLearnerReplicas-2]
	_ = x[VersionTopLevelForeignKeys-3]
	_ = x[VersionAtomicChangeReplicasTrigger-4]
	_ = x[VersionAtomicChangeReplicas-5]
	_ = x[VersionTableDescModificationTimeFromMVCC-6]
	_ = x[VersionPartitionedBackup-7]
	_ = x[Version19_2-8]
	_ = x[VersionStart20_1-9]
	_ = x[VersionContainsEstimatesCounter-10]
	_ = x[VersionChangeReplicasDemotion-11]
	_ = x[VersionSecondaryIndexColumnFamilies-12]
	_ = x[VersionNamespaceTableWithSchemas-13]
	_ = x[VersionProtectedTimestamps-14]
	_ = x[VersionPrimaryKeyChanges-15]
	_ = x[VersionAuthLocalAndTrustRejectMethods-16]
	_ = x[VersionPrimaryKeyColumnsOutOfFamilyZero-17]
	_ = x[VersionRootPassword-18]
	_ = x[VersionNoExplicitForeignKeyIndexIDs-19]
	_ = x[VersionHashShardedIndexes-20]
	_ = x[VersionCreateRolePrivilege-21]
	_ = x[VersionStatementDiagnosticsSystemTables-22]
	_ = x[VersionSchemaChangeJob-23]
	_ = x[VersionSavepoints-24]
	_ = x[VersionTimeTZType-25]
	_ = x[VersionTimePrecision-26]
	_ = x[Version20_1-27]
	_ = x[VersionStart20_2-28]
	_ = x[VersionGeospatialType-29]
	_ = x[VersionAlterColumnTypeGeneral-30]
}

const _VersionKey_name = "Version19_1VersionStart19_2VersionLearnerReplicasVersionTopLevelForeignKeysVersionAtomicChangeReplicasTriggerVersionAtomicChangeReplicasVersionTableDescModificationTimeFromMVCCVersionPartitionedBackupVersion19_2VersionStart20_1VersionContainsEstimatesCounterVersionChangeReplicasDemotionVersionSecondaryIndexColumnFamiliesVersionNamespaceTableWithSchemasVersionProtectedTimestampsVersionPrimaryKeyChangesVersionAuthLocalAndTrustRejectMethodsVersionPrimaryKeyColumnsOutOfFamilyZeroVersionRootPasswordVersionNoExplicitForeignKeyIndexIDsVersionHashShardedIndexesVersionCreateRolePrivilegeVersionStatementDiagnosticsSystemTablesVersionSchemaChangeJobVersionSavepointsVersionTimeTZTypeVersionTimePrecisionVersion20_1VersionStart20_2VersionGeospatialTypeVersionAlterColumnTypeGeneral"

var _VersionKey_index = [...]uint16{0, 11, 27, 49, 75, 109, 136, 176, 200, 211, 227, 258, 287, 322, 354, 380, 404, 441, 480, 499, 534, 559, 585, 624, 646, 663, 680, 700, 711, 727, 748, 777}

func (i VersionKey) String() string {
	if i < 0 || i >= VersionKey(len(_VersionKey_index)-1) {
		return "VersionKey(" + strconv.FormatInt(int64(i), 10) + ")"
	}
	return _VersionKey_name[_VersionKey_index[i]:_VersionKey_index[i+1]]
}
