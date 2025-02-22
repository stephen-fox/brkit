package exprocess

import (
	"context"
	"fmt"
	"log"
	"net"
	"os"
	"os/exec"
	"strings"
)

// (\|) ._. (|/) <- mr. ferris was here

// SshPgrepCtxArgs configures the SshPgrepCtx function.
type SshPgrepCtxArgs struct {
	// AddrPort is the address of the SSH server to connect to
	// in the form of <host>:<port>.
	AddrPort string

	// ProcessExeName is the name of the process' executable
	// that should be monitored.
	ProcessExeName string
}

// SshPgrepCtx creates a context.Context that is marked as done
// when a process on a SSH server exits. It uses the ssh and
// pgrep programs to accomplish this.
func SshPgrepCtx(ctx context.Context, args SshPgrepCtxArgs) (context.Context, func(), error) {
	addr, port, err := net.SplitHostPort(args.AddrPort)
	if err != nil {
		return nil, nil, err
	}

	shellScript := fmt.Sprintf(`#!/bin/sh

while true
do
  sleep 1
  info="$(pgrep -a '%s')"
  [ -n "${info}" ] && break
done

if [ "$(echo "${info}" | wc -l)" -ne 1 ]
then
  echo "found multiple matches, plz exit one of them: ${info}"
fi

pid="$(echo ${info} | cut -f 1 -d ' ')"

echo "found pid: ${pid}"

while ps -p "${pid}" > /dev/null
do
  sleep 1
done

exit
`,
		args.ProcessExeName)

	newCtx, cancelFn := context.WithCancel(ctx)

	ssh := exec.CommandContext(newCtx, "ssh", "-p", port, addr)

	ssh.Stdin = strings.NewReader(shellScript)
	ssh.Stdout = os.Stderr
	ssh.Stderr = os.Stderr

	err = ssh.Start()
	if err != nil {
		cancelFn()

		return nil, nil, fmt.Errorf("failed to start ssh process - %w", err)
	}

	go func() {
		err = ssh.Wait()
		if err != nil {
			log.Println("ssh session exited with error -", err)
		}

		cancelFn()
	}()

	return newCtx, cancelFn, nil
}
