{ pkgs ? import <nixpkgs> {} }:

pkgs.mkShell {
  buildInputs = [
    pkgs.go
    pkgs.ginkgo
    pkgs.safe
    pkgs.vault
  ];
}
