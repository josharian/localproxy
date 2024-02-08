package main

import (
	"bufio"
	"context"
	"flag"
	"fmt"
	"net"
	"os"
	"path/filepath"
	"strings"
	"time"

	"inet.af/tcpproxy"
)

func DialContext(ctx context.Context, network, address string) (net.Conn, error) {
	d := new(net.Dialer)
	for {
		conn, err := d.DialContext(ctx, network, address)
		if err == nil {
			return conn, nil
		}
		if ctx.Err() != nil {
			return nil, ctx.Err()
		}
		time.Sleep(30 * time.Millisecond)
	}
}

func main() {
	flag.Parse()
	err := Main()
	if err != nil {
		fmt.Fprintln(os.Stderr, err.Error())
		os.Exit(1)
	}
}

func Main() error {
	var configPath string
	switch flag.NArg() {
	case 0:
		homedir, err := os.UserHomeDir()
		if err != nil {
			return err
		}
		configPath = filepath.Join(homedir, ".config", "localproxy", "ports.txt")
	case 1:
		configPath = flag.Arg(0)
	default:
		fmt.Fprintf(os.Stderr, "usage: %s [path to ports.txt]", os.Args[0])
		os.Exit(2)
	}
	f, err := os.Open(configPath)
	if err != nil {
		return err
	}
	// ports.txt is a file with lines like ":6118 -> :6116" (mapping local port 6118 to local port 6116)
	// ignore lines that start with # or are empty
	mapping := make(map[string]string)
	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := scanner.Text()
		line = strings.TrimSpace(line)
		if len(line) == 0 || line[0] == '#' {
			continue
		}
		from, to, ok := strings.Cut(line, "->")
		if !ok {
			return fmt.Errorf("invalid line in %s: %s", configPath, line)
		}
		from = strings.TrimSpace(from)
		to = strings.TrimSpace(to)
		mapping[from] = to
	}
	err = scanner.Err()
	if err != nil {
		return err
	}
	err = f.Close()
	if err != nil {
		return err
	}

	var p tcpproxy.Proxy
	for from, to := range mapping {
		p.AddRoute(from, &tcpproxy.DialProxy{
			Addr:        to,
			DialContext: DialContext,
		})
		fmt.Printf("proxying %s -> %s\n", from, to)
	}

	return p.Run()
}
