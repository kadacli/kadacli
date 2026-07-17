// internal/engine/azure_engine/diff.go
package azure_engine

import (
	"errors"
	"fmt"
	"reflect"
	"strings"

	"github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/dataprotection/armdataprotection"
	"github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/recoveryservices/armrecoveryservicesbackup"
	"github.com/dbjorklund/backup-policy-as-code/internal/models/azure"
)

// TODO: Implement disk, sqlVM, SqlDb
// TODO: Test the functionality for VMs

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

	localMap = ResolvedVMToMap(local)

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
		return nil, nil, nil, nil, errors.New("remote schedule policy is nil")
	}

	localMap = scheduleToMap(local)
	remoteMap, err := remoteScheduleToMap(remote)
	if err != nil {
		return nil, nil, nil, nil, fmt.Errorf("parsing remote schedule policy: %w", err)
	}

	equal, toApply, toOverwrite, err = compareMaps(localMap, remoteMap)
	if err != nil {
		return nil, nil, nil, nil, fmt.Errorf("comparing schedule maps: %w", err)
	}

	return equal, toApply, localMap, toOverwrite, nil
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
