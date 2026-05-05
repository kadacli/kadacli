package engine

import (
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"reflect"

	"github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/recoveryservices/armrecoveryservicesbackup"
	"github.com/dbjorklund/backup-policy-as-code/internal/models"
)

// DiffPolicy compares a local VM BackupPolicy against the live Azure VM policy.
func DiffPolicy(local models.BackupPolicy, remote *armrecoveryservicesbackup.AzureIaaSVMProtectionPolicy) (
	map[string]interface{},
	map[string]interface{},
	map[string]interface{},
	map[string]interface{},
	error) {

	b, err := json.MarshalIndent(remote, "", "  ")
	if err != nil {
		log.Fatal(err)
	}

	remotePolicyMap, err := parseJson(b)
	if err != nil {
		return nil, nil, nil, nil, fmt.Errorf("error parsing response from Azure: %w", err)
	}

	localPolicyMap := backupPolicyToMap(local)

	equalSettings, settingsToApply, settingsToOverwrite, err := compareMaps(localPolicyMap, remotePolicyMap)
	if err != nil {
		return nil, nil, nil, nil, fmt.Errorf("error when comparing maps: %w", err)
	}

	return equalSettings, settingsToApply, localPolicyMap, settingsToOverwrite, nil
}

// DiffDBPolicy compares a local SQL VM BackupPolicy against the live Azure
// AzureVMWorkloadProtectionPolicy. It iterates over the SubProtectionPolicy
// slice (Full, Differential, Log) and compares each tier plus retention.
func DiffDBPolicy(local models.BackupPolicy, remote *armrecoveryservicesbackup.AzureVMWorkloadProtectionPolicy) (
	map[string]interface{},
	map[string]interface{},
	map[string]interface{},
	map[string]interface{},
	error) {

	if local.Spec.DBPolicy == nil {
		return nil, nil, nil, nil, errors.New("local policy has no db_policy block")
	}
	if remote == nil {
		return nil, nil, nil, nil, errors.New("remote DB policy is nil")
	}

	remotePolicyMap, err := remoteDBPolicyToMap(remote)
	if err != nil {
		return nil, nil, nil, nil, fmt.Errorf("error parsing remote DB policy: %w", err)
	}

	localPolicyMap := localDBPolicyToMap(local)

	equalSettings, settingsToApply, settingsToOverwrite, err := compareMaps(localPolicyMap, remotePolicyMap)
	if err != nil {
		return nil, nil, nil, nil, fmt.Errorf("error when comparing maps: %w", err)
	}

	return equalSettings, settingsToApply, localPolicyMap, settingsToOverwrite, nil
}

// localDBPolicyToMap flattens the local DBPolicy into a comparable key/value map.
// Keys are namespaced by tier (e.g. "full.time") to avoid collisions.
func localDBPolicyToMap(bp models.BackupPolicy) map[string]any {
	db := bp.Spec.DBPolicy
	m := map[string]any{
		"full.frequency":              db.Full.Frequency,
		"full.time":                   db.Full.Time,
		"full.weekdays":               db.Full.Weekdays,
		"log.frequency_minutes":       db.Log.FrequencyMinutes,
		"retention.full_days":         db.Retention.FullBackupDays,
		"retention.differential_days": db.Retention.DifferentialBackupDays,
		"retention.log_days":          db.Retention.LogBackupDays,
	}

	if db.Differential != nil && db.Differential.Enabled {
		m["differential.weekdays"] = db.Differential.Weekdays
		m["differential.time"] = db.Differential.Time
	}

	if db.Retention.LongTerm != nil {
		lt := db.Retention.LongTerm
		m["ltr.weekly_retention"] = lt.WeeklyRetention
		m["ltr.monthly_retention"] = lt.MonthlyRetention
		m["ltr.yearly_retention"] = lt.YearlyRetention
		m["ltr.week_of_year"] = lt.WeekOfYear
	}

	return m
}

