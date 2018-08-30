//-----------------------------------------------------------------------------
// nyacom FiC monitor daemon (C) 2018.05
// <kzh@nyacom.net>
//-----------------------------------------------------------------------------
package main

import (
	"fmt"
//	"errors"
//	"flag"
	"log"
//	"os"
	"time"
	"net"		// socket
	"strings"
	"strconv"
	"encoding/json"
	"./gpio"	// RPi GPIO lib
//	"ficprog"
//	"unsafe"
//	"reflect"
//	"syscall"
)

//-----------------------------------------------------------------------------
// GPIO pin setup for FPGA programming
//-----------------------------------------------------------------------------
func gpio_prog_setup() {
	gpio.Set_all_input()
	for _, v := range PIN {
		switch v {
			case PIN["RP_PWOK"], PIN["RP_INIT"], PIN["RP_DONE"], PIN["RP_G_CKSEL"]: {
				gpio.Set_input(v)
			}
			default: {
				gpio.Set_output(v)
				gpio.Clr_bus(1<<v)
			}
		}
	}

	// Check power ok
	fmt.Println("CHECK: PW_OK:", gpio.Get_pin(PIN["RP_PWOK"]))
}

//-----------------------------------------------------------------------------
// User module control
//-----------------------------------------------------------------------------
// func fic_module_reset() {
// }
// func fic_module_start() {
// }

//-----------------------------------------------------------------------------
// FPGA reset
//-----------------------------------------------------------------------------
func fic_fpga_init() {
	gpio_prog_setup() // GPIO setup for FPGA prog
	defer gpio.Set_all_input()
	gpio.Set_bus(PIN_BIT["RP_PROG"])
	gpio.Clr_bus(PIN_BIT["RP_PROG"])
}

////-----------------------------------------------------------------------------
//// FPGA programmer
////-----------------------------------------------------------------------------
//func fic_prog(bitstream []byte)(err error) {
//	if lock := gpio.Gpio_lock(); lock == false {
//		return errors.New("GPIO cant lock")
//	}
//	defer gpio.Gpio_unlock()
//
//	//gpio_prog_setup() // GPIO setup for FPGA prog
//	//defer gpio.Set_all_input()
//
//
//	// Invoke configuration
//	gpio.Set_bus((PIN_BIT["RP_PROG"]))
//	gpio.Clr_bus((PIN_BIT["RP_PROG"]|PIN_BIT["RP_CSI"]|PIN_BIT["RP_RDWR"]))
//	gpio.Set_bus((PIN_BIT["RP_PROG"]))
//
//	t1 := time.Now()
//	for gpio.Get_pin(PIN["RP_INIT"]) == 0 {
//		time.Sleep(1 * time.Second)
//		if (time.Now().Sub(t1).Seconds() > COM_TIMEOUT) {
//			return errors.New("Communication time out")
//		}
//	}
//
//	fmt.Println("PROG: Ready to program")
//
//	file_size := len(bitstream)
//	fmt.Println("PROG: File size : ", file_size, " B")
//
//	gpio.Clr_bus((PIN["RP_CCLK"]))
//
//	fmt.Println("PROG: Programming...")
//
//	buf := bitstream
//
//	for i := 0; i < len(buf); i += 2 {
//		data := (uint32(buf[i+1]) << 8 | uint32(buf[i])) << 8
//		gpio.Clr_bus((^data & 0x00ffff00) | PIN_BIT["RP_CCLK"])
//		gpio.Set_bus((data & 0x00ffff00))
//		gpio.Set_bus(PIN_BIT["RP_CCLK"])
//
//		if gpio.Get_pin(PIN["RP_INIT"]) == 0 {
//			return errors.New("Configuraion Error (while prog)")
//		}
//	}
//
//	//gpio.Clr_bus(PIN_BIT["RP_CCLK"])	// Negate CLK
//
//	fmt.Println("PROG: Waiting FPGA done")
//
//	for gpio.Get_pin(PIN["RP_DONE"]) == 0 {		// Wait until RP_DONE asserted
//		if gpio.Get_pin(PIN["RP_INIT"]) == 0 {
//			return errors.New("Configuration Error (while waiting)")
//		}
//		gpio.Set_bus(PIN_BIT["RP_CCLK"])
//		gpio.Clr_bus(PIN_BIT["RP_CCLK"])
//	}
//
//	gpio.Clr_bus(0x00ffff00 | PIN_BIT["RP_CCLK"])
//	fmt.Println("PROG: FPGA program done")
//
//	return nil
//}

