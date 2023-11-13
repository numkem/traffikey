{
  description = "Tool to write the keys for exposing traefik service";

  inputs = { nixpkgs.url = "github:NixOS/nixpkgs/nixos-unstable"; };

  outputs = { self, nixpkgs }: rec {
    packages.x86_64-linux = let
      system = "x86_64-linux";
      pkgs = import nixpkgs { inherit system; };
    in {
      traefik-keymate = pkgs.buildGoModule {
        pname = "traefik-keymate";
        version = "0.1.0";

        src = ./.;

        submodules = [ "server" ];

        vendorHash = "sha256-9ksyoevxfieIZJE8EVdgTloEHf5HX0A2MqF+ZAvUc1U=";

        postInstall = ''
          mv $out/bin/server $out/bin/traefik-keymate
        '';
      };
    };

    devShells.x86_64-linux.default = let
      system = "x86_64-linux";
      pkgs = import nixpkgs { inherit system; };
    in pkgs.mkShell { buildInputs = with pkgs; [ etcd go gopls ]; };

    overlays.default =
      (final: prev: { inherit (packages.x86_64-linux) traefik-keymate; });

    nixosConfigurations.test = let system = "x86_64-linux";
    in nixpkgs.lib.nixosSystem rec {
      inherit system;
      pkgs = import nixpkgs {
        inherit system;
        overlays = [ overlays.default ];
      };
      modules = [ nixosModules.default ./nix/configuration-test.nix ];
    };

    nixosModules.default = import ./nix/modules/default.nix;
  };
}
