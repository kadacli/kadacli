# Purpose
The models folder holds the internal representation of the policies, reports, and assignments. There are two levels of abstraction.

## 1. models/
The models folder has the first level of abstraction. These files (policy.go report.go, and models.go) contains the settings that are required for all providers. No "hard-coding" towards any one provider is allowed at this level.

## 2. models/providers/<provider>
In each provider folder there should be a file per resource that can be backed up. E.G: `models/providers/azure/azure_vm.go/azure_sql_vm_iaas.go/azure_sql_vm_paas.go/azure_disk.go` or `models/providers/aws/aws_ec2.go`