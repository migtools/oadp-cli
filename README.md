# sample-cli

A simple command-line interface (CLI) tool for demonstration purposes.

> This project aims to create a `kubectl` plugin CLI based on Velero, so that downloading the Velero CLI separately isn't required.

## Build and Install

1. **Build the CLI:**
   ```sh
   go build -o kubectl-velero
   ```

2. **Find the location of your `kubectl` binary:**
   - On Linux/macOS:
     ```sh
     which kubectl
     ```
   - On Windows (Command Prompt):
     ```cmd
     where kubectl
     ```

3. **Move the built binary to the same directory:**
   ```sh
   mv kubectl-velero /path/to/kubectl-directory/
   ```
   If you need root permissions, prepend `sudo`:
   ```sh
   sudo mv kubectl-velero /path/to/kubectl-directory/
   ```
   Replace `/path/to/kubectl-directory/` with the directory path from the previous step.

4. **Verify installation:**
   ```sh
   kubectl velero --help
   ```