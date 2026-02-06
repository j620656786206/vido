#!/bin/bash
#
# Cleanup Test Processes Script
#
# This script cleans up orphaned test server processes.
# Use --all to clean up all test processes, or --session <id> for a specific session.
#
# Usage:
#   ./scripts/cleanup-test-processes.sh          # Clean up processes from current session
#   ./scripts/cleanup-test-processes.sh --all    # Clean up ALL test processes (use with caution)
#   ./scripts/cleanup-test-processes.sh --list   # List test processes without killing
#

set -e

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

# Find test-related processes
find_test_processes() {
    local pids=""

    # Go backend processes
    pids+=$(pgrep -f "go run.*cmd/api" 2>/dev/null | tr '\n' ' ' || true)

    # Vite dev server processes
    pids+=$(pgrep -f "vite.*serve|nx.*serve.*web" 2>/dev/null | tr '\n' ' ' || true)

    # Playwright test processes
    pids+=$(pgrep -f "playwright.*test" 2>/dev/null | tr '\n' ' ' || true)

    # Node processes on test ports
    for port in 8080 4200; do
        local port_pids=$(lsof -i :$port -t 2>/dev/null | tr '\n' ' ' || true)
        pids+="$port_pids"
    done

    # Return unique PIDs
    echo "$pids" | tr ' ' '\n' | sort -u | grep -v '^$' | tr '\n' ' '
}

# Kill a process gracefully
kill_graceful() {
    local pid=$1
    if kill -0 "$pid" 2>/dev/null; then
        echo -e "  ${YELLOW}Sending SIGTERM to PID $pid${NC}"
        kill -TERM "$pid" 2>/dev/null || true

        # Wait for process to exit
        for i in {1..20}; do
            if ! kill -0 "$pid" 2>/dev/null; then
                echo -e "  ${GREEN}PID $pid exited${NC}"
                return 0
            fi
            sleep 0.1
        done

        # Force kill if still running
        if kill -0 "$pid" 2>/dev/null; then
            echo -e "  ${RED}Force killing PID $pid${NC}"
            kill -KILL "$pid" 2>/dev/null || true
        fi
    fi
}

# List mode
list_processes() {
    echo -e "${YELLOW}Test-related processes:${NC}"
    echo ""

    local pids=$(find_test_processes)

    if [ -z "$pids" ]; then
        echo -e "${GREEN}No test processes found${NC}"
        return 0
    fi

    for pid in $pids; do
        if kill -0 "$pid" 2>/dev/null; then
            local cmd=$(ps -p "$pid" -o command= 2>/dev/null | head -c 80)
            echo -e "  PID $pid: $cmd"
        fi
    done
}

# Main cleanup function
cleanup_all() {
    echo -e "${YELLOW}Cleaning up ALL test processes...${NC}"
    echo ""

    local pids=$(find_test_processes)

    if [ -z "$pids" ]; then
        echo -e "${GREEN}No test processes found${NC}"
        return 0
    fi

    for pid in $pids; do
        kill_graceful "$pid"
    done

    echo ""
    echo -e "${GREEN}Cleanup complete${NC}"
}

# Parse arguments
case "${1:-}" in
    --all)
        cleanup_all
        ;;
    --list)
        list_processes
        ;;
    --help|-h)
        echo "Usage: $0 [--all|--list|--help]"
        echo ""
        echo "Options:"
        echo "  --all   Clean up ALL test processes"
        echo "  --list  List test processes without killing"
        echo "  --help  Show this help message"
        echo ""
        echo "Without arguments, lists test processes."
        ;;
    *)
        list_processes
        ;;
esac
