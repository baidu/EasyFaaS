/*
 * Copyright (c) 2020 Baidu, Inc.
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

// Package command
package command

import (
	"bufio"
	"bytes"
	"context"
	"fmt"
	"io"
	"io/ioutil"
	"net"
	"os"
	"os/exec"
	"runtime"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/rogpeppe/go-internal/testenv"
)

func helperCommandContext(t *testing.T, ctx context.Context, s ...string) (cmd *exec.Cmd) {
	testenv.MustHaveExec(t)

	cs := []string{"-test.run=TestHelperProcess", "--"}
	cs = append(cs, s...)
	if ctx != nil {
		cmd = exec.CommandContext(ctx, os.Args[0], cs...)
	} else {
		cmd = exec.Command(os.Args[0], cs...)
	}
	cmd.Env = []string{"GO_WANT_HELPER_PROCESS=1"}
	return cmd
}

func helperCommand(t *testing.T, s ...string) *exec.Cmd {
	return helperCommandContext(t, nil, s...)
}

func TestEcho(t *testing.T) {
	bs, err := helperCommand(t, "echo", "foo bar", "baz").Output()
	if err != nil {
		t.Errorf("echo: %v", err)
	}
	if g, e := string(bs), "foo bar baz\n"; g != e {
		t.Errorf("echo: want %q, got %q", e, g)
	}
}

func TestCatStdin(t *testing.T) {
	// Cat, testing stdin and stdout.
	input := "Input string\nLine 2"
	p := helperCommand(t, "cat")
	p.Stdin = strings.NewReader(input)
	bs, err := p.Output()
	if err != nil {
		t.Errorf("cat: %v", err)
	}
	s := string(bs)
	if s != input {
		t.Errorf("cat: want %q, got %q", input, s)
	}
}

func TestHelperProcess(*testing.T) {
	if os.Getenv("GO_WANT_HELPER_PROCESS") != "1" {
		return
	}
	defer os.Exit(0)

	// Determine which command to use to display open files.
	ofcmd := "lsof"
	switch runtime.GOOS {
	case "dragonfly", "freebsd", "netbsd", "openbsd":
		ofcmd = "fstat"
	case "plan9":
		ofcmd = "/bin/cat"
	case "aix":
		ofcmd = "procfiles"
	}

	args := os.Args
	for len(args) > 0 {
		if args[0] == "--" {
			args = args[1:]
			break
		}
		args = args[1:]
	}
	if len(args) == 0 {
		fmt.Fprintf(os.Stderr, "No command\n")
		os.Exit(2)
	}

	cmd, args := args[0], args[1:]
	switch cmd {
	case "echo":
		iargs := []interface{}{}
		for _, s := range args {
			iargs = append(iargs, s)
		}
		fmt.Println(iargs...)
	case "echoenv":
		for _, s := range args {
			fmt.Println(os.Getenv(s))
		}
		os.Exit(0)
	case "cat":
		if len(args) == 0 {
			io.Copy(os.Stdout, os.Stdin)
			return
		}
		exit := 0
		for _, fn := range args {
			f, err := os.Open(fn)
			if err != nil {
				fmt.Fprintf(os.Stderr, "Error: %v\n", err)
				exit = 2
			} else {
				defer f.Close()
				io.Copy(os.Stdout, f)
			}
		}
		os.Exit(exit)
	case "pipetest":
		bufr := bufio.NewReader(os.Stdin)
		for {
			line, _, err := bufr.ReadLine()
			if err == io.EOF {
				break
			} else if err != nil {
				os.Exit(1)
			}
			if bytes.HasPrefix(line, []byte("O:")) {
				os.Stdout.Write(line)
				os.Stdout.Write([]byte{'\n'})
			} else if bytes.HasPrefix(line, []byte("E:")) {
				os.Stderr.Write(line)
				os.Stderr.Write([]byte{'\n'})
			} else {
				os.Exit(1)
			}
		}
	case "read3": // read fd 3
		fd3 := os.NewFile(3, "fd3")
		bs, err := ioutil.ReadAll(fd3)
		if err != nil {
			fmt.Printf("ReadAll from fd 3: %v", err)
			os.Exit(1)
		}
		switch runtime.GOOS {
		case "dragonfly":
			// TODO(jsing): Determine why DragonFly is leaking
			// file descriptors...
		case "darwin":
			// TODO(bradfitz): broken? Sometimes.
			// https://golang.org/issue/2603
			// Skip this additional part of the test for now.
		case "netbsd":
			// TODO(jsing): This currently fails on NetBSD due to
			// the cloned file descriptors that result from opening
			// /dev/urandom.
			// https://golang.org/issue/3955
		case "illumos", "solaris":
			// TODO(aram): This fails on Solaris because libc opens
			// its own files, as it sees fit. Darwin does the same,
			// see: https://golang.org/issue/2603
		default:
			// Now verify that there are no other open fds.
			var files []*os.File
			for wantfd := basefds() + 1; wantfd <= 100; wantfd++ {
				// if poll.IsPollDescriptor(wantfd) {
				// 	continue
				// }
				f, err := os.Open(os.Args[0])
				if err != nil {
					fmt.Printf("error opening file with expected fd %d: %v", wantfd, err)
					os.Exit(1)
				}
				if got := f.Fd(); got != wantfd {
					fmt.Printf("leaked parent file. fd = %d; want %d\n", got, wantfd)
					var args []string
					switch runtime.GOOS {
					case "plan9":
						args = []string{fmt.Sprintf("/proc/%d/fd", os.Getpid())}
					case "aix":
						args = []string{fmt.Sprint(os.Getpid())}
					default:
						args = []string{"-p", fmt.Sprint(os.Getpid())}
					}
					out, _ := exec.Command(ofcmd, args...).CombinedOutput()
					fmt.Print(string(out))
					os.Exit(1)
				}
				files = append(files, f)
			}
			for _, f := range files {
				f.Close()
			}
		}
		// Referring to fd3 here ensures that it is not
		// garbage collected, and therefore closed, while
		// executing the wantfd loop above. It doesn't matter
		// what we do with fd3 as long as we refer to it;
		// closing it is the easy choice.
		fd3.Close()
		os.Stdout.Write(bs)
	case "exit":
		n, _ := strconv.Atoi(args[0])
		os.Exit(n)
	case "describefiles":
		f := os.NewFile(3, fmt.Sprintf("fd3"))
		ln, err := net.FileListener(f)
		if err == nil {
			fmt.Printf("fd3: listener %s\n", ln.Addr())
			ln.Close()
		}
		os.Exit(0)
	case "extraFilesAndPipes":
		n, _ := strconv.Atoi(args[0])
		pipes := make([]*os.File, n)
		for i := 0; i < n; i++ {
			pipes[i] = os.NewFile(uintptr(3+i), strconv.Itoa(i))
		}
		response := ""
		for i, r := range pipes {
			ch := make(chan string, 1)
			go func(c chan string) {
				buf := make([]byte, 10)
				n, err := r.Read(buf)
				if err != nil {
					fmt.Fprintf(os.Stderr, "Child: read error: %v on pipe %d\n", err, i)
					os.Exit(1)
				}
				c <- string(buf[:n])
				close(c)
			}(ch)
			select {
			case m := <-ch:
				response = response + m
			case <-time.After(5 * time.Second):
				fmt.Fprintf(os.Stderr, "Child: Timeout reading from pipe: %d\n", i)
				os.Exit(1)
			}
		}
		fmt.Fprintf(os.Stderr, "child: %s", response)
		os.Exit(0)
	case "exec":
		cmd := exec.Command(args[1])
		cmd.Dir = args[0]
		output, err := cmd.CombinedOutput()
		if err != nil {
			fmt.Fprintf(os.Stderr, "Child: %s %s", err, string(output))
			os.Exit(1)
		}
		fmt.Printf("%s", string(output))
		os.Exit(0)
	case "lookpath":
		p, err := exec.LookPath(args[0])
		if err != nil {
			fmt.Fprintf(os.Stderr, "LookPath failed: %v\n", err)
			os.Exit(1)
		}
		fmt.Print(p)
		os.Exit(0)
	case "stderrfail":
		fmt.Fprintf(os.Stderr, "some stderr text\n")
		os.Exit(1)
	case "sleep":
		time.Sleep(3 * time.Second)
		os.Exit(0)
	default:
		fmt.Fprintf(os.Stderr, "Unknown command %q\n", cmd)
		os.Exit(2)
	}
}

func basefds() uintptr {
	n := os.Stderr.Fd() + 1
	// The poll (epoll/kqueue) descriptor can be numerically
	// either between stderr and the testlog-fd, or after
	// testlog-fd.
	// if poll.IsPollDescriptor(n) {
	// 	n++
	// }
	for _, arg := range os.Args {
		if strings.HasPrefix(arg, "-test.testlogfile=") {
			n++
		}
	}
	return n
}
