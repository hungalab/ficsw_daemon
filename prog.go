//-----------------------------------------------------------------------------
// prog.go
// nyacom (C) 2018.05
// Xilinx SelectMAP x16 x8 interface
// References
// https://japan.xilinx.com/support/documentation/application_notes/j_xapp583-fpga-configuration.pdf
// https://japan.xilinx.com/support/documentation/user_guides/j_ug570-ultrascale-configuration.pdf
//-----------------------------------------------------------------------------
package main

import (
	"fmt"
//	"flag"
//	"log"
//	"os"
//	"os/signal"
	"time"
	"errors"
//	"unsafe"
//	"reflect"
//	"syscall"
	"./gpio"
)

//-----------------------------------------------------------------------------
func init_pin16() {
	gpio.Set_all_input()
	for _, v := range PIN {
		switch v {
		// Set input
		case PIN["RP_PWOK"], PIN["RP_INIT"], PIN["RP_DONE"], PIN["RP_G_CKSEL"]:
			gpio.Set_input(v)

		// Set output
		case PIN["RP_PROG"], PIN["RP_CSI"], PIN["RP_RDWR"]:
			gpio.Set_output(v)
		//	gpio.Set_bus(1<<v)	// Negate

		// Set output
		case PIN["RP_CCLK"],
			PIN["RP_CD0"], PIN["RP_CD1"], PIN["RP_CD2"], PIN["RP_CD3"],
			PIN["RP_CD4"], PIN["RP_CD5"], PIN["RP_CD6"], PIN["RP_CD7"],
			PIN["RP_CD8"], PIN["RP_CD9"], PIN["RP_CD10"], PIN["RP_CD11"],
			PIN["RP_CD12"], PIN["RP_CD13"], PIN["RP_CD14"], PIN["RP_CD15"]:
			gpio.Set_output(v)
			gpio.Clr_bus(1<<v)	// Negate
		}
	}
}

//-----------------------------------------------------------------------------
func init_pin8() {
	gpio.Set_all_input()
	for _, v := range PIN {
		switch v {
		// Set input
		case PIN["RP_PWOK"], PIN["RP_INIT"], PIN["RP_DONE"], PIN["RP_G_CKSEL"]:
			gpio.Set_input(v)

		// Set output
		case PIN["RP_PROG"], PIN["RP_CSI"], PIN["RP_RDWR"]:
			gpio.Set_output(v)
		//	gpio.Set_bus(1<<v)	// Negate

		// Set output
		case PIN["RP_CCLK"],
			PIN["RP_CD0"], PIN["RP_CD1"], PIN["RP_CD2"], PIN["RP_CD3"],
			PIN["RP_CD4"], PIN["RP_CD5"], PIN["RP_CD6"], PIN["RP_CD7"] :
			gpio.Set_output(v)
			gpio.Clr_bus(1<<v)	// Negate
		}
	}
}

//-----------------------------------------------------------------------------
func Setup() {
	gpio.Setup()
	gpio.Set_all_input()

	// Check power ok
	//fmt.Println("CHECK: PW_OK:", gpio.Get_pin(PIN["RP_PWOK"]))
}

// prog with Selectmap 8 method
func Prog8(bitstream []byte, prMode bool)(error){
	err := gpio.Gpio_lock()
	if err != nil {
		return err
	}
	defer gpio.Gpio_unlock()

	fmt.Println("PROG: Entering Xilinx SelectMap x8 configuration mode...")

	init_pin8()

	// Invoke configuration
	if prMode == false {
		gpio.Set_bus(uint32(PIN_BIT["RP_PROG"]|PIN_BIT["RP_CSI"]|PIN_BIT["RP_RDWR"]))	// Negate
		gpio.Clr_bus(uint32(PIN_BIT["RP_PROG"]|PIN_BIT["RP_CSI"]|PIN_BIT["RP_RDWR"]))	// Assert
		gpio.Set_bus(uint32(PIN_BIT["RP_PROG"]))                                        // Negate

		for gpio.Get_pin(PIN["RP_INIT"]) == 0 {
			time.Sleep(1 * time.Second)
		}

	} else {
		gpio.Set_bus(PIN_BIT["RP_CSI"]|PIN_BIT["RP_RDWR"]) // Negate
		fmt.Println("PROG: Partial Reconfiguration mode selected")
		gpio.Clr_bus(PIN_BIT["RP_CSI"]|PIN_BIT["RP_RDWR"]) // Assert
	}

	fmt.Println("PROG: Ready to program")

	file_size := len(bitstream)
	fmt.Println("PROG: Size : ", file_size, " B")

	gpio.Clr_bus(uint32(PIN["RP_CCLK"]))

	fmt.Println("PROG: Programming...")
	buf := bitstream

	//read_byte := 0

	for i := 0; i < len(buf); i = i + 1 {
		data := (uint32(buf[i]) << 8)
		gpio.Clr_bus((^data & 0x0000ff00) | uint32(PIN_BIT["RP_CCLK"]))
		gpio.Set_bus((data & 0x0000ff00))
		gpio.Set_bus(uint32(PIN_BIT["RP_CCLK"]))

		if gpio.Get_pin(PIN["RP_INIT"]) == 0 {
			return errors.New("Configuraion Error (while prog)")
		}
	}

	//fmt.Printf("PROG: %d / %d (%.2f %%)\n",
	//read_byte, file_size, float32(read_byte) / float32(file_size) * 100)

	//gpio.Clr_bus(uint32(PIN_BIT["RP_CCLK"]))	// Negate CLK

	if prMode == false {
		fmt.Println("PROG: Waiting FPGA done")

		for gpio.Get_pin(PIN["RP_DONE"]) == 0 {		// Wait until RP_DONE asserted
			if gpio.Get_pin(PIN["RP_INIT"]) == 0 {
				return errors.New("Configuration Error (while waiting)")
			}
			gpio.Set_bus(uint32(PIN_BIT["RP_CCLK"]))
			gpio.Clr_bus(uint32(PIN_BIT["RP_CCLK"]))
		}

		gpio.Clr_bus(0x0000ff00 | uint32(PIN_BIT["RP_CCLK"]))
		fmt.Println("PROG: FPGA program done")
	}

	defer gpio.Set_all_input()

	return nil
}

// prog with Selectmap 16 method
func Prog16(bitstream []byte, prMode bool)(error) {
	err := gpio.Gpio_lock()
	if err != nil {
		return err
	}
	defer gpio.Gpio_unlock()

	fmt.Println("PROG: Entering Xilinx SelectMap x16 configuration mode...")

	init_pin16()

	if prMode == false {
		gpio.Set_bus(uint32(PIN_BIT["RP_PROG"]|PIN_BIT["RP_CSI"]|PIN_BIT["RP_RDWR"]))	// Negate
		gpio.Clr_bus(uint32(PIN_BIT["RP_PROG"]|PIN_BIT["RP_CSI"]|PIN_BIT["RP_RDWR"]))	// Assert
		gpio.Set_bus(uint32(PIN_BIT["RP_PROG"]))                                        // Negate

		for gpio.Get_pin(PIN["RP_INIT"]) == 0 {
			time.Sleep(1 * time.Second)
		}

	} else {
		gpio.Set_bus(PIN_BIT["RP_CSI"]|PIN_BIT["RP_RDWR"]) // Negate
		fmt.Println("PROG: Partial Reconfiguration mode selected")
		gpio.Clr_bus(PIN_BIT["RP_CSI"]|PIN_BIT["RP_RDWR"]) // Assert
	}

	fmt.Println("PROG: Ready to program")

	file_size := len(bitstream)
	fmt.Println("PROG: Size : ", file_size, " B")

	gpio.Clr_bus(uint32(PIN["RP_CCLK"]))

	fmt.Println("PROG: Programming...")
	buf := bitstream

	for i := 0; i < len(buf); i = i + 2 {
		data := (uint32(buf[i+1]) << 8 | uint32(buf[i])) << 8
		gpio.Clr_bus((^data & 0x00ffff00) | uint32(PIN_BIT["RP_CCLK"]))
		gpio.Set_bus((data & 0x00ffff00))
		gpio.Set_bus(uint32(PIN_BIT["RP_CCLK"]))

		if gpio.Get_pin(PIN["RP_INIT"]) == 0 {
			return errors.New("Configuraion Error (while prog)")
		}
	}

//		fmt.Printf("PROG: %d / %d (%.2f %%)\n",
//			read_byte, file_size, float32(read_byte) / float32(file_size) * 100)

	//gpio.Clr_bus(uint32(PIN_BIT["RP_CCLK"]))	// Negate CLK

	if prMode == false {
		fmt.Println("PROG: Waiting FPGA done")

		for gpio.Get_pin(PIN["RP_DONE"]) == 0 {		// Wait until RP_DONE asserted
			if gpio.Get_pin(PIN["RP_INIT"]) == 0 {
				return errors.New("Configuration Error (while waiting)")
			}
			gpio.Set_bus(uint32(PIN_BIT["RP_CCLK"]))
			gpio.Clr_bus(uint32(PIN_BIT["RP_CCLK"]))
		}

		gpio.Clr_bus(0x00ffff00 | uint32(PIN_BIT["RP_CCLK"]))
		fmt.Println("PROG: FPGA program done")
	}

	defer gpio.Set_all_input()

	return nil
}

