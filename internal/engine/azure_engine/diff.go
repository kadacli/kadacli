// internal/engine/azure_engine/diff.go
package azure_engine

import (
	"errors"
	"fmt"
	"maps"
	"reflect"
	"strconv"
	"strings"
	"time"

	"github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/dataprotection/armdataprotection"
	"github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/recoveryservices/armrecoveryservicesbackup"
	"github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/sql/armsql"
	"github.com/kadacli/kadacli/internal/models/azure"
)

// DiffVMPolicy compares a local ResolvedVM against the live Azure
// protected-item response. Returns four maps:
//   - equal:       fields that match between local and remote
//   - toApply:     fields in local that differ from remote (need to be applied)
//   - localMap:       the full local map
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
			"Remote protected item is not an AzureIaaSComputeVMProtectedItem (got %T)",
			remote.Properties,
		)
	}

	remoteMap := map[string]any{
		"policyName":      derefStringPtr(vmItem.PolicyName),
		"isInclusionList": derefIsInclusionList(vmItem.ExtendedProperties),
		"diskLunList":     derefDiskLunList(vmItem.ExtendedProperties),
	}

	localMap = ResolvedVMToMap(local)

	equal, toApply, toOverwrite, err = compareMaps(localMap, remoteMap)
	if err != nil {
		return nil, nil, nil, nil, fmt.Errorf("Comparing VM policy maps: %w", err)
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
		return nil, nil, nil, nil, errors.New("Remote retention policy is nil")
	}

	localMap = retentionToMap(local)
	remoteMap, err := remoteRetentionToMap(remote)
	if err != nil {
		return nil, nil, nil, nil, fmt.Errorf("Parsing remote retention policy: %w", err)
	}

	equal, toApply, toOverwrite, err = compareMaps(localMap, remoteMap)
	if err != nil {
		return nil, nil, nil, nil, fmt.Errorf("Comparing retention maps: %w", err)
	}

	return equal, toApply, localMap, toOverwrite, nil
}

// DiffVMSchedule compares the local FullSchedule spec against the live
// Azure SchedulePolicy (V1 *SimpleSchedulePolicy or V2 *SimpleSchedulePolicyV2).
func DiffVMSchedule(
	local azure.ResolvedVM,
	remote armrecoveryservicesbackup.SchedulePolicyClassification,
) (
	equal map[string]interface{},
	toApply map[string]interface{},
	localMap map[string]interface{},
	toOverwrite map[string]interface{},
	err error,
) {
	if remote == nil {
		return nil, nil, nil, nil, errors.New("Remote schedule policy is nil")
	}

	localMap = scheduleToMap(local)
	remoteMap, err := remoteScheduleToMap(remote)
	if err != nil {
		return nil, nil, nil, nil, fmt.Errorf("Parsing remote schedule policy: %w", err)
	}

	equal, toApply, toOverwrite, err = compareMaps(localMap, remoteMap)
	if err != nil {
		return nil, nil, nil, nil, fmt.Errorf("Comparing schedule maps: %w", err)
	}

	return equal, toApply, localMap, toOverwrite, nil
}

// DiffVMPolicyType compares the local policy_type (Standard/Enhanced) against
// the live Azure PolicyType (V1/V2). This isn't implied by any retention or
// schedule field diff — a V1-Daily and V2-Daily policy can have identical
// schedule/retention shapes while still being fundamentally different policy
// types, so it needs its own explicit check.
func DiffVMPolicyType(
	local azure.ResolvedVM,
	remote *armrecoveryservicesbackup.IAASVMPolicyType,
) (equal, toApply, overwrite map[string]any) {
	equal = map[string]any{}
	toApply = map[string]any{}
	overwrite = map[string]any{}

	localCode := "V1"
	if strings.EqualFold(local.Spec.PolicyType, "Enhanced") {
		localCode = "V2"
	}

	remoteCode := ""
	remoteLabel := ""
	if remote != nil {
		remoteCode = string(*remote)
		switch remoteCode {
		case "V1":
			remoteLabel = "Standard"
		case "V2":
			remoteLabel = "Enhanced"
		default:
			remoteLabel = remoteCode
		}
	}

	if remoteCode == "" || localCode == remoteCode {
		equal["policyType"] = local.Spec.PolicyType
		return
	}
	toApply["policyType"] = local.Spec.PolicyType
	overwrite["policyType"] = remoteLabel
	return
}

