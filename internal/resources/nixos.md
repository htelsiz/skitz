# NixOS

`nixos-rebuild switch` build and activate system config ^run
`nixos-rebuild test` build and activate without bootloader ^run
`nixos-rebuild build` build without activating ^run
`nixos-rebuild boot` build and set as boot default ^run
`nixos-rebuild switch --flake .` rebuild from flake ^run
`nixos-rebuild switch --upgrade` rebuild and update channels ^run
`nix-channel --list` list channels ^run
`nix-channel --update` update channels ^run
`nix-env -qaP {{pkg}}` search packages ^run:pkg
`nix-env -iA nixos.{{pkg}}` install package (imperative) ^run:pkg
`nix-env -e {{pkg}}` uninstall package (imperative) ^run:pkg
`nix-env --list-generations` list generations ^run
`nix-env --rollback` rollback to previous generation ^run
`nix-collect-garbage -d` delete old generations and gc ^run
`nix-store --gc` garbage collect store ^run
`nix-store --optimise` deduplicate store ^run
`nix search nixpkgs {{pkg}}` search packages (flakes) ^run:pkg
`nix develop` enter dev shell ^run
`nix build` build flake output ^run
`nix flake update` update flake inputs ^run
`nix flake show` show flake outputs ^run
`nix flake check` check flake ^run
`nix run nixpkgs#{{pkg}}` run package without installing ^run:pkg
`nix shell nixpkgs#{{pkg}}` temp shell with package ^run:pkg
`nix profile list` list installed packages (new cli) ^run
`nix profile install nixpkgs#{{pkg}}` install package (new cli) ^run:pkg
`systemctl status {{service}}` check service status ^run:service
`journalctl -u {{service}} -f` follow service logs ^run:service
`nixos-option {{option}}` query config option ^run:option
