# Purpose
The internal/providers/ folder maintains the functions that can get, apply and change settings for the resources for each vendor.

The workflow should be:
1. The cmd/ package handles the I/O
2. The local policy is parsed, validated, resolved, and translated into a remote policy in internal/engine/
3. internal/engine/executor.go figures out which vendor has to be applied to and then calls the relevant function in this folder.