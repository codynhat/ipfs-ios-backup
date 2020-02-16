import {Client} from '@textile/threads-client'
import { NodeHttpTransport } from "@improbable-eng/grpc-web-node-http-transport"

const threadsSchema = {
    $id: "https://example.com/person.schema.json",
    $schema: "http://json-schema.org/draft-07/schema#",
    title: "iOS Backup",
    type: "object",
    properties: {
      ID: {
        type: "string",
        description: "The item id."
      },
      deviceUUID: {
        type: "string",
        description: "The UUID of the device being backed up"
      },
      backupCID: {
        type: "string",
        description: "The CID of the backup"
      }
    },
    required: ["deviceUUID", "backupCID"]
  }

const client = new Client({
    transport: NodeHttpTransport()
})

client.newStore().then(store => {
  console.log(store)
  return client.registerSchema(store.id, 'iOS Backup', threadsSchema)
})