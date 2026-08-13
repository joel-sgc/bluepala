{
  description = "Bluepala - a lightweight terminal-friendly BlueZ wrapper written in Go";

  inputs = {
    nixpkgs.url = "github:NixOS/nixpkgs/nixos-unstable";
    flake-utils.url = "github:numtide/flake-utils";
  };

  outputs = { self, nixpkgs, flake-utils }:
    flake-utils.lib.eachDefaultSystem (system:
      let
        pkgs = import nixpkgs { inherit system; };
      in
      {
        packages.default = pkgs.buildGoModule {
          pname = "bluepala";
          version = "0.1.0";

          src = ./.;

          vendorHash = "sha256-fwnn2N0b69CAPj3LFBYEmKbF/OP38wQeIsceozxyQ2U=";

          meta = with pkgs.lib; {
            description = "A lightweight terminal-friendly BlueZ wrapper written in Go";
            homepage = "https://github.com/joel-sgc/bluepala";
            license = licenses.wtfpl;
            mainProgram = "bluepala";
          };
        };

        devShells.default = pkgs.mkShell {
          buildInputs = with pkgs; [
            go_1_25
            gopls
            gotools
            go-tools
            dbus
            bluez
          ];

          shellHook = ''
            echo "bluepala dev shell — go $(go version)"
          '';
        };
      }
    );
}
