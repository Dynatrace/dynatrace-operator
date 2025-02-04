package startup_probe

import (
	"fmt"
	"net"
	"os"
	"time"

	"github.com/pkg/errors"
	"github.com/spf13/cobra"
	"golang.org/x/net/context"
)

const (
	hostname       = "kubernetes.default.svc"
	use            = "startup-probe"
	defaultTimeout = 5
)

var (
	timeoutFlagValue = 5
)

type CommandBuilder struct {
}

func NewCommandBuilder() CommandBuilder {
	return CommandBuilder{}
}

func (builder CommandBuilder) Build() *cobra.Command {
	cmd := &cobra.Command{
		Use:  use,
		Long: "query DNS server about " + hostname,
		RunE: builder.buildRun(),
	}

	cmd.PersistentFlags().IntVar(&timeoutFlagValue, "timeout", defaultTimeout, "specify a different timeout [s]")

	cmd.SilenceUsage = true
	cmd.SilenceErrors = true

	return cmd
}

func (builder CommandBuilder) buildRun() func(*cobra.Command, []string) error {
	return func(cmd *cobra.Command, args []string) error {
		f := func(_ *cobra.Command, _ []string) error {
			ctx, cancel := context.WithTimeout(context.Background(), time.Duration(timeoutFlagValue)*time.Second)
			defer cancel()

			ips, err := net.DefaultResolver.LookupHost(ctx, hostname)
			if err != nil {
				return errors.WithMessagef(err, "DNS service not ready")
			}

			if len(ips) == 0 {
				return errors.Errorf("no DNS record found for %s", hostname)
			}

			return nil
		}
		if err := f(cmd, args); err != nil {
			fmt.Println(err) //nolint
			os.Exit(1)       //nolint
		}

		return nil
	}
}
