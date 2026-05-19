package engine

import (
	"errors"
	"fmt"
	"reflect"

	"github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/recoveryservices/armrecoveryservicesbackup"
	"github.com/dbjorklund/backup-policy-as-code/internal/models"
)

// DiffPolicy compares a local ResolvedTarget VM policy against the live Azure
// protected-item response. Returns equal, toApply, complete, toOverwrite maps.
func DiffVMPolicy(
	local models.ResolvedTarget,
	remote armrecoveryservicesbackup.ProtectedItemsClientGetResponse,
) (
	map[string]interface{},
	map[string]interface{},
	map[string]interface{},
	map[string]interface{},
	error,
) {
	vmItem, ok := remote.Properties.(*armrecoveryservicesbackup.AzureIaaSComputeVMProtectedItem)
	if !ok {
		return nil, nil, nil, nil, fmt.Errorf("remote protected item is not an AzureIaaSComputeVMProtectedItem (got %T)", remote.Properties)
	}

	remotePolicyMap := map[string]any{
		"policyName":         derefStringPtr(vmItem.PolicyName),
		"diskLunList":        derefDiskLunList(vmItem.ExtendedProperties),
		"includeFutureDisks": !derefIsInclusionList(vmItem.ExtendedProperties),
	}

	localPolicyMap := resolvedTargetToMap(local)

	equalSettings, settingsToApply, settingsToOverwrite, err := compareMaps(localPolicyMap, remotePolicyMap)
	if err != nil {
		return nil, nil, nil, nil, fmt.Errorf("error comparing maps: %w", err)
	}

	return equalSettings, settingsToApply, localPolicyMap, settingsToOverwrite, nil
}

// DiffRetentionPolicy compares a local RetentionSpec against a remote
// LongTermRetentionPolicy. Returns equal, toApply, complete, toOverwrite maps.
func DiffRetentionPolicy(
	local models.RetentionSpec,
	remote *armrecoveryservicesbackup.LongTermRetentionPolicy,
) (
	map[string]interface{},
	map[string]interface{},
	map[string]interface{},
	map[string]interface{},
	error,
) {
	if remote == nil {
		return nil, nil, nil, nil, errors.New("remote retention policy is nil")
	}

	localMap := retentionSpecToMap(local)
	remoteMap, err := remoteRetentionToMap(remote)
	if err != nil {
		return nil, nil, nil, nil, fmt.Errorf("error parsing remote retention policy: %w", err)
	}

	equal, toApply, toOverwrite, err := compareMaps(localMap, remoteMap)
	if err != nil {
		return nil, nil, nil, nil, fmt.Errorf("error comparing retention maps: %w", err)
	}

	return equal, toApply, localMap, toOverwrite, nil
}

// retentionSpecToMap flattens a local RetentionSpec into a comparable key/value map.
func retentionSpecToMap(r models.RetentionSpec) map[string]any {
	return map[string]any{
		"retention.daily.count":   r.Daily.Count,
		"retention.weekly.count":  r.Weekly.Count,
		"retention.weekly.day":    string(r.Weekly.Day),
		"retention.monthly.count": r.Monthly.Count,
		"retention.yearly.count":  r.Yearly.Count,
		"retention.yearly.month":  string(r.Yearly.Month),
		"retention.yearly.day":    string(r.Yearly.Day),
		"retention.yearly.week":   string(r.Yearly.Week),
	}
}

