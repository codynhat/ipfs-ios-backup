import {Command, flags} from '@oclif/command'
import { exec, spawn, spawnSync } from 'child_process'
import * as util from 'util'
const ipfsClient = require('ipfs-http-client')
const tar = require('tar')
const pinataSDK = require('@pinata/sdk')
import * as fs from 'fs'

export default class Backup extends Command {
  static description = 'Backup an iOS device'

  static examples = [
    `$ ipfs-ios-backup backup ipfs b4c1d70d521a5c110b16f9881f1d0 --password=backup-password --ipfsAddr=/ip4/127.0.0.1/tcp/5001`,
  ]

  static flags = {
    password: flags.string({
        required: true,
        description: "Password used to encrypt backup"
    }),
    ipfsAddr: flags.string({
      description: "IPFS host addr"
    }),
    pinataApiKey: flags.string({
      description: "Pinata API Key"
    }),
    pinataSecretApiKey: flags.string({
      description: "Pinata Secret API Key"
    }),
    uploadOnly: flags.boolean({
      description: "Only upload existing backup",
      default: false
    })
  }

  static args = [
    {
      name: "provider",
      required: true,
      description: "Provider used to store backup",
      options: ["ipfs", "pinata"]
    },
    {
      name: "device_uuid",
      required: true,
      description: "UUID of the device to backup"
    }
  ]

  async run() {
    const {args, flags} = this.parse(Backup)

    switch (args.provider) {
      case "ipfs":
        if(flags.ipfsAddr == null) {
          this.error("Missing required flag (ipfsAddr)")
          return
        }
      case "pinata":
        if(flags.pinataApiKey == null) {
          this.error("Missing required flag (pinataApiKey)")
          return
        }
        if(flags.pinataSecretApiKey == null) {
          this.error("Missing required flag (pinataSecretApiKey)")
          return
        }
    }

    const execP = util.promisify(exec)

    this.log("Enabling backup encryption. Enter passcode on device if prompted.")
    try {
        const { stdout, stderr } = await execP(`idevicebackup2 -u ${args.device_uuid} encryption on ${args.password}`)
    } catch (error) {
        this.log("Backup encryption already enabled.")
    }

    if(flags.uploadOnly == false) {
      this.log(`Starting backup to ./${args.device_uuid}`)

      spawnSync("idevicebackup2", ["-u", args.device_uuid, "backup", "./"], { stdio: 'inherit' })
    }

    await this.trimBackupForDemo(`./${args.device_uuid}`)

    this.log("Compressing backup...")
    
    await tar.create({
      gzip: true,
      file: `./${args.device_uuid}.tgz`
    }, [`./${args.device_uuid}`])

    this.log("Finished compressing backup.")

    switch (args.provider) {
      case "ipfs": {
        this.log("Pinning backup to IPFS...")
        let cid = await this.pinToIPFS(flags.ipfsAddr, `./${args.device_uuid}.tgz`)
        this.log(`Successfully pinned backup to IPFS (${cid})`)
      }
      case "pinata": {
        this.log("Pinning backup to IPFS...")
        let cid = await this.pinToIPFS(flags.ipfsAddr, `./${args.device_uuid}.tgz`)
        this.log(`Successfully pinned backup to IPFS (${cid})`)
        this.log("Sending CID to Pinata pin queue...")
        await this.sendHashToPinata(flags.pinataApiKey, flags.pinataSecretApiKey, cid)
        this.log("Successfully sent CID to Pinata queue.")
      }
    }
 }

 async pinToIPFS(ipfsAddr: string | undefined, localPath: string): Promise<string | undefined> {
  const ipfs = ipfsClient(ipfsAddr)
  const cmd = ipfs.add(fs.createReadStream(localPath), {pin: true})
  for await (const result of cmd) {
    return result.cid.toString()
  }
 }

 async sendHashToPinata(pinataApiKey: string | undefined, pinataSecretApiKey: string | undefined, cid: string | undefined) {
  const pinata = pinataSDK(pinataApiKey, pinataSecretApiKey)
  await pinata.addHashToPinQueue(cid, {
    pinataMetadata: {
      name: "My iOS Backup"
    }
  })
 }

 async trimBackupForDemo(localPath: string) {
  this.log("Trimming backup for demo...")
   let rmP = util.promisify(fs.unlink)
   let readdirP = util.promisify(fs.readdir)
   let hexes = ['0', '1', '2', '3', '4', '5', '6', '7', '8', '9', 'a', 'b', 'c', 'd', 'e', 'f']
   hexes.forEach(str1 => {
     hexes.forEach(str2 => {
      try {
        let files = fs.readdirSync(`${localPath}/${str1}${str2}/`)
        files.forEach(filePath => {
          fs.unlinkSync(`${localPath}/${str1}${str2}/${filePath}`)
        })
      } catch {}
     })
   })
 }
}
