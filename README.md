# Docker Socket Proxy

Docker socket proxy using JSON files with regex lists for access control.
I use it for minimal traefik access to docker socket, can be expanded to other use cases. See this [docker-compose.yml](/examples/traefik.docker-compose.yml) example.

[![CI](https://github.com/andrmr/docker-socket-proxy/actions/workflows/ci.yml/badge.svg)](https://github.com/andrmr/docker-socket-proxy/actions/workflows/ci.yml) [![Publish and Release](https://github.com/andrmr/docker-socket-proxy/actions/workflows/release.yml/badge.svg)](https://github.com/andrmr/docker-socket-proxy/actions/workflows/release.yml)

## Example: Traefik Policy

The following policy allows discovery while blocking sensitive actions like `exec` or `logs`.

```json
{
  "groups": {
    "CONTAINERS": ["^/containers/json$", "^/containers/[a-f0-9]{12,64}/json$"],
    "SERVICES": ["^/services$", "^/services/[a-zA-Z0-9]{25}$", "^/services/[a-zA-Z0-9]{25}/json$"],
    "TASKS": ["^/tasks$", "^/tasks/[a-zA-Z0-9]{25}$"],
    "NETWORKS": ["^/networks$", "^/networks/[a-zA-Z0-9]{25}$"],
    "INFO": ["^/info$", "^/version$"],
    "EVENTS": ["^/events$"]
  },
  "global_deny": [
    "^/containers/[a-f0-9]{12,64}/attach",
    "^/containers/[a-f0-9]{12,64}/logs",
    "^/containers/[a-f0-9]{12,64}/exec",
    "^/exec/[a-zA-Z0-9]{25}/start"
  ]
}
```

### Access Rules
- **Allowed**: `GET` access to containers, services, tasks, networks, info, and events.
- **Blocked**: All non-`GET/HEAD` methods and specific paths like `/attach`, `/logs`, or `/exec`.

## License
MIT
