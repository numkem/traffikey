{ config, lib, pkgs, ... }:

with lib;
let
  cfg = config.services.traffikey;

  settings = {
    etcd.endpoints = cfg.etcdEndpoints;
    traefik = {
      default_entrypoint = cfg.defaultEntrypoint;
      default_prefix = cfg.defaultPrefix;
    };
    targets = (attrValues (mapAttrs (name: target: {
      inherit (target) entrypoint prefix rule tls;
      name = name;
      type = target.routerType;
      urls = target.serverUrls;
      middlewares = (attrValues (mapAttrs (name: middleware: {
        name = name;
        inherit (middleware) kind values;
      }) target.middlewares));
      tls_extra_keys = target.tlsExtraKeys;
    }) cfg.targets));
  };

  settingsFormat = pkgs.formats.json { };
  serverConfigFile = settingsFormat.generate "traffikey.json" settings;

  middlewareOptions = { ... }: {
    options = {
      kind = mkOption {
        type = types.str;
        description = mdDoc ''
          Kind of traefik middleware to use.
        '';
      };

      values = mkOption {
        type = types.attrsOf types.str;
        description = mdDoc ''
          Key values to add to the middleware, this is usually extra configuraitons toa apply to the middleware.
        '';
      };
    };
  };

  targetOptions = { ... }: {
    options = {
      serverUrls = mkOption {
        type = types.listOf types.str;
        description = mdDoc ''
          Full URL to the target server including the scheme (http or https).
        '';
      };

      routerType = mkOption {
        type = types.enum [ "http" "tcp" "udp" ];
        default = "http";
        description = mkDoc ''
          Type of traefik router for this target
        '';
      };

      entrypoint = mkOption {
        type = types.str;
        default = "web";
        description = ''
          Entrypoint name to use for the traefik router. Defaults to `web`.
        '';
      };

      middlewares = mkOption {
        type = types.attrsOf (types.submodule middlewareOptions);
        default = { };
        description = mdDoc ''
          List of middlewares to apply to this target.
        '';
      };

      prefix = mkOption {
        type = types.str;
        default = "";
        description = mdDoc ''
          Prefix for the etcd key, to target a specific traefik instance. `""` means the default one would be used.
        '';
      };

      rule = mkOption {
        type = types.str;
        description = mdDoc ''
          Traefik route rule to use for this routers.
        '';
        example = "Host(`some.example.com`)";
      };

      tls = mkOption {
        type = types.bool;
        default = false;
        description = mdDoc ''
          Use the TLS cert associated to the entrypoint or not.
        '';
      };

      tlsExtraKeys = mkOption {
        type = types.attrsOf (types.str);
        default = { };
        description = mdDoc ''
          Extra keys to add to the TLS configuration on the router
        '';
      };
    };
  };
in {
  options.services.traffikey = {
    enable = mkEnableOption ''
      Enable the traffikey service.

      It will parse it's configuration than write the proper required keys to etcd
    '';

    etcdEndpoints = mkOption {
      type = types.listOf types.str;
      default = [ "http://127.0.0.1:2379" ];
      description = mdDoc "Etcd endpoints to connect to";
    };

    targets = mkOption {
      type = types.attrsOf (types.submodule targetOptions);
      default = { };
      description = mdDoc ''
        Configuration for the traffikey service.
      '';
    };

    defaultEntrypoint = mkOption {
      type = types.str;
      default = "web";
      description = mdDoc ''
        Traefik entrypoint to use as default
      '';
    };

    defaultPrefix = mkOption {
      type = types.str;
      default = "traefik";
      description = mdDoc ''
        etcd key prefix (beginning of the key names)
      '';
    };
  };

  config = mkIf cfg.enable {
    systemd.services.traffikey = {
      description = "Traefik keymate service";
      restartIfChanged = true;
      serviceConfig.Type = "oneshot";

      script = ''
        ${pkgs.traffikey}/bin/traffikey apply --config ${serverConfigFile}
      '';

      wantedBy = [ "multi-user.target" ];
      after = [ "networking.target" ];
    };
  };
}
