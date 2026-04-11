package engine

func authenticate() []error {
	// TODO: Authenticate towards Azure with GO SDK
	return nil
}

func ApplyAzureVM() []error {
	// TODO: Check validity of policy_subtype, disk_selection, and include_future_disks
	// and then implement
	return nil
}

func GetAzureVault() []error {
	// TODO: Gets Azure vault from Go SDK(maybe id?)
	return nil
}

func GetAzureBackupPolicy() []error {
	// TODO: Check validity of backup_policy in Azure (that it exists?) get an string or id back, and apply it to target resource name
	return nil
}
