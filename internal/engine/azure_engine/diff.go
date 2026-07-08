// internal/engine/azure_engine/diff.go
package azure_engine

import (
	"errors"
	"fmt"
	"reflect"

	"github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/dataprotection/armdataprotection"
	"github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/recoveryservices/armrecoveryservicesbackup"
	"github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/sql/armsql"
	"github.com/dbjorklund/backup-policy-as-code/internal/models/azure"
)

// TODO: Implement sqlVM, SqlDb

// DiffVMPolicy compares a local ResolvedVM against the live Azure
// protected-item response. Returns four maps:
//   - equal:       fields that match between local and remote
//   - toApply:     fields in local that differ from remote (need to be applied)
//   - local:       the full local map (useful for dry-run output)
//   - toOverwrite: the remote values that would be overwritten
func DiffVMPolicy(
	local azure.ResolvedVM,
	remote armrecoveryservicesbackup.ProtectedItemsClientGetResponse,
) (
	equal map[string]interface{},
	toApply map[string]interface{},
	localMap map[string]interface{},
	toOverwrite map[string]interface{},
	err error,
) {
	vmItem, ok := remote.Properties.(*armrecoveryservicesbackup.AzureIaaSComputeVMProtectedItem)
	if !ok {
		return nil, nil, nil, nil, fmt.Errorf(
			"remote protected item is not an AzureIaaSComputeVMProtectedItem (got %T)",
			remote.Properties,
		)
	}

	remoteMap := map[string]any{
		"policyName":      derefStringPtr(vmItem.PolicyName),
		"isInclusionList": derefIsInclusionList(vmItem.ExtendedProperties),
		"diskLunList":     derefDiskLunList(vmItem.ExtendedProperties),
	}

	localMap = resolvedVMToMap(local)

	equal, toApply, toOverwrite, err = compareMaps(localMap, remoteMap)
	if err != nil {
		return nil, nil, nil, nil, fmt.Errorf("comparing VM policy maps: %w", err)
	}

	return equal, toApply, localMap, toOverwrite, nil
}

// DiffVMRetention compares the local AzureRetention spec against the live
// Azure LongTermRetentionPolicy.
func DiffVMRetention(
	local azure.ResolvedVM,
	remote *armrecoveryservicesbackup.LongTermRetentionPolicy,
) (
	equal map[string]interface{},
	toApply map[string]interface{},
	localMap map[string]interface{},
	toOverwrite map[string]interface{},
	err error,
) {
	if remote == nil {
		return nil, nil, nil, nil, errors.New("remote retention policy is nil")
	}

	localMap = retentionToMap(local)
	remoteMap, err := remoteRetentionToMap(remote)
	if err != nil {
		return nil, nil, nil, nil, fmt.Errorf("parsing remote retention policy: %w", err)
	}

	equal, toApply, toOverwrite, err = compareMaps(localMap, remoteMap)
	if err != nil {
		return nil, nil, nil, nil, fmt.Errorf("comparing retention maps: %w", err)
	}

	return equal, toApply, localMap, toOverwrite, nil
}

// resolvedVMToMap builds the local side of a VM policy comparison map.
func resolvedVMToMap(r azure.ResolvedVM) map[string]any {
	luns := []int32{}
	isInclusionList := false // default: back up all disks, no inclusion filter

	if ds := r.DiskSelection; ds != nil {
		switch ds.Mode {
		case "include":
			luns = ds.LUNs
			isInclusionList = true
		case "exclude":
			luns = ds.LUNs
			// isInclusionList stays false — exclusion list
		}
	}

	return map[string]any{
		"policyName":      r.PolicyName,
		"isInclusionList": isInclusionList,
		"diskLunList":     luns,
	}
}

// retentionToMap flattens the local AzureRetention into a comparable key/value map.
// Counts are stored as int32 to match the remote representation returned by remoteRetentionToMap.
func retentionToMap(r azure.ResolvedVM) map[string]any {
	m := map[string]any{}
	ret := r.Spec.Retention

	m["retention.daily.enabled"] = ret.Daily != nil
	m["retention.weekly.enabled"] = ret.Weekly != nil
	m["retention.monthly.enabled"] = ret.Monthly != nil
	m["retention.yearly.enabled"] = ret.Yearly != nil

	if ret.Daily != nil {
		m["retention.daily.count"] = int32(ret.Daily.Count)
	}
	if ret.Weekly != nil {
		m["retention.weekly.count"] = int32(ret.Weekly.Count)
		m["retention.weekly.day"] = ret.Weekly.Day
	}
	if ret.Monthly != nil {
		m["retention.monthly.count"] = int32(ret.Monthly.Count)
	}
	if ret.Yearly != nil {
		m["retention.yearly.count"] = int32(ret.Yearly.Count)
		m["retention.yearly.month"] = ret.Yearly.Month
		m["retention.yearly.day"] = ret.Yearly.Day
		m["retention.yearly.week"] = ret.Yearly.Week
	}

	return m
}

