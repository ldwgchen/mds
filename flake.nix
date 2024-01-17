{
  description = "";

  inputs = {
    nixpkgs.url = "github:NixOS/nixpkgs/nixos-unstable";
  };

  outputs = inputs@{ self, flake-parts, ... }:
    flake-parts.lib.mkFlake { inherit inputs; } {
      systems = [ "x86_64-linux" "aarch64-linux" "riscv64-linux" ];
      perSystem = { config, self', inputs', pkgs, system, ... }: {
        packages.default = pkgs.buildGoModule rec {
          pname = "mds";
          version = self.shortRev or self.dirtyShortRev;
          src = self;
          nativeBuildInputs = with pkgs; [ go-bindata ];
          preBuild = ''
            go-bindata data/
          '';
          ldflags = [
            "-s" "-w"
          ];
          # vendorHash = pkgs.lib.fakeHash;
          vendorHash = "sha256-bKXzL3wxtqFnh2RJEQPm2UAz3bzyDk6GiJGxtXamNFk=";
          meta = with pkgs.lib; {
            description = "A simple markdown server.";
            homepage = "https://github.com/ldwgchen/mds";
            license = licenses.mit;
          };
        };

        devShells.default = pkgs.mkShell {
          packages = with pkgs; [
            go_1_21
            gotools
            gopls
            go-tools  # staticcheck
          ];
          shellHook = ''
            go version
          '';
        };
      };
    };
}
