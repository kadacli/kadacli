"""
cmd/init.go
The bpac init command. Asks the user a few questions (environment, provider, subscription ID) and scaffolds the full folder structure: policies/, _defaults/, and the two GitHub Actions workflows. Generates real working YAML, not placeholder comments. Gets a new user to a working setup in under 30 seconds.
Rule: init is purely generative — it creates files and exits. It never calls a provider or reads existing state.
"""