// remoteRetentionToMap flattens an Azure LongTermRetentionPolicy into the same
// key shape as retentionToMap so compareMaps can work against them.
func remoteRetentionToMap(r *armrecoveryservicesbackup.LongTermRetentionPolicy) (map[string]any, error) {
	m := make(map[string]any)

	m["retention.daily.enabled"] = r.DailySchedule != nil &&
		r.DailySchedule.RetentionDuration != nil &&
		derefInt32(r.DailySchedule.RetentionDuration.Count) > 0

	m["retention.weekly.enabled"] = r.WeeklySchedule != nil &&
		r.WeeklySchedule.RetentionDuration != nil &&
		derefInt32(r.WeeklySchedule.RetentionDuration.Count) > 0

	m["retention.monthly.enabled"] = r.MonthlySchedule != nil &&
		r.MonthlySchedule.RetentionDuration != nil &&
		derefInt32(r.MonthlySchedule.RetentionDuration.Count) > 0

	m["retention.yearly.enabled"] = r.YearlySchedule != nil &&
		r.YearlySchedule.RetentionDuration != nil &&
		derefInt32(r.YearlySchedule.RetentionDuration.Count) > 0

	if r.DailySchedule != nil && r.DailySchedule.RetentionDuration != nil {
		m["retention.daily.count"] = derefInt32(r.DailySchedule.RetentionDuration.Count)
	}
	if r.WeeklySchedule != nil && r.WeeklySchedule.RetentionDuration != nil {
		m["retention.weekly.count"] = derefInt32(r.WeeklySchedule.RetentionDuration.Count)
		if len(r.WeeklySchedule.DaysOfTheWeek) > 0 && r.WeeklySchedule.DaysOfTheWeek[0] != nil {
			m["retention.weekly.day"] = string(*r.WeeklySchedule.DaysOfTheWeek[0])
		}
	}
	if r.MonthlySchedule != nil && r.MonthlySchedule.RetentionDuration != nil {
		m["retention.monthly.count"] = derefInt32(r.MonthlySchedule.RetentionDuration.Count)
	}
	if r.YearlySchedule != nil && r.YearlySchedule.RetentionDuration != nil {
		m["retention.yearly.count"] = derefInt32(r.YearlySchedule.RetentionDuration.Count)
		if len(r.YearlySchedule.MonthsOfYear) > 0 && r.YearlySchedule.MonthsOfYear[0] != nil {
			m["retention.yearly.month"] = string(*r.YearlySchedule.MonthsOfYear[0])
		}
		if r.YearlySchedule.RetentionScheduleWeekly != nil {
			if len(r.YearlySchedule.RetentionScheduleWeekly.DaysOfTheWeek) > 0 &&
				r.YearlySchedule.RetentionScheduleWeekly.DaysOfTheWeek[0] != nil {
				m["retention.yearly.day"] = string(*r.YearlySchedule.RetentionScheduleWeekly.DaysOfTheWeek[0])
			}
			if len(r.YearlySchedule.RetentionScheduleWeekly.WeeksOfTheMonth) > 0 &&
				r.YearlySchedule.RetentionScheduleWeekly.WeeksOfTheMonth[0] != nil {
				m["retention.yearly.week"] = string(*r.YearlySchedule.RetentionScheduleWeekly.WeeksOfTheMonth[0])
			}
		}
	}

	return m, nil
}

// compareMaps returns where mapOne (local desired) differs from mapTwo (remote actual).
func compareMaps(mapOne, mapTwo map[string]any) (
	equal map[string]any,
	diff map[string]any,
	overwritten map[string]any,
	err error,
) {

	if len(mapOne) == 0 {
		return nil, nil, nil, errors.New("mapOne must not be nil or empty")
	}
	for k, v := range mapOne {
		if v == nil {
			return nil, nil, nil, fmt.Errorf("mapOne contains a nil value for key %q", k)
		}
	}

	equal = make(map[string]any)
	diff = make(map[string]any)
	overwritten = make(map[string]any)

	for key, valueOne := range mapOne {
		valueTwo, exists := mapTwo[key]
		if !exists {
			diff[key] = valueOne
			continue
		}
		if reflect.DeepEqual(valueOne, valueTwo) {
			equal[key] = valueOne
		} else {
			diff[key] = valueOne
			overwritten[key] = valueTwo
		}
	}

	return equal, diff, overwritten, nil
}

