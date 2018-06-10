//-----------------------------------------------------------------------------
// nyacom FiC monitor daemon (C) 2018.05
// <kzh@nyacom.net>
//-----------------------------------------------------------------------------
package main

import (
	"fmt"
	"errors"
//	"flag"
	"log"
	"os"
	"time"
	"net"		// socket
	"strings"
	"strconv"
	"encoding/json"
	"./gpio"	// RPi GPIO lib
//	"unsafe"
//	"reflect"
//	"syscall"
)

//-----------------------------------------------------------------------------
const (
	LOCKFILE = "/tmp/gpio.lock"
	TIMEOUT = 10		// Lockfile timeout
	LOCKEXPIRE = 120	// Lock file lifetime (every command should have done within 120sec)

	BUFSIZE = (1*1024*1024)

	// FiC SW Communication
	COM_TIMEOUT = 1

	// FiC SW bus direction
	COM_DIR_SND = 0
	COM_DIR_RCV = 1

	// FiC SW commands
	COM_CMD_WRITE = 0x02
	COM_CMD_READ = 0x03

	COM_MASK = 0x00cfff00

	// TCP config
	LISTEN_ADDR = "0.0.0.0:4000"
)

//-----------------------------------------------------------------------------
// FiC registers
//-----------------------------------------------------------------------------
const (
	FIC_REG_ST     = 0xffff
	FIC_REG_HLS    = 0xfffe
	FIC_REG_LINKUP = 0xfffd
	FIC_REG_DIPSW  = 0xfffc
	FIC_REG_LED    = 0xfffb
	FIC_REG_CHUP   = 0xfffa
)

//-----------------------------------------------------------------------------
// PRi PINS
//-----------------------------------------------------------------------------
var PIN = map[string] uint32 {
	"RP_INIT" : 4,
	"RP_PROG" : 5,
	"RP_DONE" : 6,
	"RP_CCLK" : 7,

	"RP_CD0" : 8,
	"RP_CD1" : 9,
	"RP_CD2" : 10,
	"RP_CD3" : 11,
	"RP_CD4" : 12,
	"RP_CD5" : 13,
	"RP_CD6" : 14,
	"RP_CD7" : 15,
	"RP_CD8" : 16,
	"RP_CD9" : 17,
	"RP_CD10" : 18,
	"RP_CD11" : 19,
	"RP_CD12" : 20,
	"RP_CD13" : 21,
	"RP_CD14" : 22,
	"RP_CD15" : 23,
	"RP_CD16" : 24,
	"RP_CD17" : 25,

	"RP_PWOK" : 24,
	"RP_G_CKSEL" : 25,
	"RP_CSI" : 26,
	"RP_RDWR" : 27,
}

var PIN_BIT = map[string] uint32 {
	"RP_INIT" : (1 << PIN["RP_INIT"]),
	"RP_PROG" : (1 << PIN["RP_PROG"]),
	"RP_DONE" : (1 << PIN["RP_DONE"]),
	"RP_CCLK" : (1 << PIN["RP_CCLK"]),

	"RP_CD0" : (1 << PIN["RP_CD0"]),
	"RP_CD1" : (1 << PIN["RP_CD1"]),
	"RP_CD2" : (1 << PIN["RP_CD2"]),
	"RP_CD3" : (1 << PIN["RP_CD3"]),
	"RP_CD4" : (1 << PIN["RP_CD4"]),
	"RP_CD5" : (1 << PIN["RP_CD5"]),
	"RP_CD6" : (1 << PIN["RP_CD6"]),
	"RP_CD7" : (1 << PIN["RP_CD7"]),
	"RP_CD8" : (1 << PIN["RP_CD8"]),
	"RP_CD9" : (1 << PIN["RP_CD9"]),
	"RP_CD10" : (1 << PIN["RP_CD10"]),
	"RP_CD11" : (1 << PIN["RP_CD11"]),
	"RP_CD12" : (1 << PIN["RP_CD12"]),
	"RP_CD13" : (1 << PIN["RP_CD13"]),
	"RP_CD14" : (1 << PIN["RP_CD14"]),
	"RP_CD15" : (1 << PIN["RP_CD15"]),
	"RP_CD16" : (1 << PIN["RP_CD16"]),
	"RP_CD17" : (1 << PIN["RP_CD17"]),

	"RP_PWOK" : (1 << PIN["RP_PWOK"]),
	"RP_G_CKSEL" : (1 << PIN["RP_G_CKSEL"]),
	"RP_CSI" : (1 << PIN["RP_CSI"]),
	"RP_RDWR" : (1 << PIN["RP_RDWR"]),
}