// remoteDBPolicyToMap flattens the Azure SubProtectionPolicy slice into the
// same key shape as localDBPolicyToMap so compareMaps can work against them.
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
				m["full.weekdays"] = derefWeekdays(s.ScheduleRunDays)
			}
			if r, ok := sub.RetentionPolicy.(*armrecoveryservicesbackup.LongTermRetentionPolicy); ok {
				if r.DailySchedule != nil && r.DailySchedule.RetentionDuration != nil {
					m["retention.full_days"] = derefInt32(r.DailySchedule.RetentionDuration.Count)
				}
				m["ltr.weekly_retention"] = derefString(r.WeeklySchedule.RetentionTimes[0].Format("15:04"))
				if r.WeeklySchedule != nil && r.WeeklySchedule.RetentionDuration != nil {
					m["ltr.weekly_retention"] = fmt.Sprintf("P%dW", derefInt32(r.WeeklySchedule.RetentionDuration.Count))
				}
				if r.MonthlySchedule != nil && r.MonthlySchedule.RetentionDuration != nil {
					m["ltr.monthly_retention"] = fmt.Sprintf("P%dM", derefInt32(r.MonthlySchedule.RetentionDuration.Count))
				}
				if r.YearlySchedule != nil && r.YearlySchedule.RetentionDuration != nil {
					m["ltr.yearly_retention"] = fmt.Sprintf("P%dY", derefInt32(r.YearlySchedule.RetentionDuration.Count))
				}
			}

		case armrecoveryservicesbackup.PolicyTypeDifferential:
			if s, ok := sub.SchedulePolicy.(*armrecoveryservicesbackup.SimpleSchedulePolicy); ok {
				m["differential.weekdays"] = derefWeekdays(s.ScheduleRunDays)
				if len(s.ScheduleRunTimes) > 0 && s.ScheduleRunTimes[0] != nil {
					m["differential.time"] = s.ScheduleRunTimes[0].Format("15:04")
				}
			}
			if r, ok := sub.RetentionPolicy.(*armrecoveryservicesbackup.SimpleRetentionPolicy); ok {
				if r.RetentionDuration != nil {
					m["retention.differential_days"] = derefInt32(r.RetentionDuration.Count)
				}
			}

		case armrecoveryservicesbackup.PolicyTypeLog:
			if s, ok := sub.SchedulePolicy.(*armrecoveryservicesbackup.LogSchedulePolicy); ok {
				m["log.frequency_minutes"] = derefInt32(s.ScheduleFrequencyInMins)
			}
			if r, ok := sub.RetentionPolicy.(*armrecoveryservicesbackup.SimpleRetentionPolicy); ok {
				if r.RetentionDuration != nil {
					m["retention.log_days"] = derefInt32(r.RetentionDuration.Count)
				}
			}
		}
	}

	return m, nil
}

// compareMaps returns where mapOne (local) differs from mapTwo (remote).
func compareMaps(mapOne, mapTwo map[string]any) (
	equal map[string]any,
	diff map[string]any,
	overwritten map[string]any,
	err error) {

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

func backupPolicyToMap(bp models.BackupPolicy) map[string]any {
	return map[string]any{
		"resourceName":    bp.SpecialSpec.Target.Selectors.ResourceName,
		"policyName":      bp.SpecialSpec.StorageProvider.BackupPolicy,
		"resourceGroup":   bp.SpecialSpec.StorageProvider.Resource_group,
		"vaultName":       bp.SpecialSpec.StorageProvider.Vault,
		"diskLunList":     bp.SpecialSpec.StorageProvider.DiskSelection,
		"isInclusionList": bp.SpecialSpec.StorageProvider.IncludeFutureDisks,
	}
}

func parseJson(data []byte) (map[string]interface{}, error) {
	var item models.AzureVMBackupPolicy
	if err := json.Unmarshal(data, &item); err != nil {
		return nil, err
	}

	diskLuns := make([]int32, len(item.ExtendedProperties.DiskExclusionProperties.DiskLunList))
	for i, ptr := range item.ExtendedProperties.DiskExclusionProperties.DiskLunList {
		if ptr != nil {
			diskLuns[i] = *ptr
		}
	}

	return map[string]interface{}{
		"policyName":      item.PolicyName,
		"diskLunList":     diskLuns,
		"isInclusionList": item.ExtendedProperties.DiskExclusionProperties.IsInclusionList,
	}, nil
}

// --- pointer dereference helpers ---

func derefInt32(p *int32) int32 {
	if p == nil {
		return 0
	}
	return *p
}

func derefString(s string) string {
	return s
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

func allValuesNil(m map[string]interface{}) bool {
	for _, v := range m {
		if v != nil {
			return false
		}
	}
	return true
}
