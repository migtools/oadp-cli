#!/bin/bash

# Build the kubectl-oadp plugin
go build -o kubectl-oadp .

# Move to system binary location
sudo mv kubectl-oadp /usr/local/bin/

echo "kubectl-oadp plugin installed successfully!"
echo "You can now use: kubectl oadp --help"
