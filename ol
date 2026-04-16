#!/bin/sh
BIN_DIR="$(cd "$(dirname "$0")" && pwd)"
if command -v systemd-run >/dev/null 2>&1; then
	exec systemd-run --quiet --scope -p Delegate=yes --setenv=OL_SYSTEMD=1 "${BIN_DIR}/ol-bin" "$@"
fi
echo "warning: systemd-run not found; cgroup_root must be set in config." >&2
exec "${BIN_DIR}/ol-bin" "$@"
