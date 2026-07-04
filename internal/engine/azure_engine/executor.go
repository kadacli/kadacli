package azure_engine

// TODO: Move to external

// func CreateOrUpdateAzureBackupPolicy(
// 	ctx context.Context,
// 	client *armrecoveryservicesbackup.ProtectionPoliciesClient,
// 	cfg azure_models.AzureRecoveryVMPolicy,
// ) (string, error) {
// 	policy := armrecoveryservicesbackup.ProtectionPolicyResource{
// 		Properties: &armrecoveryservicesbackup.AzureIaaSVMProtectionPolicy{
// 			BackupManagementType:          to.Ptr("AzureIaasVM"),
// 			TimeZone:                      to.Ptr("UTC"),
// 			InstantRpRetentionRangeInDays: to.Ptr(cfg.InstantRPRetentionDays),
// 			InstantRPDetails: &armrecoveryservicesbackup.InstantRPAdditionalDetails{
// 				AzureBackupRGNamePrefix: to.Ptr(cfg.InstantRPResourceGroupPrefix),
// 				AzureBackupRGNameSuffix: to.Ptr(cfg.InstantRPResourceGroupSuffix),
// 			},
// 			SchedulePolicy: &armrecoveryservicesbackup.SimpleSchedulePolicy{
// 				SchedulePolicyType:   to.Ptr("SimpleSchedulePolicy"),
// 				ScheduleRunFrequency: to.Ptr(armrecoveryservicesbackup.ScheduleRunTypeDaily),
// 				ScheduleRunTimes:     []*time.Time{to.Ptr(cfg.BackupTime)},
// 			},
// 			RetentionPolicy: &armrecoveryservicesbackup.LongTermRetentionPolicy{
// 				RetentionPolicyType: to.Ptr("LongTermRetentionPolicy"),
// 				DailySchedule: &armrecoveryservicesbackup.DailyRetentionSchedule{
// 					RetentionTimes: []*time.Time{to.Ptr(cfg.BackupTime)},
// 					RetentionDuration: &armrecoveryservicesbackup.RetentionDuration{
// 						Count:        to.Ptr(cfg.Retention.DailyCount),
// 						DurationType: to.Ptr(armrecoveryservicesbackup.RetentionDurationTypeDays),
// 					},
// 				},
// 				WeeklySchedule: &armrecoveryservicesbackup.WeeklyRetentionSchedule{
// 					DaysOfTheWeek:  []*armrecoveryservicesbackup.DayOfWeek{to.Ptr(cfg.Retention.WeeklyDay)},
// 					RetentionTimes: []*time.Time{to.Ptr(cfg.BackupTime)},
// 					RetentionDuration: &armrecoveryservicesbackup.RetentionDuration{
// 						Count:        to.Ptr(cfg.Retention.WeeklyCount),
// 						DurationType: to.Ptr(armrecoveryservicesbackup.RetentionDurationTypeWeeks),
// 					},
// 				},
// 				MonthlySchedule: &armrecoveryservicesbackup.MonthlyRetentionSchedule{
// 					RetentionScheduleFormatType: to.Ptr(armrecoveryservicesbackup.RetentionScheduleFormatWeekly),
// 					RetentionScheduleWeekly: &armrecoveryservicesbackup.WeeklyRetentionFormat{
// 						DaysOfTheWeek: []*armrecoveryservicesbackup.DayOfWeek{to.Ptr(cfg.Retention.WeeklyDay)},
// 						WeeksOfTheMonth: []*armrecoveryservicesbackup.WeekOfMonth{
// 							to.Ptr(armrecoveryservicesbackup.WeekOfMonthFirst),
// 						},
// 					},
// 					RetentionTimes: []*time.Time{to.Ptr(cfg.BackupTime)},
// 					RetentionDuration: &armrecoveryservicesbackup.RetentionDuration{
// 						Count:        to.Ptr(cfg.Retention.MonthlyCount),
// 						DurationType: to.Ptr(armrecoveryservicesbackup.RetentionDurationTypeMonths),
// 					},
// 				},
// 				YearlySchedule: &armrecoveryservicesbackup.YearlyRetentionSchedule{
// 					RetentionScheduleFormatType: to.Ptr(armrecoveryservicesbackup.RetentionScheduleFormatWeekly),
// 					MonthsOfYear: []*armrecoveryservicesbackup.MonthOfYear{
// 						to.Ptr(cfg.Retention.YearlyMonth),
// 					},
// 					RetentionScheduleWeekly: &armrecoveryservicesbackup.WeeklyRetentionFormat{
// 						DaysOfTheWeek: []*armrecoveryservicesbackup.DayOfWeek{
// 							to.Ptr(cfg.Retention.YearlyDay),
// 						},
// 						WeeksOfTheMonth: []*armrecoveryservicesbackup.WeekOfMonth{
// 							to.Ptr(cfg.Retention.YearlyWeek),
// 						},
// 					},
// 					RetentionTimes: []*time.Time{to.Ptr(cfg.BackupTime)},
// 					RetentionDuration: &armrecoveryservicesbackup.RetentionDuration{
// 						Count:        to.Ptr(cfg.Retention.YearlyCount),
// 						DurationType: to.Ptr(armrecoveryservicesbackup.RetentionDurationTypeYears),
// 					},
// 				},
// 			},
// 		},
// 	}

// 	resp, err := client.CreateOrUpdate(ctx, cfg.VaultName, cfg.ResourceGroup, cfg.PolicyName, policy, nil)
// 	if err != nil {
// 		return "", fmt.Errorf("creating/updating backup policy %q: %w", cfg.PolicyName, err)
// 	}

// 	if resp.ID == nil {
// 		return "", fmt.Errorf("backup policy %q created but response contained no ID", cfg.PolicyName)
// 	}

// 	return *resp.ID, nil
// }
