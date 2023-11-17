{
  description =
    "Set of services to manage traefik configuration with key/value stores";

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
          traffikey = pkgs.buildGoModule {
            pname = "traffikey";
            version = "0.2.0";

            src = ./.;

            submodules = [ "server" ];

            vendorHash = "sha256-9ksyoevxfieIZJE8EVdgTloEHf5HX0A2MqF+ZAvUc1U=";

            postInstall = ''
              mv $out/bin/server $out/bin/traffikey
            '';
          };

          default = traffikey;
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
          default = (final: prev: { inherit (packages.${system}) traffikey; });
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
