{ config, lib, pkgs, ... }:

with lib;
let
  cfg = config.services.traefik-keymate;

  settingsFormat = pkgs.formats.json { };
  serverConfigFile =
    settingsFormat.generate "traefik-keymate.json" cfg.settings;
in {
  options.services.traefik-keymate = {
    enable = mkEnableOption ''
      Enable the traefik-keymate service.

      It will parse it's configuration than write the proper required keys to etcd
    '';

    settings = mkOption {
      type = settingsFormat.type;
      default = { };
      example = {
        etcd = {
          endpoints = [ "127.0.0.1:2379" ];
          targets = [ ];
        };
      };
      description = mdDoc ''
        Configuration for the traefik-keymate service.
      '';
    };
  };

  config = mkIf cfg.enable {
    systemd.services.traefik-keymate = {
      description = "Traefik keymate service";
      restartIfChanged = true;
      serviceConfig.Type = "oneshot";

      script = ''
        ${pkgs.traefik-keymate}/bin/traefik-keymate -config ${serverConfigFile}
      '';

      wantedBy = [ "multi-user.target" ];
      after = [ "networking.target" ];
    };
  };
}