// ResolvedVMToMap builds the local side of a VM policy comparison map.
func ResolvedVMToMap(r azure.ResolvedVM) map[string]any {
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
	// Azure requires a WeeklySchedule on every VM retention policy; buildVMLongTermRetentionPolicy
	// injects a default (Sunday, 1 week) when retention.weekly is omitted, so the desired state here
	// must reflect that same default — otherwise this field would never match and every apply would
	// re-upsert the policy even when nothing actually changed.
	m["retention.weekly.enabled"] = true
	m["retention.monthly.enabled"] = ret.Monthly != nil
	m["retention.yearly.enabled"] = ret.Yearly != nil

	if ret.Daily != nil {
		m["retention.daily.count"] = int32(ret.Daily.Count)
	}
	if ret.Weekly != nil {
		m["retention.weekly.count"] = int32(ret.Weekly.Count)
		m["retention.weekly.day"] = ret.Weekly.Day
	} else {
		m["retention.weekly.count"] = int32(1)
		m["retention.weekly.day"] = "Sunday"
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

// scheduleToMap flattens the local FullSchedule into a comparable key/value map.
// Interval/duration are stored as int32 to match the remote representation
// returned by remoteScheduleToMap.
func scheduleToMap(r azure.ResolvedVM) map[string]any {
	sched := r.Spec.Schedule.Full

	m := map[string]any{
		"schedule.frequency": strings.ToLower(sched.Frequency),
		"schedule.time":      sched.ParsedTime.UTC().Format("15:04"),
	}

	switch sched.Frequency {
	case "weekly":
		m["schedule.day"] = sched.Day
	case "hourly":
		m["schedule.interval"] = int32(sched.Schedule)
		m["schedule.duration"] = int32(sched.Duration)
	}

	return m
}

// remoteScheduleToMap flattens an Azure SchedulePolicyClassification (V1 or V2)
// into the same key shape as scheduleToMap so compareMaps can work against them.
func remoteScheduleToMap(r armrecoveryservicesbackup.SchedulePolicyClassification) (map[string]any, error) {
	switch p := r.(type) {
	case *armrecoveryservicesbackup.SimpleSchedulePolicy:
		freq := ""
		if p.ScheduleRunFrequency != nil {
			freq = strings.ToLower(string(*p.ScheduleRunFrequency))
		}
		m := map[string]any{"schedule.frequency": freq}
		if len(p.ScheduleRunTimes) > 0 && p.ScheduleRunTimes[0] != nil {
			m["schedule.time"] = p.ScheduleRunTimes[0].UTC().Format("15:04")
		}
		if freq == "weekly" && len(p.ScheduleRunDays) > 0 && p.ScheduleRunDays[0] != nil {
			m["schedule.day"] = string(*p.ScheduleRunDays[0])
		}
		return m, nil

	case *armrecoveryservicesbackup.SimpleSchedulePolicyV2:
		freq := ""
		if p.ScheduleRunFrequency != nil {
			freq = strings.ToLower(string(*p.ScheduleRunFrequency))
		}
		m := map[string]any{"schedule.frequency": freq}

		switch freq {
		case "hourly":
			if p.HourlySchedule != nil {
				if p.HourlySchedule.Interval != nil {
					m["schedule.interval"] = *p.HourlySchedule.Interval
				}
				if p.HourlySchedule.ScheduleWindowDuration != nil {
					m["schedule.duration"] = *p.HourlySchedule.ScheduleWindowDuration
				}
				if p.HourlySchedule.ScheduleWindowStartTime != nil {
					m["schedule.time"] = p.HourlySchedule.ScheduleWindowStartTime.UTC().Format("15:04")
				}
			}
		case "daily":
			if p.DailySchedule != nil && len(p.DailySchedule.ScheduleRunTimes) > 0 && p.DailySchedule.ScheduleRunTimes[0] != nil {
				m["schedule.time"] = p.DailySchedule.ScheduleRunTimes[0].UTC().Format("15:04")
			}
		case "weekly":
			if p.WeeklySchedule != nil {
				if len(p.WeeklySchedule.ScheduleRunTimes) > 0 && p.WeeklySchedule.ScheduleRunTimes[0] != nil {
					m["schedule.time"] = p.WeeklySchedule.ScheduleRunTimes[0].UTC().Format("15:04")
				}
				if len(p.WeeklySchedule.ScheduleRunDays) > 0 && p.WeeklySchedule.ScheduleRunDays[0] != nil {
					m["schedule.day"] = string(*p.WeeklySchedule.ScheduleRunDays[0])
				}
			}
		}
		return m, nil

	default:
		return nil, fmt.Errorf("unsupported remote schedule policy type %T", r)
	}
}

// DiffDiskPolicy compares a local ResolvedDisk against the live Azure
// backup instance response. Returns four maps:
//   - equal:       fields that match between local and remote
//   - toApply:     fields in local that differ from remote (need to be applied)
//   - localMap:       the full local map
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
		return nil, nil, nil, nil, fmt.Errorf("Remote backup instance properties are nil")
	}

	remotePolicyID := ""
	if instance.PolicyInfo != nil && instance.PolicyInfo.PolicyID != nil {
		remotePolicyID = *instance.PolicyInfo.PolicyID
	}

	// Where snapshots land is a property of the backup instance, not the
	// policy, so it belongs in this comparison — without it, editing
	// snapshot_resource_group on a protected disk is a silent no-op.
	remoteSnapshotRGID := ""
	if instance.PolicyInfo != nil && instance.PolicyInfo.PolicyParameters != nil {
		for _, param := range instance.PolicyInfo.PolicyParameters.DataStoreParametersList {
			opParams, ok := param.(*armdataprotection.AzureOperationalStoreParameters)
			if ok && opParams.ResourceGroupID != nil {
				remoteSnapshotRGID = *opParams.ResourceGroupID
				break
			}
		}
	}

	remoteMap := map[string]any{
		"policyID":                remotePolicyID,
		"snapshotResourceGroupID": remoteSnapshotRGID,
	}

	localMap = resolvedDiskToMap(local)

	equal, toApply, toOverwrite, err = compareMaps(localMap, remoteMap)
	if err != nil {
		return nil, nil, nil, nil, fmt.Errorf("Comparing disk policy maps: %w", err)
	}

	return equal, toApply, localMap, toOverwrite, nil
}