var PIN_COMM = map[string] uint32 {
	"RREQ" : PIN["RP_CD15"],
	"RSTB" : PIN["RP_CD14"],
	"FREQ" : PIN["RP_CD13"],
	"FACK" : PIN["RP_CD12"],

	"DATA7" : PIN["RP_CD11"],
	"DATA6" : PIN["RP_CD10"],
	"DATA5" : PIN["RP_CD9"],
	"DATA4" : PIN["RP_CD8"],
	"DATA3" : PIN["RP_CD7"],
	"DATA2" : PIN["RP_CD6"],
	"DATA1" : PIN["RP_CD5"],
	"DATA0" : PIN["RP_CD4"],
}

//-----------------------------------------------------------------------------
// Create lockfile before GPIO operation
//-----------------------------------------------------------------------------
func gpio_lock()(err error){
	// Lock wait
	t1 := time.Now()
	for stat, err := os.Stat(LOCKFILE); !os.IsNotExist(err); {
		// Check if the lockfile is too old -> bug?
		if (time.Now().Sub(stat.ModTime())).Seconds() > LOCKEXPIRE {
			break
		}

		time.Sleep(1 * time.Second)

		t2 := time.Now()
		if (t2.Sub(t1)).Seconds() > TIMEOUT {
			return errors.New("Lockfile time out")
		}
	}

	fd, err := os.OpenFile(LOCKFILE, os.O_CREATE, 0666);
	defer fd.Close()
	if err != nil {
		return errors.New("Cant open Lockfile")
	}

	return nil
}

func gpio_unlock() {
	if err:= os.Remove(LOCKFILE); err != nil {
		log.Fatal(err)
	}
}

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
// set GPIO data pin dir
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

func comm_wait_fack_down() error {
	// Wait for ACK from FiC
	t1 := time.Now()
	for gpio.Get_pin(PIN_COMM["FACK"]) == 1 {
		time.Sleep(1 * time.Millisecond)
		t2 := time.Now()
		if (t2.Sub(t1).Seconds() > COM_TIMEOUT) {
			return errors.New("Communication time out")
		}
	}
	return nil
}

func comm_wait_fack_up() error {
	// Wait for ACK from FiC
	t1 := time.Now()
	for gpio.Get_pin(PIN_COMM["FACK"]) == 0 {
		time.Sleep(1 * time.Millisecond)
		t2 := time.Now()
		if (t2.Sub(t1).Seconds() > COM_TIMEOUT) {
			return errors.New("Communication time out")
		}
	}
	return nil
}

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

func comm_receive(bus uint32)(b uint8, err error) {
	gpio.Clr_bus(^(bus & COM_MASK))
	gpio.Set_bus(bus & COM_MASK)

	err = comm_wait_fack_up()	// Wait FiC ack up
	if err != nil {
		return 0, err
	}

	b = uint8((gpio.Get_bus() >> PIN_COMM["DATA0"]) & 0xff)	// Receive data

	err = comm_wait_fack_down()	// Wait FiC ack down
	if err != nil {
		return 0, err
	}

	return b, nil
}

//-----------------------------------------------------------------------------
// User module control
//-----------------------------------------------------------------------------
// func fic_module_reset() {
// }
// func fic_module_start() {
// }

