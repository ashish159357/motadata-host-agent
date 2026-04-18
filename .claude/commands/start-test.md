Run the **test-environment-agent** for the current project.

Follow these steps strictly:

1. Read `CLAUDE.md`
2. Read `docs/environment/current-environment.md`
3. Extract:
   - application path
   - binary name
   - expected language
   - expected service (if available)

4. Perform pre-execution validation:
   - Fail immediately if `docs/environment/current-environment.md` does not exist
   - Fail immediately if `CLAUDE.md` does not exist
   - Fail immediately if application path does not exist
   - Fail immediately if executable is not found in the application path
   - Fail immediately if executable permission is missing

5. Navigate to the application directory

6. Ensure executable permission:
   ```bash
   chmod +x motadata-host-agent
