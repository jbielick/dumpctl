{ pkgs ? import <nixpkgs> { } }:
pkgs.mkShell {
  name = "promote";
  buildInputs = with pkgs; [
    go_1_17
    mysql-client
  ];
  shellHook = ''
    go mod download
  '';
}
