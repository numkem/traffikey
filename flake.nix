{
  description = "Tool to write the keys for exposing traefik service";

  inputs = { nixpkgs.url = "github:NixOS/nixpkgs/nixos-unstable"; };

  outputs = inputs@{ flake-parts, ... }:
    flake-parts.lib.mkFlake { inherit inputs; } {
      imports = [ inputs.flake-parts.flakeModules.easyOverlay ];
      systems = [ "x86_64-linux" "aarch64-linux" ];
      perSystem = { config, self', inputs', pkgs, system, ... }: {
        # Per-system attributes can be defined here. The self' and inputs'
        # module parameters provide easy access to attributes of the same
        # system.
        overlayAttrs = { inherit (config.packages) traefik-keymate; };

        # Equivalent to  inputs'.nixpkgs.legacyPackages.hello;
        packages.traefik-keymate = pkgs.buildGoModule {
          pname = "traefik-keymate";
          version = "0.1.0";

          src = ./.;

          submodules = [ "server" ];

          vendorHash = "sha256-9ksyoevxfieIZJE8EVdgTloEHf5HX0A2MqF+ZAvUc1U=";

          postInstall = ''
            mv $out/bin/server $out/bin/traefik-keymate
          '';
        };

        devShells.default =
          pkgs.mkShell { buildInputs = with pkgs; [ etcd go gopls ]; };
      };
      flake = { nixosModules.default = import ./nix/modules/default.nix; };
    };
}
