import {Command, flags} from '@oclif/command'
import { exec } from 'child_process'
import * as util from 'util'

export default class Devices extends Command {
  static description = 'List connected iOS devices'

  static examples = [
    `$ ipfs-ios-backup devices
    <device-name-A>
    <device-name-B>
    `,
  ]

  static flags = {}

  static args = []

  async run() {
    const execP = util.promisify(exec)
    const { stdout, stderr } = await execP('idevice_id -l')
    if(stderr) {
      console.error("Something went wrong.")
    }
    console.log("Connected device UUIDs:")
    console.log(stdout)
  }
}