// remoteRetentionToMap flattens an Azure LongTermRetentionPolicy into the same
// key shape as retentionSpecToMap so compareMaps can work against them.
func remoteRetentionToMap(r *armrecoveryservicesbackup.LongTermRetentionPolicy) (map[string]any, error) {
	m := make(map[string]any)

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

// DiffDBPolicy compares a local ResolvedTarget SQL policy against the live
// AzureVMWorkloadProtectionPolicy. It iterates over SubProtectionPolicy
// tiers (Full, Differential, Log) and compares each plus retention.
func DiffDBPolicy(
	local models.ResolvedTarget,
	remote *armrecoveryservicesbackup.AzureVMWorkloadProtectionPolicy,
) (
	map[string]interface{},
	map[string]interface{},
	map[string]interface{},
	map[string]interface{},
	error,
) {
	if remote == nil {
		return nil, nil, nil, nil, errors.New("remote DB policy is nil")
	}

	remotePolicyMap, err := remoteDBPolicyToMap(remote)
	if err != nil {
		return nil, nil, nil, nil, fmt.Errorf("error parsing remote DB policy: %w", err)
	}

	localPolicyMap := localDBPolicyFromResolved(local)

	equalSettings, settingsToApply, settingsToOverwrite, err := compareMaps(localPolicyMap, remotePolicyMap)
	if err != nil {
		return nil, nil, nil, nil, fmt.Errorf("error comparing maps: %w", err)
	}

	return equalSettings, settingsToApply, localPolicyMap, settingsToOverwrite, nil
}

// resolvedTargetToMap builds the local side of a VM comparison map.
func resolvedTargetToMap(rt models.ResolvedTarget) map[string]any {
	diskLunList := rt.Target.DiskSelection
	if diskLunList == nil {
		diskLunList = []int32{}
	}
	return map[string]any{
		"policyName":         rt.Policy.Metadata.Name,
		"diskLunList":        diskLunList,
		"includeFutureDisks": rt.Target.IncludeFutureDisks,
	}
}

// localDBPolicyFromResolved flattens the resolved policy's schedule+retention
// into a comparable key/value map for SQL-type targets.
func localDBPolicyFromResolved(rt models.ResolvedTarget) map[string]any {
	sched := rt.Policy.Spec.Schedule
	ret := rt.Policy.Spec.Retention

	m := map[string]any{}

	if sched.Full != nil {
		m["full.frequency"] = sched.Full.Frequency
		m["full.time"] = sched.Full.Time
	}
	if sched.Log != nil {
		m["log.frequency_minutes"] = sched.Log.FrequencyMinutes
	}
	if ret.Daily.Count != 0 {
		m["retention.full_days"] = ret.Daily.Count
	}
	if sched.Differential != nil && sched.Differential.Enabled {
		m["differential.enabled"] = true
	}

	return m
}

// remoteDBPolicyToMap flattens the Azure SubProtectionPolicy slice into the
// same key shape as localDBPolicyFromResolved so compareMaps can work against them.
func remoteDBPolicyToMap(remote *armrecoveryservicesbackup.AzureVMWorkloadProtectionPolicy) (map[string]any, error) {
	m := make(map[string]any)

	for _, sub := range remote.SubProtectionPolicy {
		if sub == nil || sub.PolicyType == nil {
			continue
		}

		switch *sub.PolicyType {

		case armrecoveryservicesbackup.PolicyTypeFull:
			if s, ok := sub.SchedulePolicy.(*armrecoveryservicesbackup.SimpleSchedulePolicy); ok {
				m["full.frequency"] = string(*s.ScheduleRunFrequency)
				if len(s.ScheduleRunTimes) > 0 && s.ScheduleRunTimes[0] != nil {
					m["full.time"] = s.ScheduleRunTimes[0].Format("15:04")
				}
			}
			if r, ok := sub.RetentionPolicy.(*armrecoveryservicesbackup.LongTermRetentionPolicy); ok {
				if r.DailySchedule != nil && r.DailySchedule.RetentionDuration != nil {
					m["retention.full_days"] = derefInt32(r.DailySchedule.RetentionDuration.Count)
				}
			}

		case armrecoveryservicesbackup.PolicyTypeDifferential:
			m["differential.enabled"] = true

		case armrecoveryservicesbackup.PolicyTypeLog:
			if s, ok := sub.SchedulePolicy.(*armrecoveryservicesbackup.LogSchedulePolicy); ok {
				m["log.frequency_minutes"] = int(derefInt32(s.ScheduleFrequencyInMins))
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

func derefWeekdays(days []*armrecoveryservicesbackup.DayOfWeek) []string {
	result := make([]string, 0, len(days))
	for _, d := range days {
		if d != nil {
			result = append(result, string(*d))
		}
	}
	return result
}
