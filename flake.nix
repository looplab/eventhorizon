{
  description = "Tailswan - a bridge between tailscale and swanctl";

  inputs = {
    nixpkgs.url = "github:NixOS/nixpkgs/nixos-unstable";
    flake-utils.url = "github:numtide/flake-utils";
  };

  outputs = {
    nixpkgs,
    flake-utils,
    ...
  }:
    flake-utils.lib.eachDefaultSystem (system: let
      pkgs = nixpkgs.legacyPackages.${system};

      # Read and parse .tool-versions file
      toolVersionsContent = builtins.readFile ./.tool-versions;
      toolVersionsLines = builtins.filter (line: line != "") (pkgs.lib.strings.splitString "\n" toolVersionsContent);

      # Parse tool versions into an attribute set
      parseToolVersion = line: let
        parts = builtins.filter (x: x != "") (pkgs.lib.strings.splitString " " line);
        tool = builtins.elemAt parts 0;
        version = builtins.elemAt parts 1;
      in {
        name = tool;
        value = version;
      };

      toolVersions = builtins.listToAttrs (map parseToolVersion toolVersionsLines);

      # Extract major.minor versions for Nix package selection
      # Go version: 1.25.3 -> go_1_25
      goVersionParts = pkgs.lib.strings.splitString "." toolVersions.golang;
      goMajor = builtins.elemAt goVersionParts 0;
      goMinor = builtins.elemAt goVersionParts 1;
      go = pkgs."go_${goMajor}_${goMinor}";
    in {
      devShells.default = pkgs.mkShell {
        buildInputs =
          (with pkgs; [
            gnumake
            nodejs
            alejandra
            pre-commit
            golangci-lint
          ])
          ++ [go];
      };
    });
}
