{
  description = "Tool to write the keys for exposing traefik service";

  inputs = { nixpkgs.url = "github:NixOS/nixpkgs/nixos-unstable"; };

  outputs = { self, nixpkgs }:
    let
      supportedSystems = [ "x86_64-linux" "aarch64-linux" ];

      forAllSystems = nixpkgs.lib.genAttrs supportedSystems;
      nixpkgsFor = forAllSystems (system: import nixpkgs { inherit system; });
    in rec {
      packages = forAllSystems (system:
        let pkgs = nixpkgsFor.${system};
        in rec {
          traefik-keymate = pkgs.buildGoModule {
            pname = "traefik-keymate";
            version = "0.2.0";

            src = ./.;

            submodules = [ "server" ];

            vendorHash = "sha256-9ksyoevxfieIZJE8EVdgTloEHf5HX0A2MqF+ZAvUc1U=";

            postInstall = ''
              mv $out/bin/server $out/bin/traefik-keymate
            '';
          };

          default = traefik-keymate;
        });

      devShells = forAllSystems (system:
        let pkgs = nixpkgsFor.${system};
        in {
          default =
            pkgs.mkShell { buildInputs = with pkgs; [ etcd go gopls ]; };
        });

      overlays = forAllSystems (system:
        let pkgs = nixpkgsFor.${system};
        in {
          default =
            (final: prev: { inherit (packages.${system}) traefik-keymate; });
        });

      nixosConfigurations.test = let system = "x86_64-linux";
      in nixpkgs.lib.nixosSystem rec {
        inherit system;
        pkgs = import nixpkgs {
          inherit system;
          overlays = [ overlays.${system}.default ];
        };
        modules = [ nixosModules.default ./nix/configuration-test.nix ];
      };

      nixosModules.default = import ./nix/modules/default.nix;
    };
}