// --- pointer dereference helpers ---

func derefStringPtr(s *string) string {
	if s == nil {
		return ""
	}
	return *s
}

func derefInt32(p *int32) int32 {
	if p == nil {
		return 0
	}
	return *p
}

func derefDiskLunList(ep *armrecoveryservicesbackup.ExtendedProperties) []int32 {
	if ep == nil || ep.DiskExclusionProperties == nil {
		return []int32{}
	}
	result := make([]int32, 0, len(ep.DiskExclusionProperties.DiskLunList))
	for _, ptr := range ep.DiskExclusionProperties.DiskLunList {
		if ptr != nil {
			result = append(result, *ptr)
		}
	}
	return result
}

func derefIsInclusionList(ep *armrecoveryservicesbackup.ExtendedProperties) bool {
	if ep == nil || ep.DiskExclusionProperties == nil || ep.DiskExclusionProperties.IsInclusionList == nil {
		return false
	}
	return *ep.DiskExclusionProperties.IsInclusionList
}

// DiffDiskPolicy compares a local ResolvedDisk against the live Azure
// backup instance response. Returns four maps:
//   - equal:       fields that match between local and remote
//   - toApply:     fields in local that differ from remote (need to be applied)
//   - local:       the full local map (useful for dry-run output)
//   - toOverwrite: the remote values that would be overwritten
func DiffDiskPolicy(
	local azure.ResolvedDisk,
	remote armdataprotection.BackupInstancesClientGetResponse,
) (
	equal map[string]interface{},
	toApply map[string]interface{},
	localMap map[string]interface{},
	toOverwrite map[string]interface{},
	err error,
) {
	instance := remote.Properties
	if instance == nil {
		return nil, nil, nil, nil, fmt.Errorf("remote backup instance properties are nil")
	}

	remotePolicyID := ""
	if instance.PolicyInfo != nil && instance.PolicyInfo.PolicyID != nil {
		remotePolicyID = *instance.PolicyInfo.PolicyID
	}

	remoteMap := map[string]any{
		"policyID": remotePolicyID,
	}

	localMap = resolvedDiskToMap(local)

	equal, toApply, toOverwrite, err = compareMaps(localMap, remoteMap)
	if err != nil {
		return nil, nil, nil, nil, fmt.Errorf("comparing disk policy maps: %w", err)
	}

	return equal, toApply, localMap, toOverwrite, nil
}

// DiffSqlDbPolicy compares a local ResolvedSqlDb against the live Azure backup instance.
func DiffSqlDbPolicy(
	local azure.ResolvedSqlDb,
	remoteStr *armsql.BackupShortTermRetentionPolicy,
	remoteLtr *armsql.LongTermRetentionPolicy,
) (
	equal map[string]interface{},
	toApply map[string]interface{},
	localMap map[string]interface{},
	toOverwrite map[string]interface{},
	err error,
) {

	// 1. Create a single map from remote retention
	// 2. compare to local

	sqlItem, ok := remote.Properties.(*armrecoveryservicesbackup.AzureVMWorkloadSQLDatabaseProtectedItem)
	if !ok {
		return nil, nil, nil, nil, fmt.Errorf(
			"remote protected item is not an AzureVMWorkloadSQLDatabaseProtectedItem (got %T)",
			remote.Properties,
		)
	}

	remoteMap := map[string]any{
		"policyName": derefStringPtr(sqlItem.PolicyName),
	}

	localMap = resolvedSqlDbToMap(local)

	equal, toApply, toOverwrite, err = compareMaps(localMap, remoteMap)
	if err != nil {
		return nil, nil, nil, nil, fmt.Errorf("comparing SQL DB policy maps: %w", err)
	}

	return equal, toApply, localMap, toOverwrite, nil
}

