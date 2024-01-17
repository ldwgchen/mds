# mds

*mds* is a simple markdown server for my personal use only.

## Usage

In the desired directory for serving, run:
```shell
nix run github:ldwgchen/mds
```

## For developers

In the same directory as *flake.nix*, build the default package specified in the flake:
```shell
nix build
```

After building, run *mds* with:
```shell
./result/bin/mds
```

Building and running can be achieved with a single command:
```shell
nix run
```

To activate the development environment:
```
nix develop
```
