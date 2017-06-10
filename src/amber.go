package main

import "gopkg.in/cheggaaa/pb.v1"
import "github.com/fatih/color"
import "math/rand"
import "io/ioutil"
import "debug/pe"
import "strconv"
import "os/exec"
import "runtime"
import "strings"
import "time"
import "fmt"
import "os"

const VERSION string = "1.0.0"

type peID struct {
  
  // Parameters...
  fileName string 
  keySize int
  key []byte
  staged bool
  iat bool
  resource bool
  verbose bool

  //Analysis...
  imageBase uint32
  subsystem uint16
  VP string
  GPA string
  LLA string

}

var red *color.Color = color.New(color.FgRed)
var boldRed *color.Color = red.Add(color.Bold)
var blue *color.Color = color.New(color.FgBlue)
var boldBlue *color.Color = blue.Add(color.Bold)
var yellow *color.Color = color.New(color.FgYellow)
var boldYellow *color.Color = yellow.Add(color.Bold)
var green *color.Color = color.New(color.FgGreen)
var boldGreen *color.Color = green.Add(color.Bold)

var progressBar *pb.ProgressBar
var peid peID

func main() {

  	runtime.GOMAXPROCS(runtime.NumCPU())

 	peid.keySize = 8
  	peid.staged = false
  	peid.resource = true
  	peid.verbose = false
 	peid.iat = false

 	ARGS := os.Args[1:]

  	if len(ARGS) == 0 || ARGS[0] == "--help" || ARGS[0] == "-h"{
    	Banner()
    	Help()
    	os.Exit(0)
  	}

  	Banner()
  	peid.fileName = ARGS[0]

  	for i := 0; i < len(ARGS); i++{
  		if ARGS[i] == "-ks" || ARGS[i] == "--keysize" {
  			ks, Err := strconv.Atoi(ARGS[i+1])
      		if Err != nil {
        		boldRed.Println("\n[!] ERROR: Invalid key size.\n")
        		fmt.Println(Err)
        		os.Exit(1)
      		}else{
        		peid.keySize = ks
      		} 
  		}
  		if ARGS[i] == "-k" || ARGS[i] == "--key" {
  			peid.key = []byte(ARGS[i+1]) 
  		}
  		if ARGS[i] == "--staged" {
  			peid.staged = true 
  		}
  		if ARGS[i] == "--no-resource" {
  			peid.resource = false 
  		}
    	if ARGS[i] == "-v" || ARGS[i] == "--verbose" {
      		peid.verbose = true 
    	}
  	}

  	boldYellow.Print("\n[*] File: ")
  	boldBlue.Println(peid.fileName)
  	boldYellow.Print("[*] Staged: ")
  	boldBlue.Println(peid.staged)
  	if len(peid.key) != 0 {
    	boldYellow.Print("[*] Key: ")
    	boldBlue.Println(peid.key)
  	}else{
    	boldYellow.Print("[*] Key Size: ")
    	boldBlue.Println(peid.keySize)   
  	}
  	boldYellow.Print("[*] IAT: ")
  	boldBlue.Println(peid.iat)
  	boldYellow.Print("[*] Verbose: ")
  	boldBlue.Println(peid.verbose,"\n")


	createBar()
  	checkRequired() // 8 steps

  	file, fileErr := pe.Open(ARGS[0])
  	if fileErr != nil {
    	boldRed.Println("\n[!] ERROR: Can't open file.")
    	boldRed.Println(fileErr)
    	os.Exit(1)   
  	}
  	progress()

  	analyze(file) // 6 steps
  	progress()
  	assemble() // 8 steps

  	if peid.staged == true {
  		exec.Command("sh", "-c", string("mv Payload "+peid.fileName+".stage")).Run()
  	}else{
  		compile() // 9 steps
  	}
  	clean() // 7 steps

  	progressBar.Finish()

  	if peid.staged == true {
  		boldGreen.Println("[+] Stage generated !\n")	
  	}else{
  		boldGreen.Println("[+] File successfully crypted !\n")	
  	}


}
//################################################### PARSE ###################################################

