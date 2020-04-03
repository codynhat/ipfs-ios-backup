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
#cgo LDFLAGS: -limobiledevice -lplist
#include <stdlib.h>
#include <libimobiledevice/libimobiledevice.h>
#include <libimobiledevice/lockdown.h>
#include <libimobiledevice/devicebackup2.h>
#include <plist/plist.h>
*/
import "C"

import (
	"errors"
	"fmt"
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
		return nil, errors.New("Failed to retrieve list of devices")
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

// GetDeviceName finds the name of the device with the given ID
func GetDeviceName(deviceID DeviceID) (string, error) {
	var device C.idevice_t
	var client C.lockdownd_client_t

	var cDeviceID *C.char = C.CString(string(deviceID))
	defer C.free(unsafe.Pointer(cDeviceID))

	err := C.idevice_new_with_options(&device, cDeviceID, C.IDEVICE_LOOKUP_USBMUX|C.IDEVICE_LOOKUP_NETWORK)
	defer C.idevice_free(device)
	if err < 0 {
		return "", errors.New("Failed to retrieve device name (idevice_new_with_options)")
	}

	if device == nil {
		return "", fmt.Errorf("No device with UDID (%s) is connected", deviceID)
	}

	var cLabel *C.char = C.CString("ipfs-ios-backup")
	defer C.free(unsafe.Pointer(cLabel))
	err1 := C.lockdownd_client_new(device, &client, cLabel)
	defer C.lockdownd_client_free(client)
	if err1 != C.LOCKDOWN_E_SUCCESS {
		return "", fmt.Errorf("Failed to connect to device (%s)", deviceID)
	}

	var cDeviceName *C.char
	defer C.free(unsafe.Pointer(cDeviceName))
	err1 = C.lockdownd_get_device_name(client, &cDeviceName)
	if err1 != C.LOCKDOWN_E_SUCCESS {
		return "", fmt.Errorf("Failed to get device name (%s)", deviceID)
	}

	return C.GoString(cDeviceName), nil
}

// GetDeviceWillEncrypt queries a device to see if encryption is enabled. See ideviceinfo cmd
func GetDeviceWillEncrypt(deviceID DeviceID) (bool, error) {
	var device C.idevice_t
	var client C.lockdownd_client_t
	var node C.plist_t

	var cDeviceID *C.char = C.CString(string(deviceID))
	defer C.free(unsafe.Pointer(cDeviceID))

	err := C.idevice_new_with_options(&device, cDeviceID, C.IDEVICE_LOOKUP_USBMUX|C.IDEVICE_LOOKUP_NETWORK)
	defer C.idevice_free(device)
	if err < 0 {
		return false, errors.New("Failed to retrieve device name (idevice_new_with_options)")
	}

	if device == nil {
		return false, fmt.Errorf("No device with UDID (%s) is connected", deviceID)
	}

	var cLabel *C.char = C.CString("ipfs-ios-backup")
	defer C.free(unsafe.Pointer(cLabel))
	err1 := C.lockdownd_client_new_with_handshake(device, &client, cLabel)
	defer C.lockdownd_client_free(client)
	if err1 != C.LOCKDOWN_E_SUCCESS {
		return false, fmt.Errorf("Failed to connect to device (%s)", deviceID)
	}

	var cDomain *C.char = C.CString("com.apple.mobile.backup")
	defer C.free(unsafe.Pointer(cDomain))

	var cValue *C.char = C.CString("WillEncrypt")
	defer C.free(unsafe.Pointer(cValue))

	err1 = C.lockdownd_get_value(client, cDomain, cValue, &node)
	if err1 != C.LOCKDOWN_E_SUCCESS {
		return false, fmt.Errorf("Failed to get lockdownd value (%s)", deviceID)
	}

	var b C.uint8_t
	C.plist_get_bool_val(node, &b)

	if uint8(b) > 0 {
		return true, nil
	} else {
		return false, nil
	}

}

// PairDevice will attempt to pair a device with this computer, or do nothing if already paired
func PairDevice(deviceID DeviceID) error {
	var device C.idevice_t
	var client C.lockdownd_client_t

	var cDeviceID *C.char = C.CString(string(deviceID))
	defer C.free(unsafe.Pointer(cDeviceID))

	err := C.idevice_new_with_options(&device, cDeviceID, C.IDEVICE_LOOKUP_USBMUX|C.IDEVICE_LOOKUP_NETWORK)
	defer C.idevice_free(device)
	if err < 0 {
		return errors.New("Failed to retrieve device name (idevice_new_with_options)")
	}

	if device == nil {
		return fmt.Errorf("No device with UDID (%s) is connected", deviceID)
	}

	var cLabel *C.char = C.CString("ipfs-ios-backup")
	defer C.free(unsafe.Pointer(cLabel))

	err1 := C.lockdownd_client_new_with_handshake(device, &client, cLabel)
	defer C.lockdownd_client_free(client)
	if err1 == C.LOCKDOWN_E_SUCCESS {
		// Device is already paired
		return nil
	}

	err1 = C.lockdownd_client_new(device, &client, cLabel)
	defer C.lockdownd_client_free(client)
	if err1 != C.LOCKDOWN_E_SUCCESS {
		return fmt.Errorf("Failed to connect to device (%s)", deviceID)
	}

	err1 = C.lockdownd_pair(client, nil)
	switch err1 {
	case C.LOCKDOWN_E_SUCCESS:
		break
	case C.LOCKDOWN_E_PASSWORD_PROTECTED:
		return fmt.Errorf("Passcode detected. Try unlocking your device and trying again")
		break
	default:
		return fmt.Errorf("Failed to get lockdownd value (%s)", deviceID)
	}

	return nil
}

// PerformBackup performs a backup using devicebackup2
func PerformBackup(deviceID DeviceID, backupDirectory string) error {
	cUdid := C.CString(string(deviceID))
	defer C.free(unsafe.Pointer(cUdid))

	cBackupDir := C.CString(backupDirectory)
	defer C.free(unsafe.Pointer(cBackupDir))

	cErr := C.run_cmd(C.CMD_BACKUP, 0, cUdid, cUdid, cBackupDir, 1, nil, nil)

	if cErr < 0 {
		return fmt.Errorf("devicebackup2 failed with error code %d", cErr)
	}

	return nil
}

// RestoreBackup restores a backup using devicebackup2 with default settings
func RestoreBackup(deviceID DeviceID, backupDirectory string) error {
	cUdid := C.CString(string(deviceID))
	defer C.free(unsafe.Pointer(cUdid))

	cBackupDir := C.CString(backupDirectory)
	defer C.free(unsafe.Pointer(cBackupDir))

	cErr := C.run_cmd(C.CMD_RESTORE, 0, cUdid, cUdid, cBackupDir, 1, nil, nil)

	if cErr < 0 {
		return fmt.Errorf("devicebackup2 failed with error code %d", cErr)
	}

	return nil
}

// EnableBackupEncryption enables backup encryption interactively using devicebackup2
func EnableBackupEncryption(deviceID DeviceID) error {
	cUdid := C.CString(string(deviceID))
	defer C.free(unsafe.Pointer(cUdid))

	cErr := C.run_cmd(C.CMD_CHANGEPW, C.CMD_FLAG_ENCRYPTION_ENABLE, cUdid, cUdid, nil, 1, nil, nil)

	if cErr < 0 {
		return fmt.Errorf("devicebackup2 failed with error code %d", cErr)
	}

	return nil
}
