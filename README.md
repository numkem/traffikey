 _____            __  __ _ _              
/__   \_ __ __ _ / _|/ _(_) | _____ _   _ 
  / /\/ '__/ _` | |_| |_| | |/ / _ \ | | |
 / /  | | | (_| |  _|  _| |   <  __/ |_| |
 \/   |_|  \__,_|_| |_| |_|_|\_\___|\__, |
                                    |___/ 

# Traffikey

While being a single binary so far, Traffikey is meant to be a suite of software aimed at helping manage keys values inserted into a store (currently etcd) for dynamically configuring Traefik.

## Current state

Traffikey is currently deployed in "production" in my homelab and managing HTTP and TCP routers.

### Features

Traffikey currently supports:
- All types of routers (HTTP, TCP, UDP).
- etcd for the KV store.
- Middlewares of all kinds
- TLS where the entrypoint uses a pre-defined cert/key.
- Different key prefixes (useful for multiple traefik instances on the same cluster, public and private).
- Different entrypoints (TCP, HTTP, HTTPS).

## Usage

### Concept

#### Target

A target for Traffikey is a pair of a traefik service and traefik router. It contains:
- Zero or more middleware(s).
- A rule so that traefik can target the router.
- A key prefix (`""` would use the default one).
- An entrypoint (`""` would use the default one).
- If the router is using TLS or not (only used for HTTP routers).
- A list of server urls (setup in load balancing way).
- A router type.

### Through configuration file

The easiest way to use Taffikey is to use it's configuration file directly. The following example shows a HTTP router and a TCP one.

``` json
{
  "etcd": {
    "endpoints": [
      "http://127.0.0.1:2379"
    ]
  },
  "targets": [
    {
      "entrypoint": "web",
      "middlewares": [
        {
          "kind": "stripprefix",
          "name": "prefix",
          "values": {
            "prefixes": "/path"
          }
        }
      ],
      "name": "path",
      "prefix": "",
      "rule": "Path(`/path/`)",
      "tls": false,
      "type": "http",
      "urls": [
        "127.0.0.1:8181"
      ]
    },
    {
      "entrypoint": "ssh",
      "middlewares": [],
      "name": "ssh",
      "prefix": "",
      "rule": "HostSNI(`*`)",
      "tls": false,
      "type": "tcp",
      "urls": [
        "127.0.0.1:22"
      ]
    }
  ],
  "traefik": {
    "default_entrypoint": "web",
    "default_prefix": "traefik"
  }
}
```

Once applied through `traffikey apply --config ./traffikey.json` would write to etcd these key/values:

```
traefik/http/middlewares/prefix/stripprefix/prefixes     /path
traefik/http/routers/path/entrypoints                    web
traefik/http/routers/path/middlewares                    prefix
traefik/http/routers/path/rule                           Path(`/path/`)
traefik/http/routers/path/service                        path
traefik/http/services/path/loadbalancer/servers/0/url    http://127.0.0.1:8181
traefik/tcp/routers/ssh/entrypoints                      ssh
traefik/tcp/routers/ssh/rule                             HostSNI(`*`)
traefik/tcp/routers/ssh/service                          ssh
traefik/tcp/services/ssh/loadbalancer/servers/0/address  127.0.0.1:22
```

### Through NixOS module

This project is a flake and can be imported into your own configurations. The NixOS modules will write the JSON configuration.

For example the above JSON configuration can be written this way in nix:

```nix
services.traffikey = {
  enable = true;
  etcdEndpoints = [ "http://127.0.0.1:2379" ];
  defaultEntrypoint = "web";
  defaultPrefix = "traefik";
  targets = {
    "ssh" = {
      serverUrls = [ "127.0.0.1:22" ];
      routerType = "tcp";
      rule = "HostSNI(`*`)";
      entrypoint = "ssh";
    };
    "path" = {
      rule = "Path(`/path/`)";
      serverUrls = [ "127.0.0.1:8181" ];
      middlewares = {
        prefix = {
          kind = "stripprefix";
          values.prefixes = "/path";
        };
      };
    };
  };
};
```

A full example virtual machine can be built on NixOS (`x86_64-linux`) by doing `make testvm`.