func analyze(file *pe.File) {
  //Do analysis on pe file...

  if file.FileHeader.Machine != 0x14C {
      boldRed.Println("\n[!] ERROR: File is not a 32 bit PE.")
      os.Exit(1) 
  } 
  progress()
  var OPT *pe.OptionalHeader32 = file.OptionalHeader.(*pe.OptionalHeader32)
  // PE32 = 0x10B
  if OPT.Magic != 0x10B { 
      boldRed.Println("\n[!] ERROR: File is not a valid PE.")
      os.Exit(1) 
  }
  progress()
  peid.imageBase = OPT.ImageBase
  progress()
  peid.subsystem = OPT.Subsystem
  progress()
  if (OPT.DataDirectory[11].Size) != 0x00 {
      boldRed.Println("\n[!] ERROR: File has bounded imports.")
      os.Exit(1) 
  }
  progress()
  if (OPT.DataDirectory[13].Size) != 0x00 {
      boldRed.Println("\n[!] ERROR: File has delayed imports.")
      os.Exit(1) 
  }

  progress()
  if peid.verbose == true {
    boldYellow.Printf("[*] Machine: %X\n", file.FileHeader.Machine)
    boldYellow.Printf("[*] Magic: %X\n", OPT.Magic)
    boldYellow.Printf("[*] Subsystem: %X\n", OPT.Subsystem)
    boldYellow.Printf("[*] Image Base: %X\n", peid.imageBase)
    boldYellow.Printf("[*] Size Of Image: %X\n", OPT.SizeOfImage)
    boldYellow.Printf("[*] Import Table: %X\n", (OPT.DataDirectory[1].VirtualAddress+OPT.ImageBase))
    boldYellow.Printf("[*] Import Address Table: %X\n", (OPT.DataDirectory[12].VirtualAddress+OPT.ImageBase))
  }


}

/*
func parseIAT(file *pe.File) {
	// Parse the IAT and find required function addresses
}
*/



//################################################### BUILD ###################################################

func assemble() {

  MapPE, _ := exec.Command("sh", "-c", string("wine MapPE.exe "+peid.fileName)).Output()
  if strings.Contains(string(MapPE), "[!]") {
    boldRed.Println("\n[!] ERROR: While mapping pe file :(")
    boldRed.Println(string(MapPE))
    clean()
    os.Exit(1)      
  }
  progress()

  if peid.iat == false {
  	moveMap, moveMapErr := exec.Command("sh", "-c", "mv Mem.map ReplaceProcess/peb-based/").Output()
  	if moveMapErr != nil {
    	boldRed.Println("\n[!] ERROR: While moving the file map")
    	boldRed.Println(string(moveMap))
    	clean()
    	os.Exit(1)      
  	}

  	progress()
  	nasm, Err := exec.Command("sh", "-c", "cd ReplaceProcess/peb-based && nasm -f bin ReplaceProcess.asm -o Payload").Output()
  	if Err != nil {
    	boldRed.Println("\n[!] ERROR: While assembling payload :(")
    	boldRed.Println(string(nasm))
    	boldRed.Println(Err)
    	clean()
    	os.Exit(1)    
  	}

  	progress()

  	movePayload, movePayErr := exec.Command("sh", "-c", "mv ReplaceProcess/peb-based/Payload ./").Output()
  	if movePayErr != nil {
    	boldRed.Println("\n[!] ERROR: While moving the payload")
    	boldRed.Println(string(movePayload))
    	boldRed.Println(Err)
    	clean()
    	os.Exit(1)    
  	}
  	progress() 
  }else{
  	moveMap, moveMapErr := exec.Command("sh", "-c", "mv Mem.map ReplaceProcess/iat-based/").Output()
  	if moveMapErr != nil {
    	boldRed.Println("\n[!] ERROR: While moving the file map")
    	boldRed.Println(string(moveMap))
    	clean()
    	os.Exit(1)      
  	}

  	progress()
  	nasm, Err := exec.Command("sh", "-c", "cd ReplaceProcess/iat-based && nasm -f bin ReplaceProcess.asm -o Payload").Output()
  	if Err != nil {
    	boldRed.Println("\n[!] ERROR: While assembling payload :(")
    	boldRed.Println(string(nasm))
    	boldRed.Println(Err)
    	clean()
    	os.Exit(1)    
  	}

  	progress()

  	movePayload, movePayErr := exec.Command("sh", "-c", "mv ReplaceProcess/iat-based/Payload ./").Output()
  	if movePayErr != nil {
    	boldRed.Println("\n[!] ERROR: While moving the payload")
    	boldRed.Println(string(movePayload))
    	boldRed.Println(Err)
    	clean()
    	os.Exit(1)    
  	}
  	progress() 
  }

  if peid.verbose == true {
    _MapPE := strings.Split(string(MapPE), "github.com/egebalci/mappe")
    fmt.Println(string(_MapPE[1]))
  }


  if strings.Contains(string(MapPE), "[!] Enpty Import Table........................... [NULL]") {
    boldRed.Println("\n[!] ERROR: File has a empty import table :(")
    clean()
    os.Exit(1)
  }
  progress()

}