// DiffSqlDbRetention compares the local AzureSqlDbSpec against the live STR and LTR policies.
func DiffSqlDbRetention(
	local azure.ResolvedSqlDb,
	remoteSTR *armsql.BackupShortTermRetentionPolicy,
	remoteLTR *armsql.LongTermRetentionPolicy,
	remoteDb *armsql.Database,
) (
	equal map[string]interface{},
	toApply map[string]interface{},
	localMap map[string]interface{},
	toOverwrite map[string]interface{},
	err error,
) {
	localMap = sqlDbRetentionToMap(local)
	remoteMap := remoteSqlDbRetentionToMap(remoteSTR, remoteLTR, remoteDb)

	equal, toApply, toOverwrite, err = compareMaps(localMap, remoteMap)
	if err != nil {
		return nil, nil, nil, nil, fmt.Errorf("Comparing SQL DB retention maps: %w", err)
	}

	return equal, toApply, localMap, toOverwrite, nil
}

func sqlDbRetentionToMap(r azure.ResolvedSqlDb) map[string]any {
	m := map[string]any{}
	str := r.Spec.ShortTermRetention
	ltr := r.Spec.LongTermRetention

	m["str.retentionDays"] = int32(str.RetentionDays)
	m["str.differentialBackupFrequency"] = int32(str.DifferentialBackupFrequency)

	m["ltr.weeklyRetention"] = ltr.WeeklyRetention
	m["ltr.monthlyRetention"] = ltr.MonthlyRetention
	m["ltr.yearlyRetention"] = ltr.YearlyRetention
	m["db.redundancy"] = r.Spec.Redundancy
	if ltr.YearlyRetention > 0 {
		m["ltr.weekOfYear"] = int32(ltr.WeekOfYear)
	}

	return m
}

