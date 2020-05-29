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
  backups        Interact with iOS backups
  daemon         Run the ipfs-ios-backup daemon
  devices        Interact with connected iOS devices
  export-secrets Export secrets needed to sync backups with another device
  help           Help about any command
  init           Initialize ipfs-ios-backup repo

Flags:
      --apiAddr string       gRPC API endpoint (default "/ip4/127.0.0.1/tcp/3006")
      --config string        config file (default is $HOME/.ipfs-ios-backup.json)
      --debug                Enable debug logging
  -h, --help                 help for ipfs-ios-backup
      --ipfsAddr string      IPFS address (default "/ip4/0.0.0.0/tcp/4010")
      --repoPath string      Path to IPFS iOS Backup repo (default "$HOME/.ipfs-ios-backup.json")
      --threadsAddr string   Threads IPFS lite node address (default "/ip4/0.0.0.0/tcp/3010")

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

## Schedule Automatic Backups

Backups can be automatically performed in the background by the daemon. This can be enabled in the configuration (default is $HOME/.ipfs-ios-backup.json).

``` json
{
  "schedules": {
    "{DEVICE_NAME}": {
      "deviceID": "{DEVICE_ID}",
      "periodInHours": 6,
      "minBatteryLevel": 50
    }
  }
}
```

Where `{DEVICE_NAME}` and `{DEVICE_ID}` are the name and ID of the device you want to backup, respectively. See [finding devices](#finding-devices) for how to find the device ID.

Each schedule has the following parameters:

| Option | Description |
| ------ | ----------- |
| periodInHours | How many hours should pass between backups |
| minBatteryLevel | The minimum battery level required to perform a backup when a device is not charging |

Notes:
- If a device is connected to a charger, `minBatteryLevel` is ignored
- It is common for a device to not always be detected on WiFi. Therefore, the `periodInHours` is a best-effort and is not guaranteed

## Run the daemon
Interacting with and performing scheduled backups requires the daemon to be running
``` sh
ipfs-ios-desktop daemon
```

## brew service (macOS launchd)
If installed via [Homebrew](#homebrew), the daemon can be started automatically at launch.
``` sh
brew services start ipfs-ios-backup
```

Logs can be found at `${HOMEBREW_PREFIX}/var/log/ipfs-ios-backup.log`

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

## Sync backups with multiple devices

Backups can be stored on multiple devices that are part of the same private IPFS network. This may be multiple computers on your home network, or a private cloud-hosted instance.

To start, a single node must be [initialized](#initialize-repo). Then the secrets from this node will need to be exported.

```
ipfs-ios-backup export-secrets > secrets.json
```

The output should be saved and sent via some secure mechanism to other machines that want to join the network. These secrets can be passed when initializing these other machines.

```
ipfs-ios-backup init --secrets secrets.json
```

*WARNING*: Only send these secrets to trusted nodes. They will join a private IPFS swarm that has access to your backups. While the backups' contents are still encrypted using a password, the metadata is not. Any node that is part of this network will have access to the metadata.

# Architecture

![IPFS iOS Backup Architecture](https://raw.githubusercontent.com/codynhat/ipfs-ios-backup/master/docs/IPFS%20iOS%20Backup%20Architecture.png)

# Roadmap

- Encryption of metadata
- Integration with pinning services

