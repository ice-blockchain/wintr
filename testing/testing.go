// SPDX-License-Identifier: BUSL-1.1

package testing

func GIVEN(_ string, logic func()) {
	logic()
}

func WHEN(_ string, logic func()) {
	logic()
}

func THEN(logic func()) {
	logic()
}

func IT(_ string, logic func()) {
	logic()
}

func AND(_ string, logic func()) {
	logic()
}

func SETUP(_ string, logic func()) {
	logic()
}
