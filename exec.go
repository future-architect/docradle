package docradle

import (
	"context"
	"github.com/gookit/color"
	"io"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"syscall"
	"time"

	"github.com/shirou/gopsutil/process"
	"golang.org/x/sync/errgroup"
)

var signals = make(chan os.Signal, 1)

func DumpCommand(stdout io.Writer, command string, args []string, dryRun bool) {
	color.Fprintln(stdout, "<bg=black;fg=lightBlue;op=reverse;>  Execute Command  </>\n")
	if dryRun {
		color.Fprintf(stdout, "<gray>(dry run)</> ")
	} else {
		color.Fprintf(stdout, "<gray>$</> ")
	}
	color.Fprintf(stdout, "<cyan>%s</>", command)
	for _, arg := range args {
		color.Fprintf(stdout, " <yellow>%s</>", arg)
	}
	if dryRun {
		io.WriteString(stdout, "\n")
	} else {
		io.WriteString(stdout, "\n\n")
	}
}

// Exec executes command
func Exec(stdout, stderr io.Writer, config *Config, command string, args []string, envvar *EnvVar) error {
	ctx, cancel := context.WithCancel(context.Background())

	stdoutLogger, err := NewLogger(ctx, StdOut, stdout, config.LogLevel, config.Stdout, envvar)
	if err != nil {
		return err
	}
	stderrLogger, err := NewLogger(ctx, StdErr, stderr, config.LogLevel, config.Stderr, envvar)
	if err != nil {
		return err
	}

	cmd := exec.CommandContext(ctx, command, args...)
	cmd.Env = envvar.EnvsForExec()

	// Setup signaling
	sigs := make(chan os.Signal, 1)
	signal.Notify(sigs, syscall.SIGHUP, syscall.SIGINT, syscall.SIGTERM, syscall.SIGKILL)

	eg, _ := errgroup.WithContext(ctx)

	stdoutPipe, err := cmd.StdoutPipe()
	if err != nil {
		return err
	}
	stderrPipe, err := cmd.StderrPipe()
	if err != nil {
		return err
	}
	stdoutLogger.StartOutput(eg, stdoutPipe)
	stderrLogger.StartOutput(eg, stderrPipe)

	eg.Go(func() error {
		select {
		case sig := <-sigs:
			signalProcessWithTimeout(cmd, sig, stderr)
			cancel()
		case <-ctx.Done():
			// exit when context is done
		}
		return nil
	})

	eg.Go(func() error {
		color.Fprintln(stdout, "<bg=black;fg=lightBlue;op=reverse;>  Start Execution  </>\n")

		defer stdoutLogger.Close()
		defer stderrLogger.Close()
		start := time.Now()
		err := cmd.Start()
		defer cancel()
		if err != nil {
			color.Fprintf(stderr, "<red>Error: %s</>\n\n", err.Error())
			return err
		}
		cwd, _ := filepath.Abs(".")
		stdoutLogger.WriteProcessStart(start, cmd.Process.Pid, cwd, command, args)
		proc, err := process.NewProcess(int32(cmd.Process.Pid))
		if err == nil {
			eg.Go(func() error {
				ticker := time.NewTicker(2 * time.Second)
				for {
					select {
					case <-ticker.C:
						mem, err := proc.MemoryInfo()
						if err != nil {
							return nil
						}
						mp, err := proc.MemoryPercentWithContext(ctx)
						if err != nil {
							return nil
						}
						cp, err := proc.CPUPercentWithContext(ctx)
						if err != nil {
							return nil
						}
						stdoutLogger.WriteMetrics(mem.RSS, mp, cp)
					case <-ctx.Done():
						return nil
					}
				}
			})
		}
		result := cmd.Wait()
		exit := time.Now()
		stdoutLogger.WriteProcessResult(exit, cmd.ProcessState.String(),
			exit.Sub(start), cmd.ProcessState.UserTime(), cmd.ProcessState.SystemTime())
		color.Fprintln(stdout, "\n<bg=black;fg=lightBlue;op=reverse;>  Process Result  </>\n")
		if cmd.ProcessState.Success() {
			color.Fprintf(stdout, "    <fg=lightGreen;op=underscore,bold;>%s</>\n", cmd.ProcessState.String())
		} else {
			color.Fprintf(stdout, "    <fg=red;op=underscore,bold;>%s</>\n", cmd.ProcessState.String())
		}
		if err != nil {
			color.Fprintf(stderr, "<red>Error: %s</>\n\n", err.Error())
		}
		return result
	})
	return eg.Wait()
}

func signalProcessWithTimeout(process *exec.Cmd, sig os.Signal, stderr io.Writer) {
	done := make(chan struct{})

	go func() {
		process.Process.Signal(sig) // pretty sure this doesn't do anything. It seems like the signal is automatically sent to the command?
		process.Wait()
		close(done)
	}()
	select {
	case <-done:
		return
	case <-time.After(10 * time.Second):
		color.Fprintf(stderr, "Killing command due to timeout")
		process.Process.Kill()
	}
}
