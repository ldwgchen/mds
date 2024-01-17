# mds

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
