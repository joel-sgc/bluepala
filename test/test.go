package test

import "bluepala/common"

var Devices = []common.Device{
	{Path: "/org/bluez/hci0/dev_00_1A_7D_DA_71_13", Name: "Sony WH-1000XM4", Address: "00:1A:7D:DA:71:13", Icon: "audio-headset", AddressType: "LE", Paired: true, Trusted: true, Connected: true, Battery: 85, Connectable: true, RSSI: -42},
	{Path: "/org/bluez/hci0/dev_3C_5A_B4_6F_12_89", Name: "Logitech MX Master 3", Address: "3C:5A:B4:6F:12:89", Icon: "input-mouse", AddressType: "LE", Paired: true, Trusted: true, Connected: true, Battery: 100, Connectable: true, RSSI: -38},
	{Path: "/org/bluez/hci0/dev_5D_4F_3B_2C_78_44", Name: "Apple AirPods Pro", Address: "5D:4F:3B:2C:78:44", Icon: "audio-headset", AddressType: "LE", Paired: true, Trusted: true, Connected: false, Battery: 67, Connectable: true, RSSI: -55},
	{Path: "/org/bluez/hci0/dev_7E_3C_6A_4B_91_21", Name: "Fitbit Charge 5", Address: "7E:3C:6A:4B:91:21", Icon: "input-watch", AddressType: "LE", Paired: true, Trusted: false, Connected: true, Battery: 92, Connectable: true, RSSI: -60},
	{Path: "/org/bluez/hci0/dev_12_34_56_78_9A_BC", Name: "Samsung Galaxy Buds 2", Address: "12:34:56:78:9A:BC", Icon: "audio-headset", AddressType: "LE", Paired: false, Trusted: false, Connected: false, Battery: 45, Connectable: true, RSSI: -70},
	{Path: "/org/bluez/hci0/dev_98_76_54_32_10_FE", Name: "Microsoft Surface Keyboard", Address: "98:76:54:32:10:FE", Icon: "input-keyboard", AddressType: "LE", Paired: true, Trusted: true, Connected: true, Battery: 88, Connectable: true, RSSI: -50},
	{Path: "/org/bluez/hci0/dev_AB_CD_EF_12_34_56", Name: "Bose QuietComfort 35", Address: "AB:CD:EF:12:34:56", Icon: "audio-headset", AddressType: "BR/EDR", Paired: true, Trusted: true, Connected: true, Battery: 75, Connectable: true, RSSI: -40},
	{Path: "/org/bluez/hci0/dev_11_22_33_44_55_66", Name: "Garmin Venu 2", Address: "11:22:33:44:55:66", Icon: "input-watch", AddressType: "LE", Paired: true, Trusted: false, Connected: false, Battery: 60, Connectable: true, RSSI: -65},
	{Path: "/org/bluez/hci0/dev_66_55_44_33_22_11", Name: "Razer DeathAdder V2", Address: "66:55:44:33:22:11", Icon: "input-mouse", AddressType: "LE", Paired: false, Trusted: false, Connected: false, Battery: -1, Connectable: true, RSSI: -75},
	{Path: "/org/bluez/hci0/dev_FF_EE_DD_CC_BB_AA", Name: "JBL Charge 5", Address: "FF:EE:DD:CC:BB:AA", Icon: "audio-speaker", AddressType: "LE", Paired: true, Trusted: true, Connected: true, Battery: 50, Connectable: true, RSSI: -48},
	{Path: "/org/bluez/hci0/dev_01_23_45_67_89_AB", Name: "Dell KM717 Keyboard/Mouse", Address: "01:23:45:67:89:AB", Icon: "input-keyboard", AddressType: "LE", Paired: true, Trusted: true, Connected: true, Battery: 95, Connectable: true, RSSI: -35},
	{Path: "/org/bluez/hci0/dev_BA_DC_FE_98_76_54", Name: "Sony WF-1000XM4", Address: "BA:DC:FE:98:76:54", Icon: "audio-headset", AddressType: "LE", Paired: true, Trusted: false, Connected: false, Battery: 80, Connectable: true, RSSI: -52},
	{Path: "/org/bluez/hci0/dev_23_45_67_89_AB_CD", Name: "Logitech G915 Keyboard", Address: "23:45:67:89:AB:CD", Icon: "input-keyboard", AddressType: "LE", Paired: true, Trusted: true, Connected: true, Battery: 70, Connectable: true, RSSI: -43},
	{Path: "/org/bluez/hci0/dev_34_56_78_9A_BC_DE", Name: "Anker Soundcore 2", Address: "34:56:78:9A:BC:DE", Icon: "audio-speaker", AddressType: "LE", Paired: true, Trusted: false, Connected: true, Battery: 40, Connectable: true, RSSI: -65},
	{Path: "/org/bluez/hci0/dev_45_67_89_AB_CD_EF", Name: "Samsung Galaxy Watch 4", Address: "45:67:89:AB:CD:EF", Icon: "input-watch", AddressType: "LE", Paired: true, Trusted: true, Connected: false, Battery: 55, Connectable: true, RSSI: -62},
	{Path: "/org/bluez/hci0/dev_56_78_9A_BC_DE_F0", Name: "Corsair Dark Core Mouse", Address: "56:78:9A:BC:DE:F0", Icon: "input-mouse", AddressType: "LE", Paired: true, Trusted: true, Connected: true, Battery: 90, Connectable: true, RSSI: -47},
	{Path: "/org/bluez/hci0/dev_67_89_AB_CD_EF_01", Name: "Beats Studio Buds", Address: "67:89:AB:CD:EF:01", Icon: "audio-headset", AddressType: "LE", Paired: false, Trusted: false, Connected: false, Battery: 35, Connectable: true, RSSI: -73},
	{Path: "/org/bluez/hci0/dev_78_9A_BC_DE_F0_12", Name: "HP Elite Keyboard", Address: "78:9A:BC:DE:F0:12", Icon: "input-keyboard", AddressType: "LE", Paired: true, Trusted: true, Connected: true, Battery: 85, Connectable: true, RSSI: -41},
	{Path: "/org/bluez/hci0/dev_89_AB_CD_EF_01_23", Name: "Bose SoundLink Mini", Address: "89:AB:CD:EF:01:23", Icon: "audio-speaker", AddressType: "BR/EDR", Paired: true, Trusted: true, Connected: false, Battery: 60, Connectable: true, RSSI: -58},
	{Path: "/org/bluez/hci0/dev_9A_BC_DE_F0_12_34", Name: "Roku Remote", Address: "9A:BC:DE:F0:12:34", Icon: "input-remote", AddressType: "LE", Paired: false, Trusted: false, Connected: false, Battery: -1, Connectable: true, RSSI: -68},
}
