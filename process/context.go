package process

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/exec"
	"strings"
)

// (\|) ._. (|/) <- mr. ferris was here

type SshPgrepCtxArgs struct {
	Host            string
	ProgramFileName string
}

func SshPgrepCtx(ctx context.Context, args SshPgrepCtxArgs) (context.Context, func(), error) {
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
		args.ProgramFileName)

	newCtx, cancelFn := context.WithCancel(ctx)

	ssh := exec.CommandContext(newCtx,
		"ssh",
		args.Host)

	ssh.Stdin = strings.NewReader(shellScript)
	ssh.Stdout = os.Stderr
	ssh.Stderr = os.Stderr

	err := ssh.Start()
	if err != nil {
		cancelFn()
		return nil, nil, fmt.Errorf("failed to start ssh session - %w", err)
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
