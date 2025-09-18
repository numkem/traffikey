{
  description = "Set of services to manage traefik configuration with key/value stores";

  inputs = {
    nixpkgs.url = "github:NixOS/nixpkgs/nixos-unstable";
    nixos-generators.url = "github:nix-community/nixos-generators";
  };

  outputs =
    { self, nixpkgs, nixos-generators, ... }:
    let
      supportedSystems = [
        "x86_64-linux"
        "aarch64-linux"
      ];

      forAllSystems = nixpkgs.lib.genAttrs supportedSystems;
      nixpkgsFor = forAllSystems (system: import nixpkgs { inherit system; });
    in
    rec {
      packages = forAllSystems (
        system:
        let
          pkgs = nixpkgsFor.${system};
        in
        {
          traffikey = pkgs.buildGoModule {
            pname = "traffikey";
            version = "0.4.0";

            src = ./.;

            submodules = [ "server" ];

            vendorHash = "sha256-fHWkG8qoYmajUNWnzy2QjEjyEZVKCah0uMB758drA5s=";

            doCheck = false;

            postInstall = ''
              mv $out/bin/server $out/bin/traffikey
            '';
          };

          dockerImage = pkgs.dockerTools.buildImage {
            name = "traffikey";
            tag = "latest";

            copyToRoot = pkgs.buildEnv {
              name = "image-root";
              paths = [ self.packages.${system}.traffikey ];
              pathsToLink = [ "/bin" ];
            };

            config = {
              Cmd = [ "/bin/traffikey" ];
            };

            diskSize = 1024;
            buildVMMemorySize = 512;
          };

          testVM = nixos-generators.nixosGenerate {
            inherit system;
            modules = [
              nixosModules.default
              ./nix/configuration-test.nix
              {
                nixpkgs.overlays = [ overlays.default ];
              }
            ];

            format = "vm";
          };

          default = self.packages.${system}.traffikey;
        }
      );

      devShells = forAllSystems (
        system:
        let
          pkgs = nixpkgsFor.${system};
        in
        {
          default = pkgs.mkShell {
            buildInputs = with pkgs; [
              etcd
              go
              gopls
              gotools
            ];
          };
        }
      );

      overlays.default = (
        final: prev: {
          inherit (packages.${final.system}) traffikey;
        }
      );

      nixosModules.default = import ./nix/modules/default.nix;
    };
}
