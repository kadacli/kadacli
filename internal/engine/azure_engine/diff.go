// internal/engine/azure_engine/diff.go
package azure_engine

import (
	"errors"
	"fmt"
	"reflect"

	"github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/recoveryservices/armrecoveryservicesbackup"
	azure_models "github.com/dbjorklund/backup-policy-as-code/internal/models/azure_models"
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
	local azure_models.ResolvedVM,
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
		"policyName":         derefStringPtr(vmItem.PolicyName),
		"includeFutureDisks": !derefIsInclusionList(vmItem.ExtendedProperties),
		"diskLunList":        derefDiskLunList(vmItem.ExtendedProperties),
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
	local azure_models.ResolvedVM,
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
func resolvedVMToMap(r azure_models.ResolvedVM) map[string]any {
	return map[string]any{
		"policyName":         r.PolicyName,
		"includeFutureDisks": r.IncludeFutureDisks,
		"diskLunList":        []int32{},
	}
}

// retentionToMap flattens the local AzureRetention into a comparable key/value map.
func retentionToMap(r azure_models.ResolvedVM) map[string]any {
	m := map[string]any{}
	ret := r.Spec.Retention

	if ret.Daily != nil {
		m["retention.daily.count"] = ret.Daily.Count
	}
	if ret.Weekly != nil {
		m["retention.weekly.count"] = ret.Weekly.Count
		m["retention.weekly.day"] = ret.Weekly.Day
	}
	if ret.Monthly != nil {
		m["retention.monthly.count"] = ret.Monthly.Count
	}
	if ret.Yearly != nil {
		m["retention.yearly.count"] = ret.Yearly.Count
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
