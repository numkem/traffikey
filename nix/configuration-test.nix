{ config, lib, pkgs, ... }:
let
  traefikConf = pkgs.writeText "traefik.yaml" ''
    entryPoints:
      web:
        address: ":80"
      traefik:
        address: ":8080"
      ssh:
        address: ":2222"

    providers:
      etcd:
        endpoints:
          - "127.0.0.1:2379"
  '';

  nginxRoot = pkgs.stdenv.mkDerivation {
    name = "nginx-root";

    src = ./.;

    doBuild = false;
    doConfigure = false;
    doCheck = false;

    installPhase = ''
      mkdir $out
      echo "<html><body><h1>Hi this is nginx</h1></body></html>" > $out/index.html
    '';
  };
in {
  networking.firewall.allowedTCPPorts = [ 22 ];
  environment.systemPackages = with pkgs; [ etcd ];

  services.openssh = {
    enable = true;
    settings.PasswordAuthentication = true;
  };

  services.etcd = {
    enable = true;
    listenClientUrls = [ "http://127.0.0.1:2379" ];
  };

  services.nginx = {
    enable = true;
    defaultHTTPListenPort = 8181;
    virtualHosts."default" = {
      default = true;
      listen = [{
        addr = "0.0.0.0";
        port = 8181;
      }];
      root = nginxRoot;
    };
  };

  services.traefik = {
    enable = true;
    staticConfigFile = traefikConf;
  };

  services.traefik-keymate = {
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

  systemd.services.traefik-keymate.requires = [ "etcd.service" ];

  users.users = {
    admin = {
      isNormalUser = true;
      extraGroups = [ "wheel" ];
      password = "admin";
    };
  };

  virtualisation.vmVariant = {
    # following configuration is added only when building VM with build-vm
    virtualisation = {
      memorySize = 2048; # Use 2048MiB memory.
      cores = 2;
      graphics = false;
    };
  };

  security.sudo.wheelNeedsPassword = false;

  system.stateVersion = "23.11";
}