// DiffSqlDbRetention compares the local AzureSqlDbSpec retention against the live
// Azure AzureVMWorkloadProtectionPolicy.
func DiffSqlDbRetention(
	local azure.ResolvedSqlDb,
	remote *armrecoveryservicesbackup.AzureVMWorkloadProtectionPolicy,
) (
	equal map[string]interface{},
	toApply map[string]interface{},
	localMap map[string]interface{},
	toOverwrite map[string]interface{},
	err error,
) {
	if remote == nil {
		return nil, nil, nil, nil, errors.New("remote SQL DB protection policy is nil")
	}

	localMap = sqlDbRetentionToMap(local)
	remoteMap, err := remoteSqlDbRetentionToMap(remote)
	if err != nil {
		return nil, nil, nil, nil, fmt.Errorf("parsing remote SQL DB retention policy: %w", err)
	}

	equal, toApply, toOverwrite, err = compareMaps(localMap, remoteMap)
	if err != nil {
		return nil, nil, nil, nil, fmt.Errorf("comparing SQL DB retention maps: %w", err)
	}

	return equal, toApply, localMap, toOverwrite, nil
}

// resolvedSqlDbToMap builds the local side of a SQL DB policy comparison map.
func resolvedSqlDbToMap(r azure.ResolvedSqlDb) map[string]any {
	return map[string]any{
		"policyName": r.PolicyName,
	}
}

// sqlDbRetentionToMap flattens the local AzureSqlDbSpec into a comparable key/value map.
func sqlDbRetentionToMap(r azure.ResolvedSqlDb) map[string]any {
	m := map[string]any{}
	spec := r.Spec

	m["str.retentionDays"] = int32(spec.ShortTermRetention.RetentionDays)
	m["str.differentialBackupFrequency"] = int32(spec.ShortTermRetention.DifferentialBackupFrequency)

	m["ltr.enabled"] = spec.LongTermRetention.WeeklyRetention > 0 ||
		spec.LongTermRetention.MonthlyRetention > 0 ||
		spec.LongTermRetention.YearlyRetention > 0

	if spec.LongTermRetention.WeeklyRetention > 0 {
		m["ltr.weeklyRetention"] = int32(spec.LongTermRetention.WeeklyRetention)
	}
	if spec.LongTermRetention.MonthlyRetention > 0 {
		m["ltr.monthlyRetention"] = int32(spec.LongTermRetention.MonthlyRetention)
	}
	if spec.LongTermRetention.YearlyRetention > 0 {
		m["ltr.yearlyRetention"] = int32(spec.LongTermRetention.YearlyRetention)
		m["ltr.weekOfYear"] = int32(spec.LongTermRetention.WeekOfYear)
	}

	// Apply per-target overrides if present
	if r.Overrides != nil {
		if r.Overrides.ShortTermRetention != nil {
			if r.Overrides.ShortTermRetention.RetentionDays > 0 {
				m["str.retentionDays"] = int32(r.Overrides.ShortTermRetention.RetentionDays)
			}
			if r.Overrides.ShortTermRetention.DifferentialBackupFrequency > 0 {
				m["str.differentialBackupFrequency"] = int32(r.Overrides.ShortTermRetention.DifferentialBackupFrequency)
			}
		}
		if r.Overrides.LongTermRetention != nil {
			o := r.Overrides.LongTermRetention
			if o.WeeklyRetention != "" {
				m["ltr.weeklyRetention"] = o.WeeklyRetention
			}
			if o.MonthlyRetention != "" {
				m["ltr.monthlyRetention"] = o.MonthlyRetention
			}
			if o.YearlyRetention != "" {
				m["ltr.yearlyRetention"] = o.YearlyRetention
				m["ltr.weekOfYear"] = int32(o.WeekOfYear)
			}
		}
	}

	return m
}

// remoteSqlDbRetentionToMap flattens an Azure AzureVMWorkloadProtectionPolicy into
// the same key shape as sqlDbRetentionToMap so compareMaps can work against them.
func remoteSqlDbRetentionToMap(remote *armrecoveryservicesbackup.AzureVMWorkloadProtectionPolicy) (map[string]any, error) {
	m := make(map[string]any)

	for _, rule := range remote.SubProtectionPolicy {
		if rule == nil || rule.PolicyType == nil {
			continue
		}
		switch *rule.PolicyType {
		case armrecoveryservicesbackup.PolicyTypeLog:
			// not tracked
		case armrecoveryservicesbackup.PolicyTypeFull:
			if rule.RetentionPolicy != nil {
				ltr, ok := rule.RetentionPolicy.(*armrecoveryservicesbackup.SimpleRetentionPolicy)
				if ok && ltr.RetentionDuration != nil {
					m["str.retentionDays"] = derefInt32(ltr.RetentionDuration.Count)
				}
			}
		case armrecoveryservicesbackup.PolicyTypeDifferential:
			if rule.SchedulePolicy != nil {
				sp, ok := rule.SchedulePolicy.(*armrecoveryservicesbackup.SimpleSchedulePolicy)
				if ok && sp.ScheduleRunFrequency != nil {
					freq := int32(0)
					if *sp.ScheduleRunFrequency == armrecoveryservicesbackup.ScheduleRunTypeHourly {
						freq = 12
					} else {
						freq = 24
					}
					m["str.differentialBackupFrequency"] = freq
				}
			}
		}
	}

	return m, nil
}

