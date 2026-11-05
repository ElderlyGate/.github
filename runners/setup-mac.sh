#!/usr/bin/env bash
# setup-mac.sh makes an Apple Silicon Mac a home for the org's CI runners: Tart virtual machines,
# one GitHub Actions runner in each, started again whenever the Mac's user logs in.
#
#   setup-mac.sh --name <machine> --group <runner group> --token <registration token> [--no-macos] [--force]
#
#   --name     a short name for this machine, e.g. burke: the VMs are <name>-linux and <name>-macos,
#              the runners carry the label <name> beside eg-linux / eg-macos
#   --group    the org's runner group the runners join, e.g. burke-macos-mini (it must exist)
#   --token    the registration token: org Settings > Actions > Runners > New self-hosted runner,
#              the value after --token in its ./config.sh line; valid for one hour, enough for both VMs
#   --no-macos only the Linux VM (the macOS VM needs about 50 GB of disk and 4 GB of RAM more)
#   --force    delete and build again a VM that already exists
#
# Run it as the Mac's own user, not with sudo. It asks for nothing else; it prints each step.
set -euo pipefail

ORG_URL=https://github.com/ElderlyGate
LINUX_IMAGE=ghcr.io/cirruslabs/ubuntu:24.04
MACOS_IMAGE=ghcr.io/cirruslabs/macos-tahoe-base:latest
CPU=4
MEMORY=4096
LINUX_DISK=60

name="" group="" token="" macos=true force=false
while [ $# -gt 0 ]; do
  case "$1" in
    --name) name=${2:-}; shift 2 ;;
    --group) group=${2:-}; shift 2 ;;
    --token) token=${2:-}; shift 2 ;;
    --no-macos) macos=false; shift ;;
    --force) force=true; shift ;;
    -h | --help) sed -n '2,17p' "$0" | sed 's/^# \{0,1\}//'; exit 0 ;;
    *) echo "unknown argument: $1 (see --help)" >&2; exit 2 ;;
  esac
done
[ -n "$name" ] && [ -n "$group" ] && [ -n "$token" ] || { echo "--name, --group and --token are needed (see --help)" >&2; exit 2; }
[[ "$name" =~ ^[a-z0-9][a-z0-9-]*$ ]] || { echo "--name is lowercase letters, digits and -: $name" >&2; exit 2; }

step() { printf '\n==> %s\n' "$*"; }
fail() { echo "setup-mac: $*" >&2; exit 1; }

step "checking this Mac"
[ "$(uname -s)" = Darwin ] || fail "this is not macOS"
[ "$(uname -m)" = arm64 ] || fail "Tart needs an Apple Silicon Mac (arm64), this is $(uname -m)"
[ "$(id -u)" != 0 ] || fail "run it as the Mac's user, not with sudo"
need=$([ "$macos" = true ] && echo 90 || echo 30)
free=$(df -g "$HOME" | awk 'NR==2 {print $4}')
[ "$free" -ge "$need" ] || fail "needs about $need GB free disk, has $free GB"
echo "macOS $(sw_vers -productVersion), $(sysctl -n hw.ncpu) cores, $(( $(sysctl -n hw.memsize) / 1073741824 )) GB RAM, $free GB free"

step "Tart and jq (Homebrew)"
command -v brew >/dev/null || fail "Homebrew is missing: install it from https://brew.sh, then run this again"
command -v tart >/dev/null || brew install cirruslabs/cli/tart
command -v jq >/dev/null || brew install jq
tart --version