//-----------------------------------------------------------------------------
// Write 1Byte 
//-----------------------------------------------------------------------------
func fic_write8(addr uint16, data uint8) error {
	gpio_comm_setup()
	comm_dir(COM_DIR_SND)

	// Send Handshake and CMD
	bus := uint32((1<<PIN_COMM["RREQ"])|(1<<PIN_COMM["RSTB"])|(COM_CMD_WRITE<<PIN_COMM["DATA0"]))
	err := comm_send(bus)
	if err != nil {
		return err
	}

	// Send address high
	bus = (1<<PIN_COMM["RREQ"])|(1<<PIN_COMM["RSTB"])|((uint32(addr)>>8)<<PIN_COMM["DATA0"])
	err = comm_send(bus)
	if err != nil {
		return err
	}

	// Send address low
	bus = (1<<PIN_COMM["RREQ"])|(1<<PIN_COMM["RSTB"])|((uint32(addr)&0xff)<<PIN_COMM["DATA0"])
	err = comm_send(bus)
	if err != nil {
		return err
	}

	// Send data
	bus = (1<<PIN_COMM["RREQ"])|(1<<PIN_COMM["RSTB"])|(uint32(data)<<PIN_COMM["DATA0"])
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
	bus := uint32((1<<PIN_COMM["RREQ"])|(1<<PIN_COMM["RSTB"])|(COM_CMD_READ<<PIN_COMM["DATA0"]))
	err = comm_send(bus)
	if err != nil {
		return 0, err
	}

	// Send address high
	bus = (1<<PIN_COMM["RREQ"])|(1<<PIN_COMM["RSTB"])|((uint32(addr)>>8)<<PIN_COMM["DATA0"])
	err = comm_send(bus)
	if err != nil {
		return 0, err
	}

	// Send address low
	bus = (1<<PIN_COMM["RREQ"])|(1<<PIN_COMM["RSTB"])|((uint32(addr)&0xff)<<PIN_COMM["DATA0"])
	err = comm_send(bus)
	if err != nil {
		return 0, err
	}

	comm_dir(COM_DIR_RCV)	// Switch bus direction

	// recive data1
	bus = (1<<PIN_COMM["RREQ"])|(1<<PIN_COMM["RSTB"])
	b, err = comm_receive(bus)
	if err != nil {
		return 0, err
	}

	bus = (1<<PIN_COMM["RREQ"])|(1<<PIN_COMM["RSTB"])
	gpio.Clr_bus(^bus) // Negate REQ and STB

	defer gpio.Set_all_input()

	return b, nil
}

//-----------------------------------------------------------------------------
// FPGA reset
//-----------------------------------------------------------------------------
func fic_fpga_init() {
	gpio_prog_setup() // GPIO setup for FPGA prog
	defer gpio.Set_all_input()
	gpio.Set_bus(PIN_BIT["RP_PROG"])
	gpio.Clr_bus(PIN_BIT["RP_PROG"])
}

