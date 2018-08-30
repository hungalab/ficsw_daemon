//-----------------------------------------------------------------------------
// comm.go ()
// nyacom (C) 2018.05
//-----------------------------------------------------------------------------
package main

import (
	"time"
	"errors"
	"./gpio"	// RPi GPIO lib
	"fmt"
)

//-----------------------------------------------------------------------------
// GPIO pin setup for communication
//-----------------------------------------------------------------------------
func gpio_comm_setup() {
	gpio.Set_all_input()
	for _, v := range PIN {
		switch v {
			case PIN_COMM["FACK"], PIN_COMM["FREQ"] : {
				gpio.Set_input(v)
			}
			case PIN_COMM["RREQ"], PIN_COMM["RSTB"],
				PIN_COMM["DATA7"], PIN_COMM["DATA6"],
				PIN_COMM["DATA5"], PIN_COMM["DATA4"],
				PIN_COMM["DATA3"], PIN_COMM["DATA2"],
				PIN_COMM["DATA1"], PIN_COMM["DATA0"] : {
				gpio.Set_output(v)
				gpio.Clr_bus(1<<v)
			}
		}
	}
}

//-----------------------------------------------------------------------------
// set GPIO data pin direction
//-----------------------------------------------------------------------------
func comm_dir(dir int) {
	data_pins := []uint32{ PIN_COMM["DATA7"], PIN_COMM["DATA6"],
				PIN_COMM["DATA5"], PIN_COMM["DATA4"],
				PIN_COMM["DATA3"], PIN_COMM["DATA2"],
				PIN_COMM["DATA1"], PIN_COMM["DATA0"] }

	if dir == COM_DIR_SND {
		for _, p := range data_pins {
			gpio.Set_output(p)
		}

	} else if dir == COM_DIR_RCV {
		for _, p := range data_pins {
			gpio.Set_input(p)
		}
	}
}

//-----------------------------------------------------------------------------
// Wait until fack is down
//-----------------------------------------------------------------------------
func comm_wait_fack_down() error {
	// Wait for ACK from FiC
	t1 := time.Now()
	for gpio.Get_pin(PIN_COMM["FACK"]) == 1 {
		time.Sleep(1 * time.Millisecond)
		t2 := time.Now()
		if (t2.Sub(t1).Seconds() > COM_TIMEOUT) {
			return errors.New("Communication time out (fack_down)")
		}
	}
	return nil
}

//-----------------------------------------------------------------------------
// Wait until fack is up
//-----------------------------------------------------------------------------
func comm_wait_fack_up() error {
	// Wait for ACK from FiC
	t1 := time.Now()
	for gpio.Get_pin(PIN_COMM["FACK"]) == 0 {
		time.Sleep(1 * time.Millisecond)
		t2 := time.Now()
		if (t2.Sub(t1).Seconds() > COM_TIMEOUT) {
			return errors.New("Communication time out (fack_up)")
		}
	}
	return nil
}

//-----------------------------------------------------------------------------
// Send data bus
//-----------------------------------------------------------------------------
func comm_send(bus uint32) error {
	gpio.Clr_bus(^(bus & COM_MASK))
	gpio.Set_bus(bus & COM_MASK)

	err := comm_wait_fack_up()	// Wait FiC ack up
	if err != nil {
		return err
	}

	gpio.Clr_bus((1<<PIN_COMM["RSTB"]))	// Negate RPi stb 

	err = comm_wait_fack_down()	// Wait FiC ack down
	if err != nil {
		return err
	}

	return nil
}

//-----------------------------------------------------------------------------
// Receive data bus
//-----------------------------------------------------------------------------
func comm_receive(bus uint32)(b uint8, err error) {
	// assert rstb
	bus = (1<<PIN_COMM["RREQ"])|(1<<PIN_COMM["RSTB"])
	gpio.Clr_bus(^(bus & COM_MASK))
	gpio.Set_bus(bus & COM_MASK)
	//fmt.Printf("DEBUG: send rstb %x\n", bus)

	err = comm_wait_fack_up()	// Wait FiC ack up
	if err != nil {
		fmt.Println("DEBUG: comm_receive timeout")
		return 0, err
	}

	b = uint8((gpio.Get_bus() >> PIN_COMM["DATA0"]) & 0xff)	// Receive data

	// negate rstb
	bus = (1<<PIN_COMM["RREQ"])
	gpio.Clr_bus(^(bus & COM_MASK))
	//gpio.Set_bus(bus & COM_MASK)
	//fmt.Printf("DEBUG: send ~rstb %x\n", bus)

	err = comm_wait_fack_down()	// Wait FiC ack down
	if err != nil {
		return 0, err
	}

	return b, nil
}

