# Exit Codes Reference

Comprehensive list of exit codes for testing dev-cli failure detection.

---

## Universal Exit Codes

| Code    | Meaning              | Trigger Examples                                                    |
| ------- | -------------------- | ------------------------------------------------------------------- |
| **0**   | Success              | `true`, `echo hello`, `ls /tmp`                                     |
| **1**   | General error        | `false`, `grep notfound /etc/passwd`, `exit 1`                      |
| **2**   | Misuse of command    | `ls --invalidflag`, `grep`, `cd ""`                                 |
| **126** | Permission denied    | `./script.sh` (no +x), `/etc/shadow`, `touch /root/test`            |
| **127** | Command not found    | `notarealcommand`, `npx nonexistent`, `./missing.sh`                |
| **128** | Invalid exit code    | `exit -1`, `exit 999`, `exit abc`                                   |
| **130** | Ctrl+C (SIGINT)      | `sleep 100` then Ctrl+C, `cat` then Ctrl+C, `npm start` then Ctrl+C |
| **137** | Killed (SIGKILL/OOM) | `docker run --memory=10m stress`, `kill -9 $$`, memory-heavy script |
| **139** | Segfault (SIGSEGV)   | Run a buggy C program, corrupted binary, `kill -11 $$`              |
| **143** | Terminated (SIGTERM) | `kill <pid>`, `docker stop <container>`, `systemctl stop service`   |

---

## npm / Node.js

| Code       | Meaning          | Trigger Examples                                                                    |
| ---------- | ---------------- | ----------------------------------------------------------------------------------- |
| **1**      | Generic error    | `npm install nonexistent-pkg-xyz`, `node syntax_error.js`, `npm run missing-script` |
| **ENOENT** | Missing file     | `npm install` (no package.json), `node missing.js`, `npx -p nonexistent`            |
| **EACCES** | Permission issue | `npm install -g pkg` (no sudo), `node /root/app.js`, write to readonly              |

---

## Git

| Code    | Meaning         | Trigger Examples                                                                    |
| ------- | --------------- | ----------------------------------------------------------------------------------- |
| **1**   | General failure | `git checkout nonexistent`, `git merge --abort` (no merge), `git diff` (no changes) |
| **128** | Fatal error     | `git clone invalid://url`, `git push` (no remote), `git init /root/noperm`          |
| **129** | Invalid options | `git --invalidflag`, `git commit -x`, `git log --badopt`                            |

---

## Docker

| Code    | Meaning         | Trigger Examples                                                                                 |
| ------- | --------------- | ------------------------------------------------------------------------------------------------ |
| **1**   | Container error | `docker run alpine exit 1`, `docker exec stopped_container ls`, `docker build .` (no Dockerfile) |
| **125** | Daemon error    | `docker run --invalid-flag`, `docker run nonexistent:image`, daemon not running                  |
| **137** | OOM killed      | `docker run --memory=5m node -e "a=[];while(1)a.push(1)"`, memory limit exceeded                 |

---

## Python

| Code  | Meaning   | Trigger Examples                                                        |
| ----- | --------- | ----------------------------------------------------------------------- |
| **1** | Exception | `python -c "raise Exception()"`, `python missing.py`, `python -c "1/0"` |
| **2** | CLI error | `python --badflag`, `python -c`, `python -m nonexistent`                |

---

## Go

| Code  | Meaning             | Trigger Examples                                                |
| ----- | ------------------- | --------------------------------------------------------------- |
| **1** | Build/runtime error | `go build broken.go`, `go run missing.go`, `go test` (failures) |
| **2** | CLI misuse          | `go build --badflag`, `go`, `go invalidcmd`                     |

---

## Prisma

| Code  | Meaning       | Trigger Examples                                                                                         |
| ----- | ------------- | -------------------------------------------------------------------------------------------------------- |
| **1** | General error | `prisma migrate dev` (invalid schema), `prisma generate` (no schema), `prisma db push` (connection fail) |

---

## Quick Test Script

```bash
#!/bin/bash
# test-exit-codes.sh - Run to test dev-cli failure detection

echo "=== Testing Exit Codes ==="

echo -n "true (expect 0): "
true; echo $?

echo -n "false (expect 1): "
false; echo $?

echo -n "bad flag (expect 2): "
ls --badflag 2>/dev/null; echo $?

echo -n "not found (expect 127): "
notarealcmd 2>/dev/null; echo $?

echo "=== Done ==="
```

Run with: `chmod +x e2e/test-exit-codes.sh && ./e2e/test-exit-codes.sh`
