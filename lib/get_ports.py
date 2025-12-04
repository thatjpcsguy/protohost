#!/usr/bin/env python3
"""
Protohost Deploy - Port Allocation Script

Finds available ports for Docker Compose services or reuses existing ports
if containers are already running.
"""

import sys
import socket
import subprocess
import json
import os

def is_port_in_use(port):
    """Check if a port is currently in use on localhost."""
    with socket.socket(socket.AF_INET, socket.SOCK_STREAM) as s:
        return s.connect_ex(('localhost', port)) == 0

def get_running_ports(project_name):
    """
    Check if containers for this project are already running and return their ports.

    Returns:
        dict: {'WEB': port, 'MYSQL': port, 'REDIS': port} or None if not running
    """
    try:
        # List containers for this project
        cmd = ["docker", "compose", "-p", project_name, "ps", "--format", "json"]
        result = subprocess.run(cmd, capture_output=True, text=True)

        if result.returncode != 0 or not result.stdout.strip():
            return None

        # Parse JSON output (docker compose ps --format json returns a stream of objects)
        ports = {}
        for line in result.stdout.strip().split('\n'):
            if not line:
                continue
            try:
                container = json.loads(line)
                # Extract published ports
                # Format typically looks like: [{"HostIp":"0.0.0.0","HostPort":"3000"}]
                publishers = container.get('Publishers', [])
                if not publishers:
                    continue

                service = container.get('Service', '')
                for pub in publishers:
                    if service == 'web':
                        ports['WEB'] = int(pub['HostPort'])
                    elif service == 'mysql':
                        ports['MYSQL'] = int(pub['HostPort'])
                    elif service == 'redis':
                        ports['REDIS'] = int(pub['HostPort'])
            except (json.JSONDecodeError, KeyError, ValueError):
                continue

        # Only return if we found all three ports
        if 'WEB' in ports and 'MYSQL' in ports and 'REDIS' in ports:
            return ports

    except Exception:
        # If docker command fails, assume not running
        pass
    return None

def find_free_slot(base_web=3000, base_mysql=3306, base_redis=6379, max_slots=100):
    """
    Find the first free slot of ports starting from base values.

    Args:
        base_web: Base port for web service (default: 3000)
        base_mysql: Base port for MySQL service (default: 3306)
        base_redis: Base port for Redis service (default: 6379)
        max_slots: Maximum number of slots to check (default: 100)

    Returns:
        dict: {'WEB': port, 'MYSQL': port, 'REDIS': port}

    Raises:
        Exception: If no free ports found within max_slots
    """
    offset = 0
    while offset < max_slots:
        web = base_web + offset
        mysql = base_mysql + offset
        redis = base_redis + offset

        if (not is_port_in_use(web) and
            not is_port_in_use(mysql) and
            not is_port_in_use(redis)):
            return {'WEB': web, 'MYSQL': mysql, 'REDIS': redis}

        offset += 1

    raise Exception(f"No free ports found in first {max_slots} slots")

def main():
    if len(sys.argv) < 2:
        print("Usage: python3 get_ports.py <project_name>", file=sys.stderr)
        sys.exit(1)

    project_name = sys.argv[1]

    # Read optional base ports from environment
    base_web = int(os.environ.get('BASE_WEB_PORT', 3000))
    base_mysql = int(os.environ.get('BASE_MYSQL_PORT', 3306))
    base_redis = int(os.environ.get('BASE_REDIS_PORT', 6379))

    # 1. Check if containers are already running
    ports = get_running_ports(project_name)

    # 2. If not running, find new slot
    if not ports:
        ports = find_free_slot(base_web, base_mysql, base_redis)

    # Output space-separated ports: WEB MYSQL REDIS
    print(f"{ports['WEB']} {ports['MYSQL']} {ports['REDIS']}")

if __name__ == "__main__":
    main()