func compile() {

    if peid.verbose == true {
      boldYellow.Println("[*] Ciphering payload...")    
    }
    crypt() // 6 steps
    progress() 

    xxd := exec.Command("sh", "-c", "rm Payload && mv Payload.xor Payload && xxd -i Payload > Stub/PAYLOAD.h")
  xxd.Stdout = os.Stdout
    xxd.Stderr = os.Stderr
    xxd.Run()

    progress()  

    _xxd := exec.Command("sh", "-c", "xxd -i Payload.key > Stub/KEY.h")
    _xxd.Stdout = os.Stdout
    _xxd.Stderr = os.Stderr
    _xxd.Run()

    progress()  

    if peid.verbose == true {
      key, _ := exec.Command("sh", "-c", "xxd -i Payload.key").Output() 
      boldYellow.Println("[*] Payload ciphered with: ")
      boldBlue.Println(string(key))    
    }
}

//################################################### CRYPT ###################################################

func crypt() {
  
    if peid.verbose == true {
      boldYellow.Println("[*] Ciphering payload...")    
    }

    if len(peid.key) != 0 {
      payload, err := ioutil.ReadFile("Payload")
      if err != nil {
        boldRed.Println("[!] ERROR: Can't open payload file.")
        clean()
        os.Exit(1)
      }
      progress()
      payload = xor(payload,peid.key)
      payload_xor, err2 := os.Create("Payload.xor")
      if err2 != nil {
        boldRed.Println("[!] ERROR: Can't create payload.xor file.")
        clean()
        os.Exit(1)      
      }
      progress()
      payload_key, err3 := os.Create("Payload.key")
      if err3 != nil {
        boldRed.Println("[!] ERROR: Can't create payload.xor file.")
        clean()
        os.Exit(1)      
      }
      payload_xor.Write(payload)
      payload_xor.Write(peid.key)

      payload_xor.Close()
      payload_key.Close()
      progress()

    }else{
      key := generateKey(peid.keySize)
      progress()
      payload, err := ioutil.ReadFile("Payload")
      if err != nil {
        boldRed.Println("[!] ERROR: Can't open payload file.")
        clean()
        os.Exit(1)
      }
      progress()
      payload = xor(payload,key)
      payload_xor, err2 := os.Create("Payload.xor")
      if err2 != nil {
        boldRed.Println("[!] ERROR: Can't create payload.xor file.")
        clean()
        os.Exit(1)      
      }
      progress()
      payload_key, err3 := os.Create("Payload.key")
      if err3 != nil {
        boldRed.Println("[!] ERROR: Can't create payload.xor file.")
        clean()
        os.Exit(1)      
      }
      payload_xor.Write(payload)
      payload_xor.Write(key)

      payload_xor.Close()
      payload_key.Close()
    }
    progress()  

    xxd := exec.Command("sh", "-c", "rm Payload && mv Payload.xor Payload && xxd -i Payload > Stub/PAYLOAD.h")
    xxd.Stdout = os.Stdout
    xxd.Stderr = os.Stderr
    xxd.Run()

    progress()  

    _xxd := exec.Command("sh", "-c", "xxd -i Payload.key > Stub/KEY.h")
    _xxd.Stdout = os.Stdout
    _xxd.Stderr = os.Stderr
    _xxd.Run()

    progress()  

    if peid.verbose == true {
      key, _ := exec.Command("sh", "-c", "xxd -i Payload.key").Output() 
      boldYellow.Println("[*] Payload ciphered with: ")
      boldBlue.Println(string(key))    
    } 
}



func xor(Data []byte, Key []byte) ([]byte){
  for i := 0; i < len(Data); i++{
    Data[i] = (Data[i] ^ (Key[(i%len(Key))]))
  }
  return Data
}


func generateKey(Size int) ([]byte){
  Key := make([]byte, Size)
  rand.Seed(time.Now().UTC().UnixNano())
  for i := 0; i < Size; i++{
    Key[i] = byte(rand.Intn(255))   
  }
  return Key
}

// Implement RC4...
//################################################### REQUIREMENTS ###################################################

