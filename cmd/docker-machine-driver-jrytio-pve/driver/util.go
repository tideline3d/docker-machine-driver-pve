package driver

import (
	"slices"
	"strings"
)

func getMACFromPveNetworkDevice(device string) string {
	models := []string{
		"e1000",
		"e1000-82540em",
		"e1000-82544gc",
		"e1000-82545em",
		"e1000e",
		"i82551",
		"i82557b",
		"i82559er",
		"ne2k_isa",
		"ne2k_pci",
		"pcnet",
		"rtl8139",
		"virtio",
		"vmxnet3",
	}

	for _, param := range strings.Split(device, ",") {
		//nolint:mnd
		values := strings.SplitN(param, "=", 2)

		//nolint:mnd
		if len(values) != 2 {
			continue
		}

		if slices.Contains(models, values[0]) {
			return values[1]
		}
	}

	return ""
}
