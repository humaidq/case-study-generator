{
  description = "Case Study Generator";
  inputs = {
    nixpkgs.url = "github:NixOS/nixpkgs/nixpkgs-unstable";
    flake-parts.url = "github:hercules-ci/flake-parts";
  };

  outputs = inputs @ {flake-parts, ...}:
    flake-parts.lib.mkFlake {inherit inputs;} {
      perSystem = {pkgs, ...}: {
        devShells.default = pkgs.mkShell {
          packages = with pkgs; [
            bashInteractive
            go
            chromium
            ibm-plex
          ];
        };
        packages.default = pkgs.buildGoModule {
          name = "case-study-gen";
          src = ./.;
          buildInputs = [
            pkgs.chromium
            pkgs.ibm-plex
          ];
          # Because vendor file exists, need to set to null
          vendorHash = null;
          meta = with pkgs.lib; {
            description = "Case Study Generator";
            homepage = "https://github.com/humaidq/case-study-gen";
            license = licenses.mit;
          };
        };
      };

      systems = [
        "x86_64-linux"
        "aarch64-linux"
      ];
    };
}
