#!/bin/bash

# ============================================================================
# Demo: put/get/commit workflow
# Sets up a fresh Git repo in testdata/example and runs grif commands to
# demonstrate the basic put, get, status, and commit workflow.
# Prerequisite: run 'make install' so grif is on your PATH.
# ============================================================================

set -euo pipefail

REPO_DIR="testdata/example"

cleanup() {
    rm -rf "${REPO_DIR}"
}

main() {
    # Ensure grif is available
    if ! command -v grif &>/dev/null; then
        echo "Error: grif not found on PATH. Run 'make install' first." >&2
        exit 1
    fi

    # Start fresh
    cleanup
    mkdir -p "${REPO_DIR}"

    cd "${REPO_DIR}"
    git init
    touch readme.md
    git add readme.md
    git commit --message "Add README"

    grif init default
    grif put default/network/vpc --data "10.0.0.0/16"
    grif put default/network/subnet --data "10.0.1.0/24"
    grif put default/compute/instance --data '{"type": "t3.micro"}'
    grif status default
    grif commit default --message "Set up initial infrastructure"
    grif get default/network
    grif get default/network/vpc
}

main "$@"