func checkRequired() {

    CheckMingw, mingwErr := exec.Command("sh", "-c", "i686-w64-mingw32-g++-win32 --version").Output()
    if (!strings.Contains(string(CheckMingw), "Copyright")) {
      boldRed.Println("\n\n[!] ERROR: mingw is not installed.")
      red.Println(string(CheckMingw))
      red.Println(mingwErr)
      os.Exit(1)
    }
    progress()
    CheckNasm, _ := exec.Command("sh", "-c", "nasm -h").Output()
    if (!strings.Contains(string(CheckNasm), "usage:")) {
      boldRed.Println("\n\n[!] ERROR: nasm is not installed.")
      red.Println(string(CheckNasm))
      os.Exit(1)
    }
    progress()
    CheckStrip, _ := exec.Command("sh", "-c", "strip -V").Output()
    if (!strings.Contains(string(CheckStrip), "Copyright")) {
      boldRed.Println("\n\n[!] ERROR: strip is not installed.")
      red.Println(string(CheckStrip))
      os.Exit(1)
    }
    progress()
    CheckWine, _ := exec.Command("sh", "-c", "wine --help").Output()
    if (!strings.Contains(string(CheckWine), "Usage:")) {
      boldRed.Println("\n\n[!] ERROR: wine is not installed.")
      red.Println(string(CheckWine))
      os.Exit(1)
    }
    progress()
    CheckMapPE, _ := exec.Command("sh", "-c", "ls MapPE.exe").Output()
    if (!strings.Contains(string(CheckMapPE), "MapPE.exe")) {
      boldRed.Println("\n\n[!] ERROR: MapPE.exe is missing.")
      red.Println(string(CheckMapPE))
      red.Println(mingwErr)
      os.Exit(1)
    }
    progress()
  CheckXXD, _ := exec.Command("sh", "-c", "echo Amber|xxd").Output()
    if (!strings.Contains(string(CheckXXD), "Amber")) {
      boldRed.Println("\n\n[!] ERROR: xxd is not installed.")
      red.Println(string(CheckMingw))
      os.Exit(1)
    }
    progress()
    CheckMultiLib, _ := exec.Command("sh", "-c", "apt-cache policy gcc-multilib").Output()
    if (strings.Contains(string(CheckMultiLib), "(none)")) {
      boldRed.Println("\n\n[!] ERROR: gcc-multilib is not installed.")
      red.Println(string(CheckMultiLib))
      os.Exit(1)
    }
    progress()
   CheckMultiLibPlus, _ := exec.Command("sh", "-c", "apt-cache policy g++-multilib").Output()
    if (strings.Contains(string(CheckMultiLibPlus), "(none)")) {
      boldRed.Println("\n\n[!] ERROR: g++-multilib is not installed.")
      red.Println(string(CheckMultiLibPlus))
      os.Exit(1)
    }
    progress()

}


//################################################### PROGRESS ###################################################

func progress() {
  if peid.verbose == false {
    progressBar.Increment()
  }
}

func createBar() {
  
  var full int = 40

  if peid.verbose == false {
    if peid.staged == true {
      full -= 9
    }

    progressBar = pb.New(full)
    progressBar.SetWidth(80)
    progressBar.Start() 
  }
}


func clean() {

  exec.Command("sh", "-c", "rm ReplaceProcess/Mem.map").Run()
  progress()
  exec.Command("sh", "-c", "rm Stub.o").Run()
  progress()
  exec.Command("sh", "-c", "rm Payload").Run()
  progress()
  exec.Command("sh", "-c", "rm Payload.xor").Run()
  progress()
  exec.Command("sh", "-c", "rm Payload.key").Run()
  progress()
  exec.Command("sh", "-c", "echo   > Stub/PAYLOAD.h").Run()
  progress()
  exec.Command("sh", "-c", "echo   > Stub/KEY.h").Run()
  progress()
 
}



//################################################### GRAPHICS ###################################################
func Banner() {

  	var BANNER string = `

//   █████╗ ███╗   ███╗██████╗ ███████╗██████╗ 
//  ██╔══██╗████╗ ████║██╔══██╗██╔════╝██╔══██╗
//  ███████║██╔████╔██║██████╔╝█████╗  ██████╔╝
//  ██╔══██║██║╚██╔╝██║██╔══██╗██╔══╝  ██╔══██╗
//  ██║  ██║██║ ╚═╝ ██║██████╔╝███████╗██║  ██║
//  ╚═╝  ╚═╝╚═╝     ╚═╝╚═════╝ ╚══════╝╚═╝  ╚═╝
//  POC Crypter For ReplaceProcess                                             
`
  boldRed.Print(BANNER)
  boldBlue.Print("\n# Version: ")
  boldGreen.Println(VERSION)
  boldBlue.Print("# Source: ")
  boldGreen.Println("github.com/EgeBalci/Amber")
  
}
	
func Help() {
   var Help string = `

USAGE: 
  amber file.exe [options]


OPTIONS:
  
  -k, --key       [string]        Custom cipher key
  -ks,--keysize   <length>        Size of the encryption key in bytes (Max:100/Min:4)
  --staged                        Generated a staged payload
  --iat                           Use the import address table entries instead of hash_api
  --no-resource                   Don't add any resource
  -v, --verbose                   Verbose output mode
  -h, --help                      Show this massage

EXAMPLE:
  (Default settings if no option parameter passed)
  amber file.exe -ks 8
`
  color.Green(Help)

}