////-----------------------------------------------------------------------------
//func init() {
//	fmt.Println("")
//	fmt.Println("FiC FPGA Configurator (golang ver)")
//	fmt.Println("nyacom (C) 2018.05 <kzh@nyacom.net>")
//}
//
//func help_str() {
//	fmt.Println("Usage: ficprog.go INPUT_FILE.bin [-m {8, 16}] [-c]")
//	fmt.Println(" -m {8, 16} ... Selectmap data width (default 16)")
//	fmt.Println(" -c ... No-reset mode for PR (default reset mode)")
//}
//
//func main() {
//	// Check arguments
//	if len(os.Args) < 2 {
//		help_str()
//		log.Fatal("Insufficient argument")
//	}
//
//	if os.Args[1] == "help" {
//	}
//
//	fs := flag.NewFlagSet(os.Args[0], flag.ExitOnError)
//	var (
//		confMode = fs.Int("m", 16, "Selectmap mode (default 16)")
//		prMode = fs.Bool("c", false, "No-reset mode (default Reset mode)")
//	)
//
//	noopt := os.Args[1]
//	fs.Parse(os.Args[1:])
//
//	left_args := fs.Args()
//	if len(left_args) > 0 {
//		noopt = left_args[0]
//	}
//	fs.Parse(left_args[1:])
//
//	// Check options
//	if *confMode != 8 && *confMode != 16 {
//		log.Fatal("Error: Invalid configurtaion mode")
//	}
//
//	// Help
//	if noopt == "help" {
//		help_str()
//		return
//	}
//
//	infile := noopt
//
//	// Create GPIO lockfile
//	if lock := gpio.Gpio_lock(); !lock {
//		log.Fatal("Error: Can't lock for GPIO")
//	}
//	defer gpio.Gpio_unlock();
//
//	// Signal handring
//	sig_ch := make(chan os.Signal, 1)
//	signal.Notify(sig_ch,
//		syscall.SIGHUP,
//		syscall.SIGINT,
//		syscall.SIGTERM,
//		syscall.SIGQUIT)
//
//	go func() {
//		for sig := range sig_ch {
//			close(sig_ch)
//			fmt.Println("INTR:", sig)
//			gpio.Gpio_unlock()
//			os.Exit(1)
//		}
//	} ()
//
//	// GPIO setup
//	Setup()
//
//	// Select configuration mode
//	switch *confMode {
//	case 8:
//		err := Prog8(infile, *prMode)
//		if err != nil {
//			fmt.Fprint(os.Stderr, err, "\n")
//		}
//
//	case 16:
//		err := Prog16(infile, *prMode)
//		if err != nil {
//			fmt.Fprint(os.Stderr, err, "\n")
//		}
//	}
//}

