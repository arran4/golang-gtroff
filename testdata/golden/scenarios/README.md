Binary snapshots for the shared integration scenarios. Filenames combine the
scenario subdirectory/name, one-based page number and encoding. TXTAR files
contain readable layout and diagnostic expectations; PNG/JPEG bytes live here.
Refresh only after inspecting changes:

    go test . -run '^TestIntegrationScenarios$' -update-integration

Ordinary tests decode, compare dimensions and compare exact encoded bytes.
