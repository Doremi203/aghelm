# Disposable compatibility checks

Build the candidate once, then run the separate matrix and slower process gate:

```sh
go build -o /tmp/aghelm-candidate .
python3 -m unittest discover -s tools/compatibility -p 'test_*.py'
python3 tools/compatibility/matrix.py --binary /tmp/aghelm-candidate --peer-binary /tmp/aghelm-peer
bash tools/compatibility/two_manager.sh /tmp/aghelm-candidate
```

The matrix compares the candidate with the binary given by `--peer-binary`,
for example a build of an earlier commit, and records the peer's path and
hash, results and gaps. It downloads nothing. CLI/MCP runs have isolated profiles; the sequential mixed-version task checks
share only their own disposable profile. Concurrent checks create 16 tasks
through both binaries and race six claims, asserting one winner and matching
stored ownership through both readers. They cover task writes, not every
mutation family. The runner
seeds a disposable caller identity, since caller discovery through process
ancestry must not authorize the developer's real session inside a test profile.

The process gate takes about 70 seconds, including real heartbeat aging and
reclamation. It starts the managers on PTYs as owned subprocesses so pause and
cleanup target exact PIDs. All tmux calls use absolute fixture socket paths.
Results and logs remain at the printed artifact directory on failure.

Passing these checks proves only the named local contracts. Installed client
extensions, mixed-version concurrent writes beyond tasks, in-flight delivery fencing and
real SSH remain separate rollout requirements. No production authority or schema
change is introduced by this harness.

`--peer-binary` is required. Copy the executable first if its installed path
could be replaced during the run; all profile state remains inside the
artifact directory.
