/*
Copyright © 2020 Cody Hatfield <cody.hatfield@me.com>

Permission is hereby granted, free of charge, to any person obtaining a copy
of this software and associated documentation files (the "Software"), to deal
in the Software without restriction, including without limitation the rights
to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
copies of the Software, and to permit persons to whom the Software is
furnished to do so, subject to the following conditions:

The above copyright notice and this permission notice shall be included in
all copies or substantial portions of the Software.

THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN
THE SOFTWARE.
*/
package cmd

import (
	"fmt"

	"github.com/codynhat/ipfs-ios-backup/idevice"
	"github.com/spf13/cobra"
)

var devicesCmd = &cobra.Command{
	Use:   "devices [command]",
	Short: "Interact with connected iOS devices",
	Long:  "Interact with connected iOS devices",
}

var devicesListCmd = &cobra.Command{
	Use:   "list",
	Short: "List connected iOS devices",
	Long:  "List connected iOS devices",
	Run: func(cmd *cobra.Command, args []string) {
		devices, err := idevice.GetDevices()

		if err != nil {
			panic(err)
		}

		if len(devices) == 0 {
			fmt.Println("No connected devices found.")
		}

		// Print out devices
		for i := 0; i < len(devices); i++ {
			device := devices[i]
			var connTypeStr string
			switch device.ConnectionType {
			case idevice.USB:
				connTypeStr = "USB"
			case idevice.WIFI:
				connTypeStr = "WiFi"
			default:
				connTypeStr = "Unknown"
			}

			deviceName, err := idevice.GetDeviceName(device.Udid)
			if err != nil {
				panic(err)
			}

			fmt.Printf("%s (Name: \"%s\", Connection Type: \"%s\")\n", device.Udid, deviceName, connTypeStr)
		}
	},
}

func init() {
	rootCmd.AddCommand(devicesCmd)
	devicesCmd.AddCommand(devicesListCmd)
}
