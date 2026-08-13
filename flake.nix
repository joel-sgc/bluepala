{
  description = "Bluepala - a lightweight terminal-friendly BlueZ wrapper written in Go";

  inputs = {
    nixpkgs.url = "github:NixOS/nixpkgs/nixos-unstable";
    flake-utils.url = "github:numtide/flake-utils";
  };

  outputs =
    {
      self,
      nixpkgs,
      flake-utils,
    }:
    flake-utils.lib.eachDefaultSystem (
      system:
      let
        pkgs = nixpkgs.legacyPackages.${system};

        bluepala = pkgs.buildGoModule {
          pname = "bluepala";
          version = "1.0.0";

          src = pkgs.lib.cleanSourceWith {
            src = ./.;
            filter =
              path: type:
              let
                baseName = baseNameOf path;
              in
              # Exclude vendor directory, result symlinks, and other non-essential files
              !(
                baseName == "vendor"
                || baseName == "result"
                || pkgs.lib.hasPrefix "result-" baseName
                || baseName == ".direnv"
                || baseName == ".git"
                || baseName == "flake.lock"
              );
          };

          vendorHash = "sha256-fwnn2N0b69CAPj3LFBYEmKbF/OP38wQeIsceozxyQ2U=";

          meta = with pkgs.lib; {
            description = "A lightweight terminal-friendly BlueZ wrapper written in Go";
            homepage = "https://github.com/joel-sgc/bluepala";
            license = licenses.wtfpl;
            maintainers = [ ];
            platforms = platforms.linux;
            mainProgram = "bluepala";
          };

          ldflags = [
            "-s"
            "-w"
          ];

          # Runtime dependencies
          nativeBuildInputs = with pkgs; [
            pkg-config
          ];

          buildInputs = with pkgs; [
            dbus
          ];
        };
      in
      {
        packages = {
          default = bluepala;
          bluepala = bluepala;
        };

        apps = {
          default = flake-utils.lib.mkApp {
            drv = bluepala;
          };
          bluepala = flake-utils.lib.mkApp {
            drv = bluepala;
          };
        };

        devShells.default = pkgs.mkShell {
          buildInputs = with pkgs; [
            go
            gopls
            go-tools
            golangci-lint
            dbus
            bluez
            pkg-config
          ];

          shellHook = ''
            echo "Bluepala development environment"
            echo "Go version: $(go version)"
            echo ""
            echo "Available commands:"
            echo "  go build    - Build the application"
            echo "  go run .    - Run the application"
            echo "  go test     - Run tests"
            echo "  nix build   - Build with Nix"
          '';
        };
      }
    )
    // {
      # NixOS module for system-wide installation
      nixosModules.default =
        {
          config,
          lib,
          pkgs,
          ...
        }:
        let
          cfg = config.programs.bluepala;
        in
        {
          options.programs.bluepala = {
            enable = lib.mkEnableOption "bluepala bluetooth manager TUI";

            package = lib.mkOption {
              type = lib.types.package;
              default = self.packages.${pkgs.stdenv.hostPlatform.system}.default;
              description = "The bluepala package to use";
            };
          };

          config = lib.mkIf cfg.enable {
            environment.systemPackages = [ cfg.package ];

            # Ensure BlueZ and dbus are available
            services.dbus.enable = true;
            hardware.bluetooth.enable = lib.mkDefault true;
          };
        };

      # Home Manager module for per-user installation + config.toml generation
      homeManagerModules.default =
        {
          config,
          lib,
          pkgs,
          ...
        }:
        let
          cfg = config.programs.bluepala;
          tomlFormat = pkgs.formats.toml { };
        in
        {
          options.programs.bluepala = {
            enable = lib.mkEnableOption "bluepala bluetooth manager TUI";

            package = lib.mkOption {
              type = lib.types.package;
              default = self.packages.${pkgs.stdenv.hostPlatform.system}.default;
              description = "The bluepala package to use";
            };

            settings = lib.mkOption {
              type = tomlFormat.type;
              default = { };
              description = ''
                Configuration written to
                `$XDG_CONFIG_HOME/bluepala/config.toml`. Any keys you omit
                fall back to bluepala's built-in defaults. See
                `config/config.go` for the full set of `[colors]` and
                `[keybindings]` keys.
              '';
              example = lib.literalExpression ''
                {
                  colors = {
                    primary = "#a7abca";
                    active = "#9cca69";
                  };
                  keybindings = {
                    quit.keys = [ "q" "ctrl+c" ];
                    scan.keys = [ "s" ];
                  };
                }
              '';
            };
          };

          config = lib.mkIf cfg.enable {
            home.packages = [ cfg.package ];

            xdg.configFile."bluepala/config.toml" = lib.mkIf (cfg.settings != { }) {
              source = tomlFormat.generate "bluepala-config.toml" cfg.settings;
            };
          };
        };

      # Overlay for use with other flakes
      overlays.default = final: prev: {
        bluepala = self.packages.${prev.stdenv.hostPlatform.system}.default;
      };
    };
}