// DiffDiskRetention compares the local AzureDiskRetention spec against the live
// Azure BackupPolicy retention rules.
func DiffDiskRetention(
	local azure.ResolvedDisk,
	remote *armdataprotection.BackupPolicy,
) (
	equal map[string]interface{},
	toApply map[string]interface{},
	localMap map[string]interface{},
	toOverwrite map[string]interface{},
	err error,
) {
	if remote == nil {
		return nil, nil, nil, nil, errors.New("remote backup policy is nil")
	}

	localMap = diskRetentionToMap(local)
	remoteMap, err := remoteDiskRetentionToMap(remote)
	if err != nil {
		return nil, nil, nil, nil, fmt.Errorf("parsing remote disk retention policy: %w", err)
	}

	equal, toApply, toOverwrite, err = compareMaps(localMap, remoteMap)
	if err != nil {
		return nil, nil, nil, nil, fmt.Errorf("comparing disk retention maps: %w", err)
	}

	return equal, toApply, localMap, toOverwrite, nil
}

// resolvedDiskToMap builds the local side of a disk policy comparison map.
func resolvedDiskToMap(r azure.ResolvedDisk) map[string]any {
	return map[string]any{
		"policyID": fmt.Sprintf(
			"/subscriptions/%s/resourceGroups/%s/providers/Microsoft.DataProtection/backupVaults/%s/backupPolicies/%s",
			r.SubscriptionID, r.Resource.Vault.ResourceGroup, r.VaultName, r.PolicyName,
		),
	}
}

// diskRetentionToMap flattens the local AzureDiskRetention into a comparable key/value map.
func diskRetentionToMap(r azure.ResolvedDisk) map[string]any {
	m := map[string]any{}
	ret := r.Spec.Retention

	for _, rule := range ret.RetentionRules {
		prefix := "retention." + rule.RetentionRule
		m[prefix+".enabled"] = true
		m[prefix+".duration"] = rule.RetentionDuration
	}

	return m
}

// remoteDiskRetentionToMap flattens an Azure BackupPolicy's retention rules into
// the same key shape as diskRetentionToMap so compareMaps can work against them.
func remoteDiskRetentionToMap(remote *armdataprotection.BackupPolicy) (map[string]any, error) {
	m := make(map[string]any)

	for _, rule := range remote.PolicyRules {
		retRule, ok := rule.(*armdataprotection.AzureRetentionRule)
		if !ok {
			continue
		}
		if retRule.Name == nil {
			continue
		}
		name := *retRule.Name
		prefix := "retention." + name

		m[prefix+".enabled"] = true

		if len(retRule.Lifecycles) > 0 && retRule.Lifecycles[0].DeleteAfter != nil {
			deleteAfter, ok := retRule.Lifecycles[0].DeleteAfter.(*armdataprotection.AbsoluteDeleteOption)
			if !ok || deleteAfter.Duration == nil {
				continue
			}
			duration, err := parseISO8601Duration(*deleteAfter.Duration)
			if err != nil {
				return nil, fmt.Errorf("parsing duration for rule %q: %w", name, err)
			}
			m[prefix+".duration"] = duration
		}
	}

	return m, nil
}

// parseISO8601Duration converts an ISO 8601 duration string (e.g. "P7D", "P2W")
// into an integer number of days or weeks to match the local retention duration format.
func parseISO8601Duration(s string) (int, error) {
	if len(s) < 2 || s[0] != 'P' {
		return 0, fmt.Errorf("invalid ISO 8601 duration: %q", s)
	}
	var value int
	_, err := fmt.Sscanf(s[1:len(s)-1], "%d", &value)
	if err != nil {
		return 0, fmt.Errorf("could not parse duration value from %q: %w", s, err)
	}
	return value, nil
}
