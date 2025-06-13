{
  description = "protoc-gen-go-aip-test - Go protobuf plugin for Google API Improvement Proposals (AIP) compliance testing";

  inputs = {
    nixpkgs.url = "github:NixOS/nixpkgs/nixos-unstable";
    flake-utils.url = "github:numtide/flake-utils";
  };

  outputs = { self, nixpkgs, flake-utils }:
    {
      overlays.default = final: prev: {
        protoc-gen-go-aip-test = self.packages.${final.system}.default;
      };
    } // flake-utils.lib.eachDefaultSystem (system:
      let
        pkgs = nixpkgs.legacyPackages.${system};
      in {
        packages.default = pkgs.buildGoModule rec {
          pname = "protoc-gen-go-aip-test";
          version = "0.1.0";

          src = pkgs.lib.fileset.toSource {
            root = ./.;
            fileset = pkgs.lib.fileset.unions [
              ./main.go
              ./go.mod
              ./go.sum
              ./LICENSE
              ./internal
            ];
          };

          subPackages = [ "." ];

          vendorHash = "sha256-19cyx6C4rst5uMDXcjv34K7zd3j5jaLvko/oBJTTzp4=";
          proxyVendor = true;

          postInstall = ''
            install -Dm644 LICENSE $out/share/licenses/${pname}/LICENSE
          '';

          meta = with pkgs.lib; {
            description = "Go protobuf plugin for Google API Improvement Proposals (AIP) compliance testing";
            homepage = "https://github.com/bndry-co/protoc-gen-go-aip-test";
            license = licenses.mit;
            maintainers = [ ];
          };
        };

        devShells.default = pkgs.mkShell {
          buildInputs = with pkgs; [
            go
            protobuf
            protoc-gen-go
          ];
        };
      });
}