func remoteSqlDbRetentionToMap(remoteSTR *armsql.BackupShortTermRetentionPolicy, remoteLTR *armsql.LongTermRetentionPolicy, remoteDb *armsql.Database) map[string]any {
	m := make(map[string]any)

	if remoteSTR != nil && remoteSTR.Properties != nil {
		m["str.retentionDays"] = derefInt32(remoteSTR.Properties.RetentionDays)
		m["str.differentialBackupFrequency"] = int32(derefDiffBackupInterval(remoteSTR.Properties.DiffBackupIntervalInHours))
	}

	if remoteLTR != nil && remoteLTR.Properties != nil {
		m["ltr.weeklyRetention"] = parseISO8601WeeksOrMonthsOrYears(derefStringPtr(remoteLTR.Properties.WeeklyRetention))
		m["ltr.monthlyRetention"] = parseISO8601WeeksOrMonthsOrYears(derefStringPtr(remoteLTR.Properties.MonthlyRetention))
		m["ltr.yearlyRetention"] = parseISO8601WeeksOrMonthsOrYears(derefStringPtr(remoteLTR.Properties.YearlyRetention))
		if derefInt32(remoteLTR.Properties.WeekOfYear) > 0 {
			m["ltr.weekOfYear"] = derefInt32(remoteLTR.Properties.WeekOfYear)
		}
	}

	if remoteDb != nil && remoteDb.Properties != nil && remoteDb.Properties.CurrentBackupStorageRedundancy != nil {
		m["db.redundancy"] = string(*remoteDb.Properties.CurrentBackupStorageRedundancy)
	}

	return m
}

func derefDiffBackupInterval(v *armsql.DiffBackupIntervalInHours) int {
	if v == nil {
		return 0
	}
	return int(*v)
}

// parseISO8601WeeksOrMonthsOrYears parses "P12W", "P12M", "P5Y" into their integer value.
func parseISO8601WeeksOrMonthsOrYears(s string) int {
	if len(s) < 2 || s[0] != 'P' {
		return 0
	}
	var value int
	fmt.Sscanf(s[1:len(s)-1], "%d", &value)
	return value
}

// DiffDiskPolicyDefinition compares the local disk spec — retention rules and
// schedule both, since Azure stores them in one policy definition — against
// the live Azure BackupPolicy.
func DiffDiskPolicyDefinition(
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
		return nil, nil, nil, nil, errors.New("Remote backup policy is nil")
	}

	localMap = mergeMaps(diskRetentionToMap(local), diskScheduleToMap(local))
	remoteMap, err := remoteDiskPolicyToMap(remote)
	if err != nil {
		return nil, nil, nil, nil, fmt.Errorf("Parsing remote disk policy: %w", err)
	}

	equal, toApply, toOverwrite, err = compareMaps(localMap, remoteMap)
	if err != nil {
		return nil, nil, nil, nil, fmt.Errorf("Comparing disk policy maps: %w", err)
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
		"snapshotResourceGroupID": r.SnapshotResourceGroupID(),
	}
}

// diskRetentionToMap flattens the local AzureDiskRetention into a comparable key/value map.
//
// Durations carry their unit ("1W", "7D") rather than a bare integer, because
// the Default rule's unit depends on which declared rule fed it — comparing
// integers alone would read "1 week" and "1 day" as the same value.
func diskRetentionToMap(r azure.ResolvedDisk) map[string]any {
	m := map[string]any{}
	ret := r.Spec.Retention

	// The Default rule governs every backup Azure takes, so it has to be
	// compared — and when no rules are declared it is the only retention that
	// exists, which is why an otherwise-valid policy used to diff against
	// nothing at all.
	m["retention.Default.enabled"] = true
	m["retention.Default.duration"] = strings.TrimPrefix(ret.DefaultISODuration(), "P")

	// Keyed by tag name, not the YAML rule name, so these line up with what
	// Azure returns — the remote rules are named after their tags.
	for _, rule := range ret.RetentionRules {
		prefix := "retention." + rule.TagName()
		m[prefix+".enabled"] = true
		m[prefix+".duration"] = strings.TrimPrefix(rule.ISODuration(), "P")
	}

	return m
}