//-----------------------------------------------------------------------------
type FicStat struct {
	Ts     time.Time	`json:"ts"`
	State  uint8		`json:"state"`		// Status register
	Hls    uint8		`json:"hls"`		// HLS share register
	Linkup uint8		`json:"linkup"`		// Link up
	Dipsw  uint8		`json:"dipsw"`		// Dip SW register
	Led    uint8		`json:"led"`		// 7seg register
	Chup   uint8		`json:"chup"`		// Ch. up
	Done   uint8		`json:"done"`		// FPGA done
	Pwr    uint8		`json:"pwr"`		// PWR OK
}

func monitor_get_status()(st FicStat, err error) {
	err = gpio.Gpio_lock()
	if err != nil {
		return st, err
	}
	defer gpio.Gpio_unlock()

	st.Ts		= time.Now()
	st.State, err	= fic_read8(FIC_REG_ST)
	st.Hls, err	= fic_read8(FIC_REG_HLS)
	st.Linkup, err	= fic_read8(FIC_REG_LINKUP)
	st.Dipsw, err	= fic_read8(FIC_REG_DIPSW)
	st.Led, err	= fic_read8(FIC_REG_LED)
	st.Chup, err	= fic_read8(FIC_REG_CHUP)
	st.Done		= uint8(gpio.Get_pin(PIN["RP_DONE"]))
	st.Pwr		= uint8(gpio.Get_pin(PIN["RP_PWOK"]))

	return st, err
}

//-----------------------------------------------------------------------------
// Kernel
//-----------------------------------------------------------------------------
func monitor_daemon() {
	listener, err := net.Listen("tcp", LISTEN_ADDR)
	if err != nil {
		log.Fatal("Can't listen", err)
	}
	defer listener.Close()

	// Obtain monitor status async
	mon, err := monitor_get_status()
	if err != nil {
		fmt.Println("DEBUG: FiC STATUS GET ERROR (INITIAL)", err)
	}
	mon_t0 := time.Now()

	// kernel loop
	for {
		// Refresh monitor info
		if time.Now().Sub(mon_t0).Seconds() > GET_STATUS_PEIROD  {
			mon, err = monitor_get_status()
			mon_t0 = time.Now()
			if err != nil {
				fmt.Println("DEBUG: FiC STATUS GET ERROR (PERIOD)", err)
			}
		}

		// Socket accept
		conn, err := listener.Accept()
		if err != nil {
			log.Fatal("Can't accept", err)
		}

		go monitor_sock_conn(conn, &mon)	// launch thread
	}
}

func monitor_resp_ok(conn net.Conn) {
	conn.Write([]byte("OK\r\n"))
}

func monitor_resp_err(conn net.Conn) {
	conn.Write([]byte("ERROR\r\n"))
}

