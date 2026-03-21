"""
cmd/apply.go
Runs plan, prompts for confirmation, calls executor, writes state. The resolved policies it works with already have inheritance applied — apply never knows the difference.
Rule: always show plan before applying. Never skip confirmation silently.
"""