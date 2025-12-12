# Exit Codes Testing Guide

Quick reference for testing dev-cli failure detection with copy-paste commands.

---

## Universal Exit Codes

### Exit 0 - Success

```bash
true
echo hello
ls /tmp
```

### Exit 1 - General Error

```bash
false
grep notfound /etc/passwd
exit 1
```

### Exit 2 - Misuse of Command

```bash
ls --invalidflag
grep
cd ""
```

### Exit 126 - Permission Denied

```bash
touch /tmp/test.sh && chmod -x /tmp/test.sh && /tmp/test.sh
cat /etc/shadow
touch /root/test
```

### Exit 127 - Command Not Found

```bash
notarealcommand
npx nonexistent
./missing.sh
```

### Exit 130 - Ctrl+C (SIGINT)

```bash
sleep 100  # then press Ctrl+C
cat        # then press Ctrl+C
```

### Exit 137 - Killed (OOM)

```bash
kill -9 $$
```

### Exit 143 - Terminated (SIGTERM)

```bash
sleep 100 &
kill $!
```

---

## npm / Node.js

### Exit 1 - Generic Error

```bash
npm install nonexistent-pkg-xyz
node syntax_error.js
npm run missing-script
```

### ENOENT - Missing File

```bash
cd /tmp && npm install  # no package.json
node missing.js
```

### EACCES - Permission Issue

```bash
npm install -g some-package  # without sudo
```

---

## Git

### Exit 1 - General Failure

```bash
git checkout nonexistent-branch
git merge --abort  # when no merge in progress
```

### Exit 128 - Fatal Error

```bash
git clone invalid://url
git push  # when no remote configured
```

### Exit 129 - Invalid Options

```bash
git --invalidflag
git commit -x
```

---

## Docker

### Exit 1 - Container Error

```bash
docker run alpine exit 1
docker exec stopped_container ls
docker build .  # no Dockerfile
```

### Exit 125 - Daemon Error

```bash
docker run --invalid-flag alpine
docker run nonexistent:image
```

### Exit 137 - OOM Killed

```bash
docker run --memory=5m node -e "a=[];while(1)a.push(1)"
```

---

## Python

### Exit 1 - Exception

```bash
python -c "raise Exception()"
python missing.py
python -c "1/0"
```

### Exit 2 - CLI Error

```bash
python --badflag
python -c
python -m nonexistent
```

---

## Go

### Exit 1 - Build/Runtime Error

```bash
go build broken.go
go run missing.go
go test  # with failing tests
```

### Exit 2 - CLI Misuse

```bash
go build --badflag
go
go invalidcmd
```

---

## Prisma

### Exit 1 - General Error

```bash
prisma migrate dev  # invalid schema
prisma generate     # no schema
prisma db push      # connection fail
```

---

## dev-cli Current Capabilities

### View Logs

```bash
# All commands
cat ~/.devlogs/history.jsonl | jq

# Only failures (exit_code != 0)
cat ~/.devlogs/history.jsonl | jq 'select(.exit_code != 0)'

# Last 5 commands
tail -5 ~/.devlogs/history.jsonl | jq

# Commands from specific directory
cat ~/.devlogs/history.jsonl | jq 'select(.cwd == "/home/opx/Projects/dev-cli")'

# Slow commands (>1000ms)
cat ~/.devlogs/history.jsonl | jq 'select(.duration_ms > 1000)'

# View command output (for LLM analysis)
cat ~/.devlogs/history.jsonl | jq -r 'select(.output != "") | {cmd: .command, output: .output}'
```

### Manual Logging

```bash
# Without output
dev-cli log-event --command "test" --exit-code 1 --cwd "/tmp" --duration-ms 500

# With output (for LLM analysis)
dev-cli log-event --command "npm install" --exit-code 1 --cwd "/tmp" --duration-ms 500 --output "Error: ENOENT: no such file"
```

### Capture Output with dcap Wrapper

The hook provides a `dcap` function to capture command output:

```bash
# Use dcap prefix to capture output for any command
dcap npm install
dcap go build ./...
dcap docker build .
```

### Get Hook Script

```bash
dev-cli hook zsh
```

### Install Hook

```bash
# Temporary (current session)
eval "$(dev-cli hook zsh)"

# Permanent (add to ~/.zshrc)
echo 'eval "$(dev-cli hook zsh)"' >> ~/.zshrc
```

---

## Quick Test Script

```bash
#!/bin/bash
# Run multiple exit codes and check logs

echo "=== Testing dev-cli ==="

true
false
ls --badflag 2>/dev/null
notarealcmd 2>/dev/null

echo ""
echo "=== Last 4 Log Entries ==="
tail -4 ~/.devlogs/history.jsonl | jq -c '{cmd: .command, exit: .exit_code, ms: .duration_ms}'
```

Save as `e2e/test-exit-codes.sh`, then:

```bash
chmod +x e2e/test-exit-codes.sh && ./e2e/test-exit-codes.sh
```
