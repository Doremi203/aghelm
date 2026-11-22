#!/usr/bin/env python3
"""Candidate/peer CLI and MCP contracts on disposable profiles.

Peer binary comparisons and MCP protocol negotiation are separate checks.
This runner does not invoke or mutate installed clients or real agent sessions.
"""
import argparse
from concurrent.futures import ThreadPoolExecutor
import hashlib
import json
import os
from pathlib import Path
import sqlite3
import subprocess
import sys
import tempfile
import threading
import time

from mcp_probe import probe


def isolated(home):
    home.mkdir(parents=True, exist_ok=True)
    env = dict(os.environ, HOME=str(home), XDG_CONFIG_HOME=str(home / '.config'),
               XDG_DATA_HOME=str(home / '.local/share'), XDG_STATE_HOME=str(home / '.local/state'))
    for key in ('TMUX', 'TMUX_PANE', 'TMUX_TMPDIR', 'AGHELM_SESSION_ID'):
        env.pop(key, None)
    return env


def profile(home):
    # isolated() sets XDG_CONFIG_HOME, which the binary honors on every OS.
    return home / '.config' / 'aghelm'


def run(candidate, work, peer):
    other = peer.resolve()
    digest = hashlib.sha256(other.read_bytes()).hexdigest()
    checks = []
    def cli(binary, home, *args, caller=None):
        env = isolated(home)
        if caller:
            env['AGHELM_SESSION_ID'] = caller
        result = subprocess.run([str(binary), *args], cwd=home, env=env, text=True,
                                capture_output=True, timeout=20)
        if result.returncode:
            raise RuntimeError(f'{args}: exit {result.returncode}: {result.stderr[-1000:]}')
        return result.stdout
    def initialize(binary, home):
        env = isolated(home)
        env['AGHELM_SESSION_ID'] = 'beefcafe'
        subprocess.run([str(binary), 'sessions'], cwd=home, env=env, capture_output=True, timeout=20)
        db = profile(home) / 'state.db'
        if not db.exists():
            raise AssertionError('bootstrap did not create disposable profile')
        with sqlite3.connect(db) as conn:
            conn.execute("INSERT OR IGNORE INTO sessions(id,name,tool,cwd,group_name,status,archived,created_at,last_status_at) VALUES('beefcafe','fixture','terminal',?,'','finished',0,?,?)", (str(home), int(time.time()), int(time.time())))

    for name, binary in [('candidate', candidate), ('peer', other)]:
        home = work / ('home-' + name)
        version = cli(binary, home, '--version')
        (work / (name + '-version.txt')).write_text(version)
        initialize(binary, home)
        cli(binary, home, 'sessions', caller='beefcafe')
        cli(binary, home, 'task', 'create', 'matrix fixture', '--json', caller='beefcafe')
        tasks = json.loads(cli(binary, home, 'task', 'list', '--json', caller='beefcafe'))
        (work / (name + '-tasks.json')).write_text(json.dumps(tasks, indent=2))
        if 'matrix fixture' not in json.dumps(tasks):
            raise AssertionError(name + ' task did not roundtrip')
        checks.append(name + ' CLI')
        for protocol in ('2024-11-05', '2025-03-26', '2025-06-18', '2025-11-25', '2026-01-01'):
            mcp_home = work / f'mcp-{name}-{protocol}'
            initialize(binary, mcp_home)
            result = probe(str(binary), str(mcp_home), protocol, 'list_sessions', caller='beefcafe')
            (work / f'{name}-mcp-{protocol}.json').write_text(json.dumps(result, indent=2))
            if not result.get('tool_count') or result.get('tools_error') or result.get('initialize_error') or 'error' in result.get('call', {}) or result.get('call', {}).get('result', {}).get('isError'):
                raise AssertionError(f'{name} MCP {protocol} failed')
            checks.append(name + ' MCP ' + protocol)
    for label, first, second in [('peer-first', other, candidate), ('candidate-first', candidate, other)]:
        home = work / ('shared-' + label)
        initialize(first, home)
        cli(first, home, 'task', 'create', 'first-version task', '--json', caller='beefcafe')
        cli(second, home, 'sessions', caller='beefcafe')
        cli(second, home, 'task', 'create', 'second-version task', '--json', caller='beefcafe')
        tasks = cli(first, home, 'task', 'list', '--json', caller='beefcafe')
        if 'first-version task' not in tasks or 'second-version task' not in tasks:
            raise AssertionError('mixed-version tasks did not roundtrip')
        checks.append(label + ' shared profile task roundtrip')

    home = work / 'concurrent-tasks'
    initialize(other, home)
    initialize(candidate, home)
    with sqlite3.connect(profile(home) / 'state.db') as conn:
        conn.execute("INSERT INTO sessions(id,name,tool,cwd,group_name,status,archived,created_at,last_status_at) VALUES('feedcafe','second-fixture','terminal',?,'','finished',0,?,?)", (str(home), int(time.time()), int(time.time())))
    actors = [('candidate', candidate, 'beefcafe'), ('peer', other, 'feedcafe')]
    create_pairs = [threading.Barrier(2) for _ in range(8)]
    def concurrent_create(item):
        name, binary, caller, index = item
        title = f'concurrent-{name}-{index}'
        create_pairs[index].wait(timeout=5)
        task = json.loads(cli(binary, home, 'task', 'create', title, '--json', caller=caller))
        if task.get('title') != title:
            raise AssertionError('concurrent task creation returned wrong title')
        return task
    jobs = [(name, binary, caller, index) for index in range(8) for name, binary, caller in actors]
    with ThreadPoolExecutor(max_workers=4) as pool:
        created = list(pool.map(concurrent_create, jobs))
    if len({task['id'] for task in created}) != len(jobs):
        raise AssertionError('concurrent task creation returned duplicate IDs')
    expected = {task['id']: task['title'] for task in created}
    for _, binary, caller in actors:
        visible = {task['id']: task['title'] for task in json.loads(cli(binary, home, 'task', 'list', '--json', caller=caller))}
        if any(visible.get(task_id) != title for task_id, title in expected.items()):
            raise AssertionError('mixed-version concurrent creates lost a task')
    races = []
    for index in range(6):
        task = created[index]
        ready = threading.Barrier(2)
        def claim(actor):
            name, binary, caller = actor
            env = isolated(home)
            env['AGHELM_SESSION_ID'] = caller
            ready.wait(timeout=5)
            result = subprocess.run([str(binary), 'task', 'claim', task['id'], '--json'],
                                    cwd=home, env=env, text=True, capture_output=True, timeout=20)
            return dict(binary=name, caller=caller, code=result.returncode, stdout=result.stdout, stderr=result.stderr)
        with ThreadPoolExecutor(max_workers=2) as pool:
            outcomes = list(pool.map(claim, actors))
        winners = [result for result in outcomes if result['code'] == 0]
        losers = [result for result in outcomes if result['code'] != 0]
        if len(winners) != 1 or len(losers) != 1 or 'already claimed' not in losers[0]['stderr']:
            raise AssertionError(f'mixed-version claim is not single-winner: {outcomes}')
        for _, binary, caller in actors:
            stored = next(row for row in json.loads(cli(binary, home, 'task', 'list', '--json', caller=caller)) if row['id'] == task['id'])
            if stored['state'] != 'in_progress' or stored['owner'] != winners[0]['caller']:
                raise AssertionError('claim readers disagree with winning caller')
        races.append(dict(task_id=task['id'], outcomes=outcomes))
    (work / 'concurrent-tasks.json').write_text(json.dumps(dict(created=created, claims=races), indent=2))
    checks.extend(['mixed-version concurrent task creation (16 writes)', 'mixed-version task claims (6 single-winner races)'])
    return dict(status='passed', peer_binary=str(other), peer_sha256=digest, checks=checks,
                gaps=['installed client extension acceptance', 'live SSH', 'mixed-version concurrent writes beyond tasks', 'exclusive authority'])


if __name__ == '__main__':
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument('--binary', type=Path, required=True)
    parser.add_argument('--peer-binary', type=Path, required=True, help='the binary the candidate is compared with')
    parser.add_argument('--artifacts', type=Path)
    args = parser.parse_args()
    work = args.artifacts or Path(tempfile.mkdtemp(prefix='am-matrix-'))
    work.mkdir(parents=True, exist_ok=True)
    started = time.monotonic()
    try:
        result = run(args.binary.resolve(), work.resolve(), args.peer_binary)
    except Exception as error:
        result = dict(status='failed', error=str(error))
    result['seconds'] = round(time.monotonic() - started, 2)
    (work / 'result.json').write_text(json.dumps(result, indent=2) + '\n')
    print(json.dumps(dict(result, artifacts=str(work))))
    sys.exit(0 if result['status'] == 'passed' else 1)