runner_version=$(curl -fsSL https://api.github.com/repos/actions/runner/releases/latest | jq -r .tag_name | tr -d v)
[ -n "$runner_version" ] || fail "could not read the latest runner version from GitHub"
echo "runner $runner_version"

# wait_vm waits until the VM answers tart exec.
wait_vm() {
  for _ in $(seq 1 60); do
    tart exec "$1" true 2>/dev/null && return 0
    sleep 5
  done
  fail "$1 did not start in five minutes"
}

# has_vm reports whether a local VM of that name exists.
has_vm() { tart list --source local --format json | jq -e --arg vm "$1" '.[] | select(.Name == $vm)' >/dev/null; }

# fresh_vm clones image to vm unless it exists (then --force rebuilds it), sets its size and starts it.
fresh_vm() {
  local vm=$1 image=$2 disk=${3:-}
  if has_vm "$vm"; then
    if [ "$force" = true ]; then
      tart stop "$vm" 2>/dev/null || true
      tart delete "$vm"
    else
      echo "$vm already exists: kept (--force builds it again)"
    fi
  fi
  if ! has_vm "$vm"; then
    tart clone "$image" "$vm"
    tart set "$vm" --cpu "$CPU" --memory "$MEMORY"
    [ -z "$disk" ] || tart set "$vm" --disk-size "$disk"
  fi
  if ! tart list --format json | jq -e --arg vm "$vm" '.[] | select(.Name == $vm and .Running)' >/dev/null; then
    nohup tart run "$vm" --no-graphics >/dev/null 2>&1 &
  fi
  wait_vm "$vm"
}

# start_at_login writes a LaunchAgent that runs the VM whenever this user logs in, and keeps it running.
start_at_login() {
  local vm=$1 label="org.elderlygate.runner.$1" plist="$HOME/Library/LaunchAgents/org.elderlygate.runner.$1.plist"
  mkdir -p "$HOME/Library/LaunchAgents" "$HOME/Library/Logs"
  cat > "$plist" <<EOF
<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0">
<dict>
  <key>Label</key><string>$label</string>
  <key>ProgramArguments</key>
  <array><string>$(command -v tart)</string><string>run</string><string>$vm</string><string>--no-graphics</string></array>
  <key>RunAtLoad</key><true/>
  <key>KeepAlive</key><true/>
  <key>StandardOutPath</key><string>$HOME/Library/Logs/$label.log</string>
  <key>StandardErrorPath</key><string>$HOME/Library/Logs/$label.log</string>
</dict>
</plist>
EOF
  # The VM started by this script stops, so the LaunchAgent is the one that runs it from now on.
  tart stop "$vm" 2>/dev/null || true
  launchctl bootout "gui/$(id -u)/$label" 2>/dev/null || true
  launchctl bootstrap "gui/$(id -u)" "$plist"
  wait_vm "$vm"
  echo "$vm starts at login ($plist)"
}

step "Linux VM $name-linux"
fresh_vm "$name-linux" "$LINUX_IMAGE" "$LINUX_DISK"
tart exec "$name-linux" bash -s <<EOF
set -euo pipefail
export DEBIAN_FRONTEND=noninteractive
sudo apt-get update -qq
sudo apt-get install -y -qq git curl jq build-essential unzip xz-utils docker.io >/dev/null
sudo usermod -aG docker "\$USER"
if ! command -v gh >/dev/null; then
  sudo mkdir -p -m 755 /etc/apt/keyrings
  curl -fsSL https://cli.github.com/packages/githubcli-archive-keyring.gpg | sudo tee /etc/apt/keyrings/githubcli-archive-keyring.gpg >/dev/null
  echo "deb [arch=\$(dpkg --print-architecture) signed-by=/etc/apt/keyrings/githubcli-archive-keyring.gpg] https://cli.github.com/packages stable main" | sudo tee /etc/apt/sources.list.d/github-cli.list >/dev/null
  sudo apt-get update -qq && sudo apt-get install -y -qq gh >/dev/null
fi
# Node, as GitHub's own Linux machines have it: some workflows set up pnpm before Node.
if ! command -v node >/dev/null; then
  v=\$(curl -fsSL https://nodejs.org/dist/index.json | jq -r '[.[] | select(.lts != false)][0].version')
  curl -fsSL "https://nodejs.org/dist/\$v/node-\$v-linux-arm64.tar.xz" | sudo tar xJ -C /usr/local --strip-components=1
fi
mkdir -p ~/actions-runner && cd ~/actions-runner
if [ ! -x ./config.sh ]; then
  curl -fsSL -o r.tgz "https://github.com/actions/runner/releases/download/v$runner_version/actions-runner-linux-arm64-$runner_version.tar.gz"
  tar xzf r.tgz && rm r.tgz
  sudo ./bin/installdependencies.sh >/dev/null
fi
if [ -f .runner ]; then sudo ./svc.sh stop >/dev/null 2>&1 || true; sudo ./svc.sh uninstall >/dev/null 2>&1 || true; fi
./config.sh --url "$ORG_URL" --token "$token" --name "$name-linux" --labels "eg-linux,$name" --runnergroup "$group" --unattended --replace
sudo ./svc.sh install "\$USER" >/dev/null
sudo ./svc.sh start >/dev/null
echo "runner $name-linux: \$(systemctl is-active actions.runner.*.service)"
EOF
start_at_login "$name-linux"

if [ "$macos" = true ]; then
  step "macOS VM $name-macos"
  fresh_vm "$name-macos" "$MACOS_IMAGE"
  tart exec "$name-macos" bash -ls <<EOF
set -euo pipefail
brew install -q gh jq node >/dev/null
mkdir -p ~/actions-runner && cd ~/actions-runner
if [ ! -x ./config.sh ]; then
  curl -fsSL -o r.tgz "https://github.com/actions/runner/releases/download/v$runner_version/actions-runner-osx-arm64-$runner_version.tar.gz"
  tar xzf r.tgz && rm r.tgz
fi
if [ -f .runner ]; then ./svc.sh stop >/dev/null 2>&1 || true; ./svc.sh uninstall >/dev/null 2>&1 || true; fi
./config.sh --url "$ORG_URL" --token "$token" --name "$name-macos" --labels "eg-macos,$name" --runnergroup "$group" --unattended --replace
./svc.sh install >/dev/null
./svc.sh start >/dev/null
echo "runner $name-macos started"
EOF
  start_at_login "$name-macos"
fi

step "done"
echo "Runners $name-linux$([ "$macos" = true ] && echo " and $name-macos") are in the group $group."
echo "They run while this Mac is on and its user is logged in. A Mac that should always serve jobs:"
echo "  System Settings > Energy: prevent automatic sleeping, start up after a power failure;"
echo "  Users & Groups: log this user in automatically."