//-----------------------------------------------------------------------------
// Set address
//-----------------------------------------------------------------------------
func comm_set_addr(addr uint16)(err error) {
	// Note: Send write address 4times in 4bit mode
	// Send address high-high (4bit)
	bus := (1<<PIN_COMM["RREQ"])|(1<<PIN_COMM["RSTB"])|((uint32(addr)>>12)<<PIN_COMM["DATA4"])
	fmt.Printf("DEBUG: send addr high-high %x\n", bus)
	err = comm_send(bus)
	if err != nil {
		//fmt.Println("DEBUG: send address high-high failed")
		return err
	}

	// Send address high-low (4bit)
	bus = (1<<PIN_COMM["RREQ"])|(1<<PIN_COMM["RSTB"])|(((uint32(addr)>>8)&0x0f)<<PIN_COMM["DATA4"])
	fmt.Printf("DEBUG: send addr high-low %x\n", bus)
	err = comm_send(bus)
	if err != nil {
		//fmt.Println("DEBUG: send address high-low failed")
		return err
	}

	// Send address low-high (4bit)
	bus = (1<<PIN_COMM["RREQ"])|(1<<PIN_COMM["RSTB"])|(((uint32(addr)>>4)&0x0f)<<PIN_COMM["DATA4"])
	fmt.Printf("DEBUG: send addr low-high %x\n", bus)
	err = comm_send(bus)
	if err != nil {
		//fmt.Println("DEBUG: send address low-high failed")
		return err
	}

	// Send address low-low (4bit)
	bus = (1<<PIN_COMM["RREQ"])|(1<<PIN_COMM["RSTB"])|((uint32(addr)&0x0f)<<PIN_COMM["DATA4"])
	fmt.Printf("DEBUG: send addr low-low %x\n", bus)
	err = comm_send(bus)
	if err != nil {
		//fmt.Println("DEBUG: send address low-low failed")
		return err
	}

	return nil
}

// Write 1Byte 
//-----------------------------------------------------------------------------
func fic_write8(addr uint16, data uint8) error {
	gpio_comm_setup()
	comm_dir(COM_DIR_SND)

	// Send Handshake and CMD
	bus := uint32((1<<PIN_COMM["RREQ"])|(1<<PIN_COMM["RSTB"])|(COM_CMD_WRITE<<PIN_COMM["DATA4"]))
	err := comm_send(bus)
	if err != nil {
		return err
	}

	// Set address
	comm_set_addr(addr)

	// Note: Send data 2 times 

	// Send data high (4bit)
	bus = (1<<PIN_COMM["RREQ"])|(1<<PIN_COMM["RSTB"])|(uint32((data&0xf0)>>4)<<PIN_COMM["DATA4"])
	fmt.Printf("DEBUG: send data high %x\n", bus)
	err = comm_send(bus)
	if err != nil {
		return err
	}

	// Send data low (4bit)
	bus = (1<<PIN_COMM["RREQ"])|(1<<PIN_COMM["RSTB"])|(uint32(data&0x0f)<<PIN_COMM["DATA4"])
	fmt.Printf("DEBUG: send data low %x\n", bus)
	err = comm_send(bus)
	if err != nil {
		return err
	}

	bus = (1<<PIN_COMM["RREQ"])|(1<<PIN_COMM["RSTB"])
	gpio.Clr_bus(^bus) // Negate REQ and STB

	defer gpio.Set_all_input()

	return nil
}

//-----------------------------------------------------------------------------
// Read 1Byte 
//-----------------------------------------------------------------------------
func fic_read8(addr uint16)(b uint8, err error){
	gpio_comm_setup()
	comm_dir(COM_DIR_SND)

	// Send Handshake and CMD
	bus := uint32((1<<PIN_COMM["RREQ"])|(1<<PIN_COMM["RSTB"])|(COM_CMD_READ<<PIN_COMM["DATA4"]))
	//fmt.Printf("DEBUG: send cmd %x\n", bus)
	err = comm_send(bus)
	if err != nil {
		//fmt.Println("DEBUG: send cmd failed")
		return 0, err
	}

	// Set address
	comm_set_addr(addr)
	
	// Switch bus direction
	comm_dir(COM_DIR_RCV)

	// Note: Read data 2 times..

	// Read high 4bit
	rcv, err := comm_receive(bus)
	fmt.Printf("DEBUG: read bus high %x\n", rcv)
	if err != nil {
		return 0, err
	}
	b = rcv & 0xf0

	// Read low 4bit
	rcv, err = comm_receive(bus)
	fmt.Printf("DEBUG: read bus low %x\n", rcv)
	if err != nil {
		return 0, err
	}
	b |= (rcv & 0xf0) >> 4

	defer gpio.Set_all_input()

	return b, nil
}
