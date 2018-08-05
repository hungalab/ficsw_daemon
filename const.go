package main

//-----------------------------------------------------------------------------
const (
	// FiC status get period in sec
	GET_STATUS_PEIROD = 5

	// FiC SW Communication
	COM_TIMEOUT = 5

	// FiC SW bus direction
	COM_DIR_SND = 0
	COM_DIR_RCV = 1

	// FiC SW commands
	COM_CMD_WRITE = 0x02
	COM_CMD_READ = 0x03

	COM_MASK = 0x00cfff00

	// TCP config
	LISTEN_ADDR = "0.0.0.0:4000"

	BUFSIZE = (1*1024*1024)
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
	"RP_PWOK" : (1 << PIN["RP_PWOK"]),	// Input
	"RP_INIT" : (1 << PIN["RP_INIT"]),
	"RP_DONE" : (1 << PIN["RP_DONE"]),
	"RP_G_CKSEL" : (1 << PIN["RP_G_CKSEL"]),

	"RP_CD0" : (1 << PIN["RP_CD0"]),	// Output
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

	"RP_PROG" : (1 << PIN["RP_PROG"]),
	"RP_CCLK" : (1 << PIN["RP_CCLK"]),
	"RP_CSI" : (1 << PIN["RP_CSI"]),
	"RP_RDWR" : (1 << PIN["RP_RDWR"]),
}

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