func monitor_sock_conn(conn net.Conn, mon *FicStat) {
	defer conn.Close()

	// Terminal commands
	const (
		TERM_CMD_STAT     = "STAT"
		TERM_CMD_PROG     = "PROG"
		TERM_CMD_PROG_PR  = "PROGPR"
		TERM_CMD_PROG8    = "PROG8"
		TERM_CMD_PROG8_PR = "PROG8PR"
		TERM_CMD_START    = "STAT"
		TERM_CMD_RESET    = "RESET"
		TERM_CMD_WRITE    = "WRITE"
		TERM_CMD_READ     = "READ"
		TERM_CMD_HELP     = "HELP"
		TERM_CMD_INIT     = "INIT"	// FPGA INIT
	)

	buf := make([]byte, 8*1024)

	fmt.Println("FiCDaemon: Listen on ", LISTEN_ADDR)
	for {
		monitor_resp_ok(conn)	// Ready for recieve CMD
		n, err := conn.Read(buf)
		if n == 0 {
			break
		}
		if err != nil {
			monitor_resp_err(conn)
			log.Fatal("ERROR: Read buffer error", err)
		}

		b := strings.Fields(string(buf[:n]))

		switch b[0] {
		// Report status
		case TERM_CMD_STAT:
			fmt.Println("DEBUG: STAT")

			//st, err := monitor_get_status()
			//if err != nil {
			//	fmt.Println("DEBUG: STATUS GET ERROR", err)
			//}

			jsonbyte, err := json.Marshal(*mon)
			if err != nil {
				fmt.Println("DEBUG: JSON ERROR")
				monitor_resp_err(conn)
				break
			}

			conn.Write(append(jsonbyte, []byte("\r\n")...))

		// FPGA Configuration
		case TERM_CMD_PROG, TERM_CMD_PROG8, TERM_CMD_PROG_PR, TERM_CMD_PROG8_PR:
			fmt.Println("DEBUG: PROG", len(b))
			if len(b) < 2 {
				fmt.Println("DEBUG: PROG ARG ERROR")
				monitor_resp_err(conn)
				break
			}
			monitor_resp_ok(conn)

			// 2nd argument is recive data size
			rcvsize, err := strconv.Atoi(b[1])
			if err != nil {
				fmt.Println("DEBUG: PROG ARG SIZE ERROR")
				monitor_resp_err(conn)
				break
			}
			fmt.Println("DEBUG: RCV SIZE", rcvsize)
			rcvdata := make([]byte, rcvsize)
			monitor_resp_ok(conn)

			// Receive FPGA bitstream data
			i := 0
			for i < rcvsize {
				n, err := conn.Read(buf)
				if n == 0 {
					//fmt.Println("DEBUG: read", n)
					break
				}
				if err != nil {
					monitor_resp_err(conn)
					fmt.Println("ERROR: Read buffer error", err)
					break
				}
				//fmt.Println("DEBUG: read", n)
				copy(rcvdata[i:], buf)
				i += n
			}

			fmt.Println("DEBUG: RCVD SIZE", i)

			if i != rcvsize {
				fmt.Println("DEBUG: Recived bitstream size mismatch ", rcvsize - i)
				monitor_resp_err(conn)
				break
			}

			// Send to FPGA
			switch b[0] {
			case TERM_CMD_PROG, TERM_CMD_PROG_PR:
				pr := false
				if b[0] == TERM_CMD_PROG_PR {
					pr = true
				}

				if err := Prog16(rcvdata, pr); err != nil {
					monitor_resp_err(conn)
					fmt.Println("ERROR: FPGA programming error", err)
					break
				}

			case TERM_CMD_PROG8, TERM_CMD_PROG8_PR:
				pr := false
				if b[0] == TERM_CMD_PROG8_PR {
					pr = true
				}

				if err := Prog8(rcvdata, pr); err != nil {
					monitor_resp_err(conn)
					fmt.Println("ERROR: FPGA programming error", err)
					break
				}
			}

			fmt.Println("DEBUG: PROG DONE")

		// Register Write
		case TERM_CMD_WRITE:
			fmt.Println("DEBUG: WRITE")
			if len(b) < 3 {
				fmt.Println("DEBUG: WRITE ARG ERROR")
				monitor_resp_err(conn)
				break
			}
			// 2nd argument is write 1b address
			addr, err := strconv.ParseInt(b[1], 16, 32)
			if err != nil {
				fmt.Println("DEBUG: WRITE ARG ADDR ERROR", err)
				monitor_resp_err(conn)
				break
			}
			// 3rd argument is write data (1byte)
			data, err := strconv.ParseInt(b[2], 16, 32)
			if err != nil {
				fmt.Println("DEBUG: WRITE ARG DATA ERROR", err)
				monitor_resp_err(conn)
				break
			}
			fic_write8(uint16(addr), uint8(data))

		// Register Read
		case TERM_CMD_READ:
			fmt.Println("DEBUG: READ")
			if len(b) < 2 {
				fmt.Println("DEBUG: READ ARG ERROR")
				monitor_resp_err(conn)
				break
			}
			// 2nd argument is read address
			addr, err := strconv.ParseInt(b[1], 16, 32)
			if err != nil {
				fmt.Println("DEBUG: READ ARG ADDR ERROR", err)
				monitor_resp_err(conn)
				break
			}

			data, err := fic_read8(uint16(addr))
			//data, err := fic_read4(uint8(addr))
			if err != nil {
				fmt.Println("DEBUG: READ DATA ERROR", err)
				monitor_resp_err(conn)
				break
			}

			// send back
			conn.Write([]byte(strconv.FormatInt(int64(data), 16)+"\r\n"))

		// FPGA reset
		case TERM_CMD_INIT:
			fmt.Println("DEBUG: INIT")
			fic_fpga_init()
		}
	}

	fmt.Println("DEBUG: Disconnected from", conn.RemoteAddr())
}

//-----------------------------------------------------------------------------
// main
//-----------------------------------------------------------------------------
func init() {
	fmt.Println("")
	fmt.Println("FiC monitor daemon")
	fmt.Println("nyacom (C) 2018.05 <kzh@nyacom.net>")
}

func main() {
	 gpio.Setup()	// GPIO setup (mmap)
	 monitor_daemon()

	// ---- R/W test ----
	// fmt.Printf("%x\n", fic_read8(0xfffc))
	//data, err := fic_read8(0xfffc)
	//if err != nil {
	//	fmt.Println(err)
	//}
	//fmt.Printf("Read Data %x\n", data)

	//fic_write8(0xffff, 0xab)
	//data, err := fic_read8(0xffff)
	//if err != nil {
	//	fmt.Println(err)
	//}
	//fmt.Printf("Data %x\n", data)

	// fmt.Printf("LED: %x\n", fic_read8(0xfffb))

	// ---- FPGA prog test ----
//	gpio_prog_setup() // GPIO setup for FPGA prog

//
//	fic_prog(infile)
}