// diskScheduleToMap flattens the local AzureDiskSchedule into the same key
// shape remoteDiskPolicyToMap produces. Only the keys that are meaningful for
// the frequency are emitted — schedule.time is invalid for hourly and
// schedule.interval is unused for daily — so an omitted field never compares
// against a remote value it has no say over.
func diskScheduleToMap(r azure.ResolvedDisk) map[string]any {
	sched := r.Spec.Schedule

	m := map[string]any{
		"schedule.frequency": sched.Frequency,
		"schedule.timezone":  r.Timezone,
	}

	switch sched.Frequency {
	case "hourly":
		m["schedule.interval"] = sched.Interval
	case "daily":
		m["schedule.time"] = sched.Time
	}

	return m
}

// mergeMaps combines two string-keyed maps into one, e.g. retention and
// schedule diff results that are displayed together under one section.
func mergeMaps(ms ...map[string]any) map[string]any {
	merged := make(map[string]any)
	for _, m := range ms {
		maps.Copy(merged, m)
	}
	return merged
}

// remoteDiskPolicyToMap flattens an Azure BackupPolicy's retention rules and
// backup schedule into the same key shape as diskRetentionToMap and
// diskScheduleToMap, so compareMaps can work against them.
func remoteDiskPolicyToMap(remote *armdataprotection.BackupPolicy) (map[string]any, error) {
	m := make(map[string]any)

	for _, rule := range remote.PolicyRules {
		if backupRule, ok := rule.(*armdataprotection.AzureBackupRule); ok {
			sched, err := remoteDiskScheduleToMap(backupRule)
			if err != nil {
				return nil, err
			}
			maps.Copy(m, sched)
			continue
		}

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
			if !strings.HasPrefix(*deleteAfter.Duration, "P") {
				return nil, fmt.Errorf("Invalid ISO 8601 duration %q for rule %q", *deleteAfter.Duration, name)
			}
			m[prefix+".duration"] = strings.TrimPrefix(*deleteAfter.Duration, "P")
		}
	}

	return m, nil
}

// remoteDiskScheduleToMap reads the schedule off an Azure backup rule. Azure
// stores it as an ISO 8601 repeating interval — "R/2023-01-01T04:00:00Z/P1D"
// for daily at 04:00, "R/2023-01-01T00:00:00Z/PT4H" for every 4 hours — plus a
// separate time zone field.
func remoteDiskScheduleToMap(rule *armdataprotection.AzureBackupRule) (map[string]any, error) {
	trigger, ok := rule.Trigger.(*armdataprotection.ScheduleBasedTriggerContext)
	if !ok || trigger.Schedule == nil {
		return nil, nil
	}

	m := make(map[string]any)

	// An absent time zone means Azure is running the schedule in UTC, which is
	// what an unset TimeZone field defaults to.
	timezone := "UTC"
	if trigger.Schedule.TimeZone != nil && *trigger.Schedule.TimeZone != "" {
		timezone = *trigger.Schedule.TimeZone
	}
	m["schedule.timezone"] = timezone

	if len(trigger.Schedule.RepeatingTimeIntervals) == 0 || trigger.Schedule.RepeatingTimeIntervals[0] == nil {
		return m, nil
	}

	interval := *trigger.Schedule.RepeatingTimeIntervals[0]
	parts := strings.Split(interval, "/")
	if len(parts) != 3 {
		return nil, fmt.Errorf("Unexpected repeating time interval %q", interval)
	}
	start, period := parts[1], parts[2]

	switch {
	case strings.HasPrefix(period, "PT") && strings.HasSuffix(period, "H"):
		hours, err := strconv.Atoi(strings.TrimSuffix(strings.TrimPrefix(period, "PT"), "H"))
		if err != nil {
			return nil, fmt.Errorf("Parsing hourly interval from %q: %w", interval, err)
		}
		m["schedule.frequency"] = "hourly"
		m["schedule.interval"] = hours
	case period == "P1D":
		t, err := time.Parse(time.RFC3339, start)
		if err != nil {
			return nil, fmt.Errorf("Parsing schedule start time from %q: %w", interval, err)
		}
		m["schedule.frequency"] = "daily"
		m["schedule.time"] = t.Format("15:04")
	default:
		return nil, fmt.Errorf("Unsupported schedule period %q in %q", period, interval)
	}

	return m, nil
}