//-----------------------------------------------------------------------------
// FPGA programmer
//-----------------------------------------------------------------------------
func fic_prog(bitstream []byte)(err error) {
	err = gpio_lock()
	defer gpio_unlock()
	if err != nil {
		return err
	}

	gpio_prog_setup() // GPIO setup for FPGA prog
	defer gpio.Set_all_input()

	fmt.Println("PROG: Entering Xilinx SelectMap x16 configuration mode...")

	// Invoke configuration
	gpio.Set_bus((PIN_BIT["RP_PROG"]))
	gpio.Clr_bus((PIN_BIT["RP_PROG"]|PIN_BIT["RP_CSI"]|PIN_BIT["RP_RDWR"]))
	gpio.Set_bus((PIN_BIT["RP_PROG"]))

	t1 := time.Now()
	for gpio.Get_pin(PIN["RP_INIT"]) == 0 {
		time.Sleep(1 * time.Second)
		if (time.Now().Sub(t1).Seconds() > COM_TIMEOUT) {
			return errors.New("Communication time out")
		}
	}

	fmt.Println("PROG: Ready to program")

	file_size := len(bitstream)
	fmt.Println("PROG: File size : ", file_size, " B")

	gpio.Clr_bus((PIN["RP_CCLK"]))

	fmt.Println("PROG: Programming...")

	buf := bitstream

	for i := 0; i < len(buf); i += 2 {
		data := (uint32(buf[i+1]) << 8 | uint32(buf[i])) << 8
		gpio.Clr_bus((^data & 0x00ffff00) | PIN_BIT["RP_CCLK"])
		gpio.Set_bus((data & 0x00ffff00))
		gpio.Set_bus(PIN_BIT["RP_CCLK"])

		if gpio.Get_pin(PIN["RP_INIT"]) == 0 {
			return errors.New("Configuraion Error (while prog)")
		}
	}

	//gpio.Clr_bus(PIN_BIT["RP_CCLK"])	// Negate CLK

	fmt.Println("PROG: Waiting FPGA done")

	for gpio.Get_pin(PIN["RP_DONE"]) == 0 {		// Wait until RP_DONE asserted
		if gpio.Get_pin(PIN["RP_INIT"]) == 0 {
			return errors.New("Configuration Error (while waiting)")
		}
		gpio.Set_bus(PIN_BIT["RP_CCLK"])
		gpio.Clr_bus(PIN_BIT["RP_CCLK"])
	}

	gpio.Clr_bus(0x00ffff00 | PIN_BIT["RP_CCLK"])
	fmt.Println("PROG: FPGA program done")

	return nil
}
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
	err = gpio_lock()
	defer gpio_unlock()
	if err != nil {
		return st, err
	}

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
		fmt.Println("DEBUG: STATUS GET ERROR", err)
	}
	mon_t0 := time.Now()

	// kernel roop
	for {
		// Refresh monitor info
		if time.Now().Sub(mon_t0).Seconds() > 5 {
			mon, err = monitor_get_status()
			mon_t0 = time.Now()
			if err != nil {
				fmt.Println("DEBUG: STATUS GET ERROR", err)
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

func monitor_sock_conn(conn net.Conn, mon *FicStat) {
	defer conn.Close()

	// Terminal commands
	const (
		TERM_CMD_STAT  = "STAT"
		TERM_CMD_PROG  = "PROG"
		TERM_CMD_START = "STAT"
		TERM_CMD_RESET = "RESET"
		TERM_CMD_WRITE = "WRITE"
		TERM_CMD_READ  = "READ"
		TERM_CMD_HELP  = "HELP"
		TERM_CMD_INIT  = "INIT"	// FPGA INIT
	)

	buf := make([]byte, 8*1024)

	fmt.Println("FiCDaemon: Listen on ", LISTEN_ADDR)
	for {
		conn.Write([]byte("OK\r\n"))
		n, err := conn.Read(buf)
		if n == 0 {
			break
		}
		if err != nil {
			conn.Write([]byte("ERROR\r\n"))
			log.Fatal("ERROR: Read buffer error", err)
		}

		b := strings.Fields(string(buf[:n]))

		switch b[0] {
			// Report status
			case TERM_CMD_STAT: {
				fmt.Println("DEBUG: STAT")

				//st, err := monitor_get_status()
				//if err != nil {
				//	fmt.Println("DEBUG: STATUS GET ERROR", err)
				//}

				jsonbyte, err := json.Marshal(*mon)
				if err != nil {
					fmt.Println("DEBUG: JSON ERROR")
					conn.Write([]byte("ERROR\r\n"))
					break
				}

				conn.Write(append(jsonbyte, []byte("\r\n")...))
			}

			// FPGA Configuration
			case TERM_CMD_PROG: {
				fmt.Println("DEBUG: PROG", len(b))
				if len(b) < 2 {
					fmt.Println("DEBUG: PROG ARG ERROR")
					conn.Write([]byte("ERROR\r\n"))
					break
				}
				conn.Write([]byte("OK\r\n"))

				// 2nd argument is recive data size
				rcvsize, err := strconv.Atoi(b[1])
				if err != nil {
					fmt.Println("DEBUG: PROG ARG SIZE ERROR")
					conn.Write([]byte("ERROR\r\n"))
					break
				}
				fmt.Println("DEBUG: RCV SIZE", rcvsize)
				rcvdata := make([]byte, rcvsize)
				conn.Write([]byte("OK\r\n"))

				// Receive FPGA bitstream data
				i := 0
				for i < rcvsize {
					n, err := conn.Read(buf)
					if n == 0 {
						break
					}
					if err != nil {
						conn.Write([]byte("ERROR\r\n"))
						log.Fatal("ERROR: Read buffer error", err)
						break
					}
					//fmt.Println("DEBUG: read", n)
					copy(rcvdata[i:], buf)
					i += n
				}

				fmt.Println("DEBUG: RCVD SIZE", i)

				// Send to FPGA
				err = fic_prog(rcvdata)
				if err != nil {
					conn.Write([]byte("ERROR\r\n"))
					log.Fatal("ERROR: FPGA programming error", err)
					break
				}

				fmt.Println("DEBUG: PROG DONE")
			}

			// Register Write
			case TERM_CMD_WRITE: {
				fmt.Println("DEBUG: WRITE")
				if len(b) < 3 {
					fmt.Println("DEBUG: WRITE ARG ERROR")
					conn.Write([]byte("ERROR\r\n"))
					break
				}
				// 2nd argument is write address
				addr, err := strconv.ParseInt(b[1], 16, 32)
				if err != nil {
					fmt.Println("DEBUG: WRITE ARG ADDR ERROR", err)
					conn.Write([]byte("ERROR\r\n"))
					break
				}
				// 3rd argument is write data (1byte)
				data, err := strconv.ParseInt(b[2], 16, 32)
				if err != nil {
					fmt.Println("DEBUG: WRITE ARG DATA ERROR", err)
					conn.Write([]byte("ERROR\r\n"))
					break
				}
				fic_write8(uint16(addr), uint8(data))
			}

			// Register Read
			case TERM_CMD_READ: {
				fmt.Println("DEBUG: READ")
				if len(b) < 2 {
					fmt.Println("DEBUG: READ ARG ERROR")
					conn.Write([]byte("ERROR\r\n"))
					break
				}
				// 2nd argument is read address
				addr, err := strconv.ParseInt(b[1], 16, 32)
				if err != nil {
					fmt.Println("DEBUG: READ ARG ADDR ERROR", err)
					conn.Write([]byte("ERROR\r\n"))
					break
				}

				data, err := fic_read8(uint16(addr))
				if err != nil {
					fmt.Println("DEBUG: READ DATA ERROR", err)
					conn.Write([]byte("ERROR\r\n"))
					break
				}

				fmt.Println(data)
				conn.Write([]byte(strconv.FormatInt(int64(data), 16)+"\r\n"))
			}

			// FPGA reset
			case TERM_CMD_INIT: {
				fmt.Println("DEBUG: INIT")
				fic_fpga_init()
			}
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

	//f := flag.NewFlagSet(os.Args[0], flag.ExitOnError)
	//fileOpt := f.String(" ", "default", "*.mcs file for FPGA")

	//f.Parse(os.Args[1:])
	//for 0 < f.NArg() {
	//	f.Parse(f.Args()[1:])
	//}

	// Check arguments
	//if len(os.Args) < 2 {
	//	help_str()
	//	log.Fatal("Insufficient argument")
	//}
}

//func help_str() {
//	fmt.Println("Usage: ficprog.go INPUT_FILE.bin")
//}

func main() {
	//infile := os.Args[1]
	//fmt.Println("Filename: ", infile)

	 //gpio_lock();
	 //defer gpio_unlock();
	 gpio.Setup()	// GPIO setup (mmap)
	 monitor_daemon()

	// ---- R/W test ----
	// fmt.Printf("%x\n", fic_read8(0xfffc))

	// fic_write8(0xffff, 0xaa)
	// fmt.Printf("%x\n", fic_read8(0xffff))

	// fmt.Printf("LED: %x\n", fic_read8(0xfffb))

	// ---- FPGA prog test ----
//	gpio_prog_setup() // GPIO setup for FPGA prog

//
//	fic_prog(infile)
}

