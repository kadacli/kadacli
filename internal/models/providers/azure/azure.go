package azure

// Contains azure specific values, that are general across all resources

type azureResource struct {
	subscriptionId string `yaml:"subscriptionId"`
	resourceKind   string `yaml:"resourceKind"` // disk/vm/sqlvm/sqlPaas etc
	resourceGroup  string `yaml:"resourceGroup"`
	vaultName      string `yaml:"vaultName"`
}
