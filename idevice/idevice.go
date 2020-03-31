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
package idevice

/*
#cgo LDFLAGS: -limobiledevice
#include <stdlib.h>
#include <libimobiledevice/libimobiledevice.h>
*/
import "C"

import (
	"errors"
	"unsafe"
)

// DeviceID is an identifier for a device
type DeviceID string

// DeviceConnectionType is a type of connection a device is available on
type DeviceConnectionType int

const (
	// USB connection type
	USB DeviceConnectionType = 1
	// WIFI connection type
	WIFI DeviceConnectionType = 2
)

// Device is a representation of an iOS device
type Device struct {
	Udid           DeviceID
	ConnectionType DeviceConnectionType
}

// GetDevices gets the DeviceID of all connected devices
func GetDevices() ([]Device, error) {
	var cDeviceInfos *C.idevice_info_t
	var length C.int

	err := C.idevice_get_device_list_extended(&cDeviceInfos, &length)
	defer C.idevice_device_list_extended_free(cDeviceInfos)
	if err < 0 {
		return nil, errors.New("Could not retrieve list of devices")
	}

	cDevices := (*[1 << 28]C.idevice_info_t)(unsafe.Pointer(cDeviceInfos))[:length:length]
	devices := make([]Device, int(length))

	for i := 0; i < int(length); i++ {
		cDevice := cDevices[i]
		var deviceID DeviceID = DeviceID(C.GoString(cDevice.udid))
		var connectionType DeviceConnectionType = DeviceConnectionType(int(cDevice.conn_type))
		devices[i] = Device{deviceID, connectionType}
	}

	return devices, nil
}
