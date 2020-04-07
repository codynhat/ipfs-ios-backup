ipfs-ios-backup
===============

Command line utility to backup iOS devices to an embedded and private IPFS node.

# Installation

## Homebrew

``` sh
brew tap codynhat/ipfs-ios-backup
brew install ipfs-ios-backup
```

## Building from source

### Requirements
- Go 1.14+
- libimobiledevice

### libimobiledevice

[libimobiledevice](http://www.libimobiledevice.org/) is a cross-platform library that enables communication with iOS devices. The main project has been forked to include a few more features needed for `ipfs-ios-backup`. 

See the [README](https://github.com/codynhat/libimobiledevice) for more info.

# Usage

```
Backup iOS devices to IPFS

Usage:
  ipfs-ios-backup [command]

Available Commands:
  backups     Interact with iOS backups
  devices     Interact with connected iOS devices
  help        Help about any command
  init        Initialize ipfs-ios-backup repo

Flags:
      --config string     config file (default is $HOME/.ipfs-ios-backup.json)
  -h, --help              help for ipfs-ios-backup
      --repoPath string   Path to IPFS iOS Backup repo (default "/Users/Cody/.ipfs-ios-backup")

Use "ipfs-ios-backup [command] --help" for more information about a command.
```

## Initialize repo

The repo will need to be initialized before doing anything.
``` sh
ipfs-ios-desktop init
```

By default, the repo will be `$HOME/.ipfs-ios-backup`. You can use the `--repoPath` flag to set a custom path
``` sh
ipfs-ios-desktop init --repoPath=$HOME/.ipfs-ios-backup-custom
```

or set `repoPath` in the configuration file at `$HOME/.ipfs-ios-backup.json`.

``` json
{
    "repoPath": "$HOME/.ipfs-ios-backup-custom"
}
```

## Finding devices

Search for devices connected via USB or on the same Wifi network.

```
ipfs-ios-backup devices list
```

## Enable backups for a device

```
ipfs-ios-backup backups enable [device-id]
```

You will be prompted to enter a password to use for encrypting backups for this device. The password is not stored anywhere on your computer. The backup is encrypted on your iOS device before any data is sent to your computer.

## Perform a backup

```
ipfs-ios-backup backups perform [device-id]
```

## Restore a backup

A backup can be restored to a device. **YOUR DEVICE AND DATA WILL BE RESTORED**. You will be prompted to enter the password of the backup before the restore begins.

```
ipfs-ios-backup backups restore [device-id]
```

## Connect to private network

Currently, the easiest way to connect `ipfs-ios-backup` to a private network is by using an IPFS daemon pointed to the embedded IPFS repo.

```
IPFS_PATH=$HOME/.ipfs-ios-backup/.ipfs ipfs daemon
```

The swarm key located in the repo (`$HOME/.ipfs-ios-backup/.ipfs/swarm.key`) must be the same as other peers in the private network. These peers will be able to query IPNS names and IPFS objects from `ipfs-ios-backup backups list`. Note, however, only the node responsible for performing backups will have the private keys for IPNS and a list of performed backups.

A current limitation exists that prevents `ipfs-ios-backup` commands from running while an IPFS daemon is running.

# Architecture

![IPFS iOS Backup Architecture](https://raw.githubusercontent.com/codynhat/ipfs-ios-backup/master/docs/IPFS%20iOS%20Backup%20Architecture.png)

# Roadmap

- Automatic backups performed by a daemon
- Encryption of metadata
- Integration with pinning